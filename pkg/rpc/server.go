// Package rpc provides the JSON-RPC 2.0 API for PgDesigner.
package rpc

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/vmkteam/pgdesigner/pkg/designer/store"
	"github.com/vmkteam/pgdesigner/pkg/pgd"
	"github.com/vmkteam/zenrpc/v2"
)

//go:generate go tool zenrpc

var (
	ErrInternal = zenrpc.NewStringError(http.StatusInternalServerError, "internal error")
)

// blockedMethods lists RPC methods blocked in read-only (public demo) mode.
// Includes: all write methods + dangerous read methods (file access, quit, state mutation).
// Names are without namespace prefix because zenrpc passes only the method part to middleware.
var blockedMethods = map[string]bool{
	// Project write methods
	RPC.ProjectService.SaveProject:           true,
	RPC.ProjectService.SaveProjectAs:         true,
	RPC.ProjectService.SaveLayout:            true,
	RPC.ProjectService.SetAutoSave:           true,
	RPC.ProjectService.UpdateTable:           true,
	RPC.ProjectService.CreateTable:           true,
	RPC.ProjectService.DeleteTable:           true,
	RPC.ProjectService.CreateSchema:          true,
	RPC.ProjectService.DeleteSchema:          true,
	RPC.ProjectService.MoveTable:             true,
	RPC.ProjectService.FixLintIssues:         true,
	RPC.ProjectService.IgnoreLintRules:       true,
	RPC.ProjectService.UnignoreLintRules:     true,
	RPC.ProjectService.UpdateProjectSettings: true,
	// App methods dangerous for public demo
	RPC.AppService.Quit:           true, // kills server
	RPC.AppService.OpenFile:       true, // arbitrary file read + DB connect
	RPC.AppService.Register:       true, // writes to disk
	RPC.AppService.GetRecentFiles:     true, // leaks server paths
	RPC.AppService.GetRecentFilesInfo: true, // leaks server paths
	RPC.AppService.RemoveRecentFile:   true, // writes to disk
	RPC.AppService.ListDirectory:      true, // leaks server filesystem
	RPC.AppService.GetHomePath:        true, // leaks server filesystem
	RPC.AppService.OpenDemo:           true, // shared state mutation
	RPC.AppService.NewProject:         true, // shared state mutation
	RPC.AppService.CloseProject:       true, // shared state mutation
}

// readOnlyMiddleware blocks write methods in read-only mode.
func readOnlyMiddleware(next zenrpc.InvokeFunc) zenrpc.InvokeFunc {
	return func(ctx context.Context, method string, params json.RawMessage) zenrpc.Response {
		if blockedMethods[method] {
			return zenrpc.NewResponseError(zenrpc.IDFromContext(ctx), http.StatusForbidden, "read-only mode: editing is disabled", nil)
		}
		return next(ctx, method, params)
	}
}

// ServerOptions configures the RPC server.
type ServerOptions struct {
	Store          *store.ProjectStore
	QuitCh         chan struct{}
	IsRegisteredFn func() bool
	Config         ConfigCallbacks
	ReadOnly       bool
	Version        string
}

// New returns a new zenrpc Server with ProjectService (read-only).
func New(project *pgd.Project, quitCh chan struct{}) *zenrpc.Server {
	ps := NewProjectService(project)
	srv := zenrpc.NewServer(zenrpc.Options{
		ExposeSMD: true,
		AllowCORS: true,
	})
	srv.RegisterAll(map[string]zenrpc.Invoker{
		"project": ps,
		"app":     NewAppService(quitCh, nil, ConfigCallbacks{}, "dev"),
	})
	return srv
}

// NewWithStore returns a new zenrpc Server with ProjectService backed by a ProjectStore.
func NewWithStore(opts ServerOptions) *zenrpc.Server {
	ps := NewProjectServiceWithStore(opts.Store, opts.IsRegisteredFn, opts.Config.AddRecentFile)
	srv := zenrpc.NewServer(zenrpc.Options{
		ExposeSMD: true,
		AllowCORS: true,
	})
	if opts.ReadOnly {
		srv.Use(readOnlyMiddleware)
	}
	srv.RegisterAll(map[string]zenrpc.Invoker{
		"project": ps,
		"app":     NewAppService(opts.QuitCh, opts.Store, opts.Config, opts.Version),
	})
	return srv
}
