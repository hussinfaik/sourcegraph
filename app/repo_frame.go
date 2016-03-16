package app

import (
	"errors"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"

	gcontext "github.com/gorilla/context"
	"github.com/sourcegraph/mux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"sourcegraph.com/sourcegraph/sourcegraph/app/appconf"
	"sourcegraph.com/sourcegraph/sourcegraph/app/internal/tmpl"
	"sourcegraph.com/sourcegraph/sourcegraph/errcode"
	"sourcegraph.com/sourcegraph/sourcegraph/go-sourcegraph/sourcegraph"
	"sourcegraph.com/sourcegraph/sourcegraph/platform"
	"sourcegraph.com/sourcegraph/sourcegraph/platform/pctx"
	"sourcegraph.com/sourcegraph/sourcegraph/util/handlerutil"
	"sourcegraph.com/sourcegraph/sourcegraph/util/httputil/httpctx"
)

// Until we have a more advanced repo config scheme, always enable
// these in a hard-coded fashion.
func isAlwaysEnabledApp(app string) bool {
	return app == "changes"
}

// orderedRepoEnabledFrames returns apps that are enabled for the given repo. Key of frames map is the app ID.
// It also returns a slice of app IDs that defines the order in which they should be displayed.
func orderedRepoEnabledFrames(repo *sourcegraph.Repo, repoConf *sourcegraph.RepoConfig) (frames map[string]platform.RepoFrame, orderedIDs []string) {
	if appconf.Flags.DisableApps {
		return nil, nil
	}

	frames = make(map[string]platform.RepoFrame)
	for _, frame := range platform.Frames() {
		if isAlwaysEnabledApp(frame.ID) || repoConf.IsAppEnabled(frame.ID) || (frame.Enable != nil && frame.Enable(repo)) {
			frames[frame.ID] = frame
			orderedIDs = append(orderedIDs, frame.ID)
		}
	}

	// TODO: Instead of prioritizing specific apps, determine the sort order
	// automatically. If little or no ranking data is present, rank alphabetically
	// and then rank based on "times all users went to this app in the repo" so
	// that the most-used app for a given repo comes first.

	// First and foremost, sort the app names alphabetically.
	sort.Strings(orderedIDs)

	// Second, enforce that Changes is the first.
	for i, appID := range orderedIDs {
		switch appID {
		case "changes":
			orderedIDs[0], orderedIDs[i] = orderedIDs[i], orderedIDs[0]
		}
	}

	return frames, orderedIDs
}

func serveRepoFrame(w http.ResponseWriter, r *http.Request) error {
	ctx, _ := handlerutil.Client(r)
	rc, vc, err := handlerutil.GetRepoAndRevCommon(ctx, mux.Vars(r))
	if err != nil {
		return err
	}

	appID := mux.Vars(r)["App"]
	frames, _ := orderedRepoEnabledFrames(rc.Repo, rc.RepoConfig)
	app, ok := frames[appID]
	if !ok {
		return &errcode.HTTPErr{Status: http.StatusNotFound, Err: errors.New("not a valid app")}
	}

	if vc.RepoCommit == nil {
		return renderRepoNoVCSDataTemplate(w, r, rc)
	}

	// TODO(beyang): think of more robust way of isolating apps to
	// prevent shared mutable state (e.g., modifying http.Requests) to
	// prevent inter-app interference
	rCopy := copyRequest(r)

	framectx, err := pctx.WithRepoFrameInfo(ctx, r)
	if err != nil {
		return err
	}
	httpctx.SetForRequest(rCopy, framectx)
	defer gcontext.Clear(rCopy) // clear the app context after finished to avoid a memory leak

	rr := httptest.NewRecorder()

	stripPrefix := pctx.BaseURI(framectx)
	if u, err := url.Parse(stripPrefix); err == nil {
		stripPrefix = u.Path
	} else {
		return err
	}

	platform.SetPlatformRequestURL(framectx, w, r, rCopy)

	app.Handler.ServeHTTP(rr, rCopy)

	// extract response body (purposefully ignoring headers)
	body := string(rr.Body.Bytes())

	// If Sourcegraph-Verbatim header was set to true, or this is a redirect,
	// relay this request to browser directly, and copy appropriate headers.
	redirect := rr.Code == http.StatusSeeOther || rr.Code == http.StatusMovedPermanently || rr.Code == http.StatusTemporaryRedirect || rr.Code == http.StatusFound
	if rr.Header().Get(platform.HTTPHeaderVerbatim) == "true" || redirect {
		copyHeader(w.Header(), rr.Header())
		w.WriteHeader(rr.Code)
		_, err := io.Copy(w, rr.Body)
		return err
	}

	var appHTML template.HTML
	var appError error
	if rr.Code == http.StatusOK {
		appHTML = template.HTML(body)
	} else if rr.Code == http.StatusUnauthorized && nil == handlerutil.UserFromContext(ctx) {
		// App returned Unauthorized, and user's not logged in. So redirect to login page and try again.
		return grpc.Errorf(codes.Unauthenticated, "platform app returned unauthorized and no authenticated user in current context")
	} else {
		appError = errors.New(body)
		if !handlerutil.DebugMode(r) {
			appError = errPlatformAppPublicFacingFatalError
		}
	}
	appSubtitle := rr.Header().Get(platform.HTTPHeaderTitle)

	return tmpl.Exec(r, w, "repo/frame.html", http.StatusOK, nil, &struct {
		handlerutil.RepoCommon
		handlerutil.RepoRevCommon

		AppSubtitle string
		AppTitle    string
		AppHTML     template.HTML
		AppError    error

		RobotsIndex bool
		tmpl.Common
	}{
		RepoCommon:    *rc,
		RepoRevCommon: *vc,

		AppSubtitle: appSubtitle,
		AppTitle:    app.Title,
		AppHTML:     appHTML,
		AppError:    appError,

		RobotsIndex: true,
	})
}

// errPlatformAppPublicFacingFatalError is the public facing error message to display for platform app
// fatal errors when not in debug mode (to hide potentially sensitive information in original error message).
var errPlatformAppPublicFacingFatalError = errors.New(`Sorry, there’s been a problem with this app.`)

// copyHeader copies whitelisted headers.
//
// TODO: Eventually, we should copy all headers minus hop-by-hop ones.
//       We didn't want to do that right away in order to build better understanding and motivation
//       for copying more headers than are needed, by by now it's becoming clear that's the way to go.
func copyHeader(dst, src http.Header) {
	// Since we're accessing the map directly, the header values must match canonicalized versions exactly.
	dst["Content-Encoding"] = src["Content-Encoding"]
	dst["Content-Type"] = src["Content-Type"]
	dst["Location"] = src["Location"]
	dst["Last-Modified"] = src["Last-Modified"]
	dst["Etag"] = src["Etag"]
}
