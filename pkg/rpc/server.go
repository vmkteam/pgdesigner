// Package rpc provides the JSON-RPC 2.0 API for PgDesigner.
package rpc

import (
	"net/http"

	"github.com/vmkteam/pgdesigner/pkg/designer/store"
	"github.com/vmkteam/pgdesigner/pkg/pgd"
	"github.com/vmkteam/zenrpc/v2"
)

//go:generate go tool zenrpc

var (
	ErrInternal = zenrpc.NewStringError(http.StatusInternalServerError, "internal error")
)

// New returns a new zenrpc Server with ProjectService (read-only).
func New(project *pgd.Project, quitCh chan struct{}) *zenrpc.Server {
	return newServer(NewProjectService(project), quitCh)
}

// NewWithStore returns a new zenrpc Server with ProjectService backed by a ProjectStore.
func NewWithStore(s *store.ProjectStore, quitCh chan struct{}) *zenrpc.Server {
	return newServer(NewProjectServiceWithStore(s), quitCh)
}

func newServer(ps *ProjectService, quitCh chan struct{}) *zenrpc.Server {
	srv := zenrpc.NewServer(zenrpc.Options{
		ExposeSMD: true,
		AllowCORS: true,
	})
	srv.RegisterAll(map[string]zenrpc.Invoker{
		"project": ps,
		"app":     NewAppService(quitCh),
	})
	return srv
}
