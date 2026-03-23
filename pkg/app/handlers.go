package app

import (
	"net/http"

	"github.com/vmkteam/zenrpc/v2"
)

func (a *App) registerHandlers(mux *http.ServeMux) {
	// CORS for dev mode (Vite dev server on different port)
	corsHandler := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	// JSON-RPC endpoint
	mux.Handle("/rpc/", corsHandler(a.rpcSrv))

	// SMDBox documentation
	mux.HandleFunc("/rpc/doc/", zenrpc.SMDBoxHandler)

	// Frontend (embedded dist/ or fallback)
	if a.frontendFS != nil {
		mux.Handle("/", http.FileServerFS(a.frontendFS))
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<!DOCTYPE html><html><body><h3>PgDesigner</h3><p>Frontend not embedded. Use <code>pnpm dev</code> in frontend/ or build with <code>pnpm build</code>.</p><p><a href="/rpc/doc/">RPC Documentation (SMDBox)</a></p></body></html>`))
		})
	}
}
