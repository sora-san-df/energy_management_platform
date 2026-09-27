package server

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type registrar func(*http.ServeMux)

func (r registrar) Register(mux *http.ServeMux) { r(mux) }

func denyAll(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer ok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func newRouter(t *testing.T, staticDir string) http.Handler {
	t.Helper()
	return NewRouter(RouterConfig{
		Public: []Registrar{registrar(func(mux *http.ServeMux) {
			mux.HandleFunc("POST /api/auth/login", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("login")) })
		})},
		Protected: []Registrar{registrar(func(mux *http.ServeMux) {
			mux.HandleFunc("GET /api/meters", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("meters")) })
		})},
		Auth:      denyAll,
		StaticDir: staticDir,
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
}

func get(h http.Handler, method, target string, headers ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func staticDir(t *testing.T) string {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>app</html>"), 0o600)
	os.MkdirAll(filepath.Join(dir, "assets"), 0o700)
	os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte("console.log(1)"), 0o600)
	return dir
}

func TestHealthIsPublic(t *testing.T) {
	rec := get(newRouter(t, staticDir(t)), http.MethodGet, "/api/health")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Fatalf("unexpected health %d %s", rec.Code, rec.Body.String())
	}
}

func TestPublicRoutesBypassAuth(t *testing.T) {
	rec := get(newRouter(t, staticDir(t)), http.MethodPost, "/api/auth/login")
	if rec.Code != http.StatusOK || rec.Body.String() != "login" {
		t.Fatalf("unexpected login %d %s", rec.Code, rec.Body.String())
	}
}

func TestProtectedRoutesRequireAuth(t *testing.T) {
	h := newRouter(t, staticDir(t))
	if rec := get(h, http.MethodGet, "/api/meters"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec := get(h, http.MethodGet, "/api/meters", "Authorization", "Bearer ok"); rec.Code != http.StatusOK || rec.Body.String() != "meters" {
		t.Fatalf("unexpected %d %s", rec.Code, rec.Body.String())
	}
}

func TestUnknownAPIRouteReturnsJSON404(t *testing.T) {
	rec := get(newRouter(t, staticDir(t)), http.MethodGet, "/api/nope", "Authorization", "Bearer ok")
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), `"code":"not_found"`) {
		t.Fatalf("unexpected %d %s", rec.Code, rec.Body.String())
	}
}

func TestSPAServesAssetsAndFallsBackToIndex(t *testing.T) {
	h := newRouter(t, staticDir(t))

	if rec := get(h, http.MethodGet, "/assets/app.js"); rec.Code != http.StatusOK || rec.Body.String() != "console.log(1)" {
		t.Fatalf("asset not served: %d %s", rec.Code, rec.Body.String())
	}
	for _, target := range []string{"/", "/meters/M-109", "/anomalies/3"} {
		rec := get(h, http.MethodGet, target)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "app") {
			t.Fatalf("%s: expected index.html, got %d %s", target, rec.Code, rec.Body.String())
		}
	}
	if rec := get(h, http.MethodGet, "/../../etc/passwd"); strings.Contains(rec.Body.String(), "root:") {
		t.Fatal("path traversal must not escape the static dir")
	}
}

func TestSPAWithoutBuildReturns503(t *testing.T) {
	rec := get(newRouter(t, filepath.Join(t.TempDir(), "missing")), http.MethodGet, "/")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", rec.Code)
	}
}
