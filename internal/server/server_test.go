package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTest(t *testing.T) (*Server, http.Handler) {
	t.Helper()
	root := t.TempDir()
	p := filepath.Join(root, "261028_1", "c1", "a-Tooth_11.stl")
	os.MkdirAll(filepath.Dir(p), 0o755)
	os.WriteFile(p, []byte("x"), 0o644)
	s := New(root)
	return s, s.Handler()
}

func do(h http.Handler, method, path string, hdr map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader("{}"))
	r.Host = "192.168.1.5:8765"
	for k, v := range hdr {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestIndexNeedsToken(t *testing.T) {
	s, h := newTest(t)
	if w := do(h, "GET", "/", nil); w.Code != 403 {
		t.Errorf("no token: %d", w.Code)
	}
	if w := do(h, "GET", "/?t=wrong", nil); w.Code != 403 {
		t.Errorf("wrong token: %d", w.Code)
	}
	w := do(h, "GET", "/?t="+s.Token(), nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), s.Token()) || strings.Contains(w.Body.String(), "__TOKEN__") {
		t.Errorf("good token: %d", w.Code)
	}
}

func TestAPIGuards(t *testing.T) {
	s, h := newTest(t)
	if w := do(h, "GET", "/api/state", nil); w.Code != 403 {
		t.Errorf("no token: %d", w.Code)
	}
	if w := do(h, "POST", "/api/apply", map[string]string{"X-Token": s.Token()}); w.Code != 415 {
		t.Errorf("non-json POST: %d", w.Code)
	}
	if w := do(h, "POST", "/api/apply", map[string]string{"X-Token": s.Token(), "Content-Type": "application/json", "Origin": "http://evil.example"}); w.Code != 403 {
		t.Errorf("cross-origin: %d", w.Code)
	}
	ok := map[string]string{"X-Token": s.Token(), "Content-Type": "application/json"}
	if w := do(h, "GET", "/api/state", ok); w.Code != 200 || !strings.Contains(w.Body.String(), `"pending":1`) {
		t.Errorf("state: %d %s", w.Code, w.Body.String())
	}
}

func TestTeethNeedsToken(t *testing.T) {
	s, h := newTest(t)
	if w := do(h, "GET", "/api/teeth", nil); w.Code != 403 {
		t.Errorf("no token: %d", w.Code)
	}
	ok := map[string]string{"X-Token": s.Token(), "Content-Type": "application/json"}
	if w := do(h, "GET", "/api/teeth", ok); w.Code != 200 || !strings.Contains(w.Body.String(), `"teeth":[]`) {
		t.Errorf("teeth: %d %s", w.Code, w.Body.String())
	}
}

func TestPlanThenApply(t *testing.T) {
	s, h := newTest(t)
	ok := map[string]string{"X-Token": s.Token(), "Content-Type": "application/json", "Origin": "http://192.168.1.5:8765"}
	if w := do(h, "POST", "/api/plan", ok); w.Code != 200 || !strings.Contains(w.Body.String(), `"kind":"Die"`) {
		t.Errorf("plan: %d %s", w.Code, w.Body.String())
	}
	if _, err := os.Stat(filepath.Join(s.root, "261028_1", "c1")); err != nil {
		t.Error("plan must not move anything")
	}
	if w := do(h, "POST", "/api/apply", ok); w.Code != 200 || !strings.Contains(w.Body.String(), `"moved":1`) {
		t.Errorf("apply: %d %s", w.Code, w.Body.String())
	}
	if _, err := os.Stat(filepath.Join(s.root, "Die", "261028_1", "c1", "a-Tooth_11.stl")); err != nil {
		t.Error(err)
	}
}

func TestExportXLSX(t *testing.T) {
	s, h := newTest(t)
	if w := do(h, "GET", "/api/export.xlsx", nil); w.Code != 403 {
		t.Errorf("no token: %d", w.Code)
	}
	ok := map[string]string{"X-Token": s.Token(), "Content-Type": "application/json"}
	w := do(h, "GET", "/api/export.xlsx", ok)
	if w.Code != 200 || !strings.HasPrefix(w.Body.String(), "PK") ||
		!strings.Contains(w.Header().Get("Content-Disposition"), "attachment; filename=\"scan-origin-") {
		t.Errorf("export: %d %q", w.Code, w.Header())
	}
}
