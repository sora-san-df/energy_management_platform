package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/sora-san-df/ai-energy-management-platform/internal/platform/httpx"
)

type contextKey struct{}

type Verifier interface {
	Verify(token string) (User, error)
}

func UserFrom(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(contextKey{}).(User)
	return u, ok
}

func Middleware(v Verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || strings.TrimSpace(token) == "" {
				httpx.Error(w, http.StatusUnauthorized, "unauthorized", "Se requiere autenticación")
				return
			}
			user, err := v.Verify(strings.TrimSpace(token))
			if err != nil {
				httpx.Error(w, http.StatusUnauthorized, "invalid_token", "Sesión inválida o expirada")
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextKey{}, user)))
		})
	}
}

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterPublic(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/auth/login", h.login)
}

func (h *Handler) RegisterProtected(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/auth/me", h.me)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	session, err := h.svc.Login(req.Email, req.Password)
	if errors.Is(err, ErrInvalidCredentials) {
		httpx.Error(w, http.StatusUnauthorized, "invalid_credentials", "Correo o contraseña incorrectos")
		return
	}
	if err != nil {
		httpx.InternalError(w)
		return
	}
	httpx.JSON(w, http.StatusOK, session)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFrom(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "Se requiere autenticación")
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}
