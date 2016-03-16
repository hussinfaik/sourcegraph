// generated by gen-mocks; DO NOT EDIT

package mockstore

import (
	"golang.org/x/net/context"
	"sourcegraph.com/sourcegraph/sourcegraph/go-sourcegraph/sourcegraph"
	"sourcegraph.com/sourcegraph/sourcegraph/store"
)

type RegisteredClients struct {
	Get_              func(v0 context.Context, v1 sourcegraph.RegisteredClientSpec) (*sourcegraph.RegisteredClient, error)
	GetByCredentials_ func(v0 context.Context, v1 sourcegraph.RegisteredClientCredentials) (*sourcegraph.RegisteredClient, error)
	Create_           func(v0 context.Context, v1 sourcegraph.RegisteredClient) error
	Update_           func(v0 context.Context, v1 sourcegraph.RegisteredClient) error
	Delete_           func(v0 context.Context, v1 sourcegraph.RegisteredClientSpec) error
	List_             func(v0 context.Context, v1 sourcegraph.RegisteredClientListOptions) (*sourcegraph.RegisteredClientList, error)
}

func (s *RegisteredClients) Get(v0 context.Context, v1 sourcegraph.RegisteredClientSpec) (*sourcegraph.RegisteredClient, error) {
	return s.Get_(v0, v1)
}

func (s *RegisteredClients) GetByCredentials(v0 context.Context, v1 sourcegraph.RegisteredClientCredentials) (*sourcegraph.RegisteredClient, error) {
	return s.GetByCredentials_(v0, v1)
}

func (s *RegisteredClients) Create(v0 context.Context, v1 sourcegraph.RegisteredClient) error {
	return s.Create_(v0, v1)
}

func (s *RegisteredClients) Update(v0 context.Context, v1 sourcegraph.RegisteredClient) error {
	return s.Update_(v0, v1)
}

func (s *RegisteredClients) Delete(v0 context.Context, v1 sourcegraph.RegisteredClientSpec) error {
	return s.Delete_(v0, v1)
}

func (s *RegisteredClients) List(v0 context.Context, v1 sourcegraph.RegisteredClientListOptions) (*sourcegraph.RegisteredClientList, error) {
	return s.List_(v0, v1)
}

var _ store.RegisteredClients = (*RegisteredClients)(nil)
