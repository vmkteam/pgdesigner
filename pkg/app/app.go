// Package app provides the HTTP server for PgDesigner.
package app

import (
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strconv"

	"github.com/vmkteam/pgdesigner/pkg/designer/store"
	"github.com/vmkteam/pgdesigner/pkg/pgd"
	"github.com/vmkteam/pgdesigner/pkg/rpc"
	"github.com/vmkteam/rpcgen/v2"
	"github.com/vmkteam/zenrpc/v2"
)

// App is the PgDesigner HTTP application.
type App struct {
	store      *store.ProjectStore
	cfg        *Config
	rpcSrv     *zenrpc.Server
	listener   net.Listener
	frontendFS fs.FS
	quitCh     chan struct{}
}

// AppOption configures App creation.
type AppOption func(*appOptions)
type appOptions struct {
	readOnly bool
	version  string
}

// WithReadOnly disables all write RPC methods.
func WithReadOnly() AppOption {
	return func(o *appOptions) { o.readOnly = true }
}

// WithVersion sets the application version for About().
func WithVersion(v string) AppOption {
	return func(o *appOptions) { o.version = v }
}

// New creates a new App with the given project.
func New(project *pgd.Project) *App {
	return NewWithStore(store.NewProjectStore(project, ""))
}

// NewWithStore creates a new App with the given store. Options: WithReadOnly.
func NewWithStore(s *store.ProjectStore, opts ...AppOption) *App {
	var o appOptions
	for _, opt := range opts {
		opt(&o)
	}

	cfg, err := Load()
	if err != nil {
		log.Printf("warning: failed to load config: %v", err)
		cfg = &Config{}
	}

	quitCh := make(chan struct{})
	return &App{
		store: s,
		cfg:   cfg,
		rpcSrv: rpc.NewWithStore(rpc.ServerOptions{
			Store:          s,
			QuitCh:         quitCh,
			IsRegisteredFn: cfg.IsRegistered,
			ReadOnly:       o.readOnly,
			Version:        o.version,
			Config: rpc.ConfigCallbacks{
				Register: func(email string) error {
					cfg.RegisteredEmail = email
					return cfg.Save()
				},
				IsRegistered:   cfg.IsRegistered,
				GetRecentFiles: func() []string { return cfg.RecentFiles },
				AddRecentFile: func(path string) error {
					cfg.AddRecentFile(path)
					return cfg.Save()
				},
				RemoveRecentFile: func(path string) error {
					cfg.RemoveRecentFile(path)
					return cfg.Save()
				},
			},
		}),
		quitCh: quitCh,
	}
}

// Config returns the application config.
func (a *App) Config() *Config { return a.cfg }

// Store returns the project store.
func (a *App) Store() *store.ProjectStore { return a.store }

// QuitCh returns a channel that is closed when the browser requests shutdown.
func (a *App) QuitCh() <-chan struct{} {
	return a.quitCh
}

// Run starts the HTTP server on the given port (0 = random).
// It registers routes and returns the listen address.
func (a *App) Run(port int) (string, error) {
	listenAddr := "127.0.0.1:" + strconv.Itoa(port)

	var err error
	a.listener, err = net.Listen("tcp", listenAddr)
	if err != nil {
		return "", fmt.Errorf("listen %s: %w", listenAddr, err)
	}

	mux := http.NewServeMux()
	a.registerHandlers(mux)

	go func() { _ = http.Serve(a.listener, mux) }()

	return "http://" + a.listener.Addr().String(), nil
}

// Addr returns the listen address.
func (a *App) Addr() string {
	if a.listener == nil {
		return ""
	}
	return a.listener.Addr().String()
}

// SetFrontend sets the embedded frontend filesystem to serve at /.
func (a *App) SetFrontend(distFS fs.FS) {
	a.frontendFS = distFS
}

// TypeScriptClient generates the TypeScript RPC client from SMD.
func (a *App) TypeScriptClient() ([]byte, error) {
	gen := rpcgen.FromSMD(a.rpcSrv.SMD())
	return gen.TSClient(nil).Generate()
}

// Close stops the HTTP server.
func (a *App) Close() error {
	if a.listener != nil {
		return a.listener.Close()
	}
	return nil
}
