package app_test

import (
	"net/http"
	"net/url"
	"testing"

	"sourcegraph.com/sourcegraph/sourcegraph/pkg/vcs"

	"golang.org/x/net/context"

	"strings"

	"sourcegraph.com/sourcegraph/sourcegraph/app/internal/apptest"
	"sourcegraph.com/sourcegraph/sourcegraph/app/router"
	"sourcegraph.com/sourcegraph/sourcegraph/go-sourcegraph/sourcegraph"
)

func TestRepoBuild(t *testing.T) {
	c, mock := apptest.New()

	calledGet := mockRepoGet(mock, "my/repo")
	calledGetConfig := mockEmptyRepoConfig(mock)
	calledGetCommit := mock.Repos.MockGetCommit_ByID_NoCheck(t, fakeCommitID)
	calledBuildsGet := mock.Builds.MockGet_Return(t,
		&sourcegraph.Build{ID: 1, Repo: "my/repo", CommitID: strings.Repeat("a", 40)},
	)

	if _, err := c.GetOK(router.Rel.URLToRepoBuild("my/repo", 1).String()); err != nil {
		t.Fatal(err)
	}
	if !*calledGet {
		t.Error("!calledGet")
	}
	if !*calledGetConfig {
		t.Error("!calledGetConfig")
	}
	if !*calledGetCommit {
		t.Error("!calledGetCommit")
	}
	if !*calledBuildsGet {
		t.Error("!calledBuildsGet")
	}
}

func TestRepoBuilds(t *testing.T) {
	c, mock := apptest.New()

	calledGet := mockRepoGet(mock, "my/repo")
	calledGetConfig := mockEmptyRepoConfig(mock)
	calledBuildsList := mock.Builds.MockList(t,
		&sourcegraph.Build{ID: 1, Repo: "my/repo", CommitID: strings.Repeat("a", 40)},
	)
	calledGetCommit := mockRepoCommit(mock, &vcs.Commit{})

	if _, err := c.GetOK(router.Rel.URLToRepoSubroute(router.RepoBuilds, "my/repo").String()); err != nil {
		t.Fatal(err)
	}
	if !*calledGet {
		t.Error("!calledGet")
	}
	if !*calledGetConfig {
		t.Error("!calledGetConfig")
	}
	if !*calledBuildsList {
		t.Error("!calledBuildsList")
	}
	if !*calledGetCommit {
		t.Error("!calledGetCommit")
	}
}

func TestRepoBuildsCreate(t *testing.T) {
	c, mock := apptest.New()

	calledGet := mockRepoGet(mock, "my/repo")
	calledGetConfig := mockEmptyRepoConfig(mock)
	var calledBuildsCreate bool
	mock.Builds.Create_ = func(ctx context.Context, op *sourcegraph.BuildsCreateOp) (*sourcegraph.Build, error) {
		if want := "c"; op.CommitID != want {
			t.Errorf("got CommitID == %q, want %q", op.CommitID, want)
		}
		calledBuildsCreate = true
		return &sourcegraph.Build{ID: 1, CommitID: strings.Repeat("a", 40), Repo: "my/repo"}, nil
	}

	q := url.Values{"CommitID": []string{"c"}}
	req, _ := http.NewRequest("POST", router.Rel.URLToRepoSubroute(router.RepoBuildsCreate, "my/repo").String(), strings.NewReader(q.Encode()))
	req.Header.Set("content-type", "application/x-www-form-urlencoded")

	resp, err := c.DoNoFollowRedirects(req)
	if err != nil {
		t.Fatal(err)
	}
	if want := http.StatusSeeOther; resp.StatusCode != want {
		t.Errorf("got status %d, want %d", resp.StatusCode, want)
	}

	if want, got := router.Rel.URLToRepoBuild("my/repo", 1).String(), resp.Header.Get("location"); got != want {
		t.Errorf("got Location %q, want %q", got, want)
	}
	if !*calledGet {
		t.Error("!calledGet")
	}
	if !*calledGetConfig {
		t.Error("!calledGetConfig")
	}
	if !calledBuildsCreate {
		t.Error("!calledBuildsCreate")
	}
}
