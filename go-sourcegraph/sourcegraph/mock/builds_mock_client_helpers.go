// GENERATED CODE - DO NOT EDIT!
// @generated
//
// Generated by:
//
//   go run gen_client_helpers.go
//
// Called via:
//
//   go generate
//

package mock

import (
	"testing"

	"golang.org/x/net/context"

	"sourcegraph.com/sourcegraph/sourcegraph/go-sourcegraph/sourcegraph"
)

func (s *BuildsClient) MockGet_Return(t *testing.T, want *sourcegraph.Build) (called *bool) {
	called = new(bool)
	s.Get_ = func(ctx context.Context, op *sourcegraph.BuildSpec) (*sourcegraph.Build, error) {
		*called = true
		return want, nil
	}
	return
}

func (s *BuildsClient) MockGetRepoBuild(t *testing.T, build *sourcegraph.Build) (called *bool) {
	called = new(bool)
	s.GetRepoBuild_ = func(ctx context.Context, rev *sourcegraph.RepoRevSpec) (*sourcegraph.Build, error) {
		*called = true
		return build, nil
	}
	return
}

func (s *BuildsClient) MockList(t *testing.T, want ...*sourcegraph.Build) (called *bool) {
	called = new(bool)
	s.List_ = func(ctx context.Context, op *sourcegraph.BuildListOptions) (*sourcegraph.BuildList, error) {
		*called = true
		return &sourcegraph.BuildList{Builds: want}, nil
	}
	return
}

func (s *BuildsClient) MockListBuildTasks(t *testing.T, want ...*sourcegraph.BuildTask) (called *bool) {
	called = new(bool)
	s.ListBuildTasks_ = func(ctx context.Context, op *sourcegraph.BuildsListBuildTasksOp) (*sourcegraph.BuildTaskList, error) {
		*called = true
		return &sourcegraph.BuildTaskList{BuildTasks: want}, nil
	}
	return
}
