package server

import (
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/sora-san-df/ai-energy-management-platform/internal/platform/httpx"
	"github.com/sora-san-df/ai-energy-management-platform/internal/platform/middleware"
)

type Registrar interface {
	Register(mux *http.ServeMux)
}

type RegistrarFunc func(mux *http.ServeMux)

func (f RegistrarFunc) Register(mux *http.ServeMux) { f(mux) }

type RouterConfig struct {
	Public         []Registrar
	Protected      []Registrar
	Auth           func(http.Handler) http.Handler
	StaticDir      string
	AllowedOrigins []string
	Logger         *slog.Logger
}

func NewRouter(cfg RouterConfig) http.Handler {
	api := http.NewServeMux()
	for _, r := range cfg.Protected {
		r.Register(api)
	}
	api.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		httpx.Error(w, http.StatusNotFound, "not_found", "Recurso no encontrado")
	})

	root := http.NewServeMux()
	started := time.Now()
	root.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]any{"status": "ok", "uptime_seconds": int(time.Since(started).Seconds())})
	})
	for _, r := range cfg.Public {
		r.Register(root)
	}
	root.Handle("/api/", cfg.Auth(api))
	root.Handle("/", spaHandler(cfg.StaticDir))

	return middleware.Chain(root,
		middleware.Recover(cfg.Logger),
		middleware.Logger(cfg.Logger),
		middleware.CORS(cfg.AllowedOrigins),
		middleware.SecurityHeaders,
	)
}

func spaHandler(dir string) http.Handler {
	files := http.FileServer(http.Dir(dir))
	index := filepath.Join(dir, "index.html")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := os.Stat(index); err != nil {
			http.Error(w, "Frontend no compilado: ejecuta `npm run build` en /web", http.StatusServiceUnavailable)
			return
		}
		clean := path.Clean("/" + r.URL.Path)
		if clean != "/" {
			target := filepath.Join(dir, filepath.FromSlash(strings.TrimPrefix(clean, "/")))
			if info, err := os.Stat(target); err == nil && !info.IsDir() {
				if strings.HasPrefix(clean, "/assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				files.ServeHTTP(w, r)
				return
			}
		}
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, index)
	})
}
