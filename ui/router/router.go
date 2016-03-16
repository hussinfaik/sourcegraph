// Package router is a URL router for the app UI handlers.
package router

import (
	"github.com/sourcegraph/mux"
	app_router "sourcegraph.com/sourcegraph/sourcegraph/app/router"
	"sourcegraph.com/sourcegraph/sourcegraph/go-sourcegraph/routevar"
)

const (
	RepoTree = "repo.tree"

	Definition  = "def"
	DefExamples = "def.examples"

	RepoCommits = "repo.commits"

	SearchTokens = "search.tokens"
	SearchText   = "search.text"

	AppdashUploadPageLoad = "appdash.upload-page-load"

	UserContentUpload = "usercontent.upload"

	UserInviteBulk = "user.invite.bulk"
)

func New(base *mux.Router) *mux.Router {
	if base == nil {
		base = mux.NewRouter()
	}

	base.StrictSlash(true)

	repoRevPath := `/` + routevar.RepoRev
	repoRev := base.PathPrefix(repoRevPath).
		PostMatchFunc(routevar.FixRepoRevVars).
		BuildVarsFunc(routevar.PrepareRepoRevRouteVars).
		Subrouter()

	repoRev.Path("/.tree" + routevar.TreeEntryPath).
		Methods("GET").
		PostMatchFunc(routevar.FixTreeEntryVars).
		BuildVarsFunc(routevar.PrepareTreeEntryRouteVars).
		Name(RepoTree)

	defPath := "/" + routevar.Def

	repoRev.Path(defPath).
		Methods("GET").
		PostMatchFunc(routevar.FixDefUnitVars).
		BuildVarsFunc(routevar.PrepareDefRouteVars).
		Name(Definition)

	def := repoRev.PathPrefix(defPath).
		PostMatchFunc(routevar.FixDefUnitVars).
		BuildVarsFunc(routevar.PrepareDefRouteVars).
		Subrouter()

	def.Path("/.examples").
		Methods("GET").
		Name(DefExamples)

	repoRev.Path("/.search/tokens").
		Methods("GET").
		Name(SearchTokens)

	repoRev.Path("/.search/text").
		Methods("GET").
		Name(SearchText)

	repo := base.PathPrefix(`/` + routevar.Repo).Subrouter()

	repo.Path("/.commits").
		Methods("GET").
		Name(RepoCommits)

	base.Path("/.appdash/upload-page-load").
		Methods("POST").
		Name(AppdashUploadPageLoad)

	base.Path("/.usercontent").
		Methods("POST").
		Name(UserContentUpload)

	base.Path("/.invite-bulk").
		Methods("POST").
		Name(UserInviteBulk)

	return base
}

// Rel is a relative url router, used for tests.
var Rel = app_router.Router{Router: *New(nil)}
