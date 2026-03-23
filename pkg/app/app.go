// Package app provides the HTTP server for PgDesigner.
package app

import (
	"fmt"
	"io/fs"
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
	rpcSrv     *zenrpc.Server
	listener   net.Listener
	frontendFS fs.FS
	quitCh     chan struct{}
}

// New creates a new App with the given project.
func New(project *pgd.Project) *App {
	return NewWithStore(store.NewProjectStore(project, ""))
}

// NewWithStore creates a new App with the given store.
func NewWithStore(s *store.ProjectStore) *App {
	quitCh := make(chan struct{})
	return &App{
		store:  s,
		rpcSrv: rpc.NewWithStore(s, quitCh),
		quitCh: quitCh,
	}
}

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
