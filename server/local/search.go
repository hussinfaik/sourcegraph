package local

import (
	"golang.org/x/net/context"
	"sourcegraph.com/sourcegraph/sourcegraph/go-sourcegraph/sourcegraph"
	"sourcegraph.com/sourcegraph/sourcegraph/pkg/vcs"
	"sourcegraph.com/sourcegraph/sourcegraph/svc"
	"sourcegraph.com/sourcegraph/sourcegraph/util/eventsutil"
)

var Search sourcegraph.SearchServer = &search{}

type search struct{}

var _ sourcegraph.SearchServer = (*search)(nil)

func (s *search) SearchTokens(ctx context.Context, opt *sourcegraph.TokenSearchOptions) (*sourcegraph.DefList, error) {
	defListOpts := &sourcegraph.DefListOptions{
		Query:       opt.Query,
		RepoRevs:    []string{opt.RepoRev.URI + "@" + opt.RepoRev.CommitID},
		ListOptions: opt.ListOptions,
		Nonlocal:    true,
		Doc:         true,
	}

	defList, err := svc.Defs(ctx).List(ctx, defListOpts)
	if err != nil {
		return nil, err
	}

	eventsutil.LogSearchQuery(ctx, "TokenSearch", defList.Total)

	return defList, nil
}

func (s *search) SearchText(ctx context.Context, opt *sourcegraph.TextSearchOptions) (*sourcegraph.VCSSearchResultList, error) {
	vcsSearchOpts := &sourcegraph.RepoTreeSearchOptions{
		SearchOptions: vcs.SearchOptions{
			Query:        opt.Query,
			QueryType:    "fixed",
			ContextLines: 1,
			N:            opt.ListOptions.PerPage,
			Offset:       (opt.ListOptions.Page - 1) * opt.ListOptions.PerPage,
		},
	}

	results, err := svc.RepoTree(ctx).Search(ctx, &sourcegraph.RepoTreeSearchOp{Rev: opt.RepoRev, Opt: vcsSearchOpts})
	if err != nil {
		return nil, err
	}

	eventsutil.LogSearchQuery(ctx, "TextSearch", results.Total)

	return results, nil
}
