// Package server is the HTTP API and static front end. It is meant to be
// reached from another machine, so every page and API call needs the random
// token printed at start-up.
package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"

	"scanorigin/internal/dataset"
	"scanorigin/internal/version"
	"scanorigin/web"
)

// Server serves the UI and API for one dataset folder.
type Server struct {
	root  string
	token string
}

// New creates a server for the dataset folder root with a fresh random token.
func New(root string) *Server {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return &Server{root: root, token: hex.EncodeToString(b)}
}

// Token is the secret required by the page and /api/*.
func (s *Server) Token() string { return s.token }

// Handler builds the route table.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.handleIndex)
	// Static assets carry no secret; only the index page embeds the token.
	mux.Handle("GET /app.js", http.FileServerFS(web.FS))
	mux.Handle("GET /style.css", http.FileServerFS(web.FS))

	mux.HandleFunc("GET /api/state", s.api(s.handleState))
	mux.HandleFunc("POST /api/plan", s.api(s.handlePlan))
	mux.HandleFunc("POST /api/apply", s.api(s.handleApply))
	return s.guard(mux)
}

// guard rejects cross-site requests: a browser-supplied Origin must be the host
// the page itself was loaded from. (The Host cannot be pinned because the
// tool is reached by LAN address; the token covers DNS rebinding.)
func (s *Server) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if o := r.Header.Get("Origin"); o != "" && o != "http://"+r.Host {
			http.Error(w, "bad origin", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) tokenOK(t string) bool {
	return subtle.ConstantTimeCompare([]byte(t), []byte(s.token)) == 1
}

// api wraps a handler with the token check (X-Token header) and requires JSON
// for anything but GET.
func (s *Server) api(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.tokenOK(r.Header.Get("X-Token")) {
			writeErr(w, http.StatusForbidden, "bad token")
			return
		}
		if r.Method != http.MethodGet && !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			writeErr(w, http.StatusUnsupportedMediaType, "json only")
			return
		}
		h(w, r)
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// handleIndex serves the page only with ?t=<token>, so a stranger on the
// network cannot read the token out of the page.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if !s.tokenOK(r.URL.Query().Get("t")) {
		http.Error(w, "需要 token：請使用 scan-origin 啟動時印出的完整網址（含 ?t=…）", http.StatusForbidden)
		return
	}
	b, err := fs.ReadFile(web.FS, "index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	page := strings.Replace(string(b), "__TOKEN__", s.token, 1)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write([]byte(page))
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	st, err := dataset.Scan(s.root)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"root": s.root, "version": version.Version, "stats": st})
}

func (s *Server) handlePlan(w http.ResponseWriter, r *http.Request) {
	p, err := dataset.BuildPlan(s.root)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleApply(w http.ResponseWriter, r *http.Request) {
	res, err := dataset.Apply(s.root)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}
