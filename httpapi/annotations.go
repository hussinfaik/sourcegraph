package httpapi

import (
	"net/http"

	"sourcegraph.com/sourcegraph/sourcegraph/go-sourcegraph/sourcegraph"
	"sourcegraph.com/sourcegraph/sourcegraph/util/handlerutil"
)

func serveAnnotations(w http.ResponseWriter, r *http.Request) error {
	ctx, cl := handlerutil.Client(r)

	var opt sourcegraph.AnnotationsListOptions
	if err := schemaDecoder.Decode(&opt, r.URL.Query()); err != nil {
		return err
	}

	if err := handlerutil.ResolveRepoRev(r, &opt.Entry.RepoRev); err != nil {
		return err
	}

	anns, err := cl.Annotations.List(ctx, &opt)
	if err != nil {
		return err
	}

	return writeJSON(w, anns)
}
