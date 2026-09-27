package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var now = time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC)

func newTestService() *Service {
	svc := NewService(Config{
		Email:    "admin@energia.demo",
		Password: "demo1234",
		Name:     "Ana Operaciones",
		Secret:   []byte("test-secret-with-enough-length-123"),
		TTL:      time.Hour,
	})
	svc.now = func() time.Time { return now }
	return svc
}

func TestLoginAndVerify(t *testing.T) {
	svc := newTestService()

	session, err := svc.Login("Admin@Energia.demo ", "demo1234")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if session.AccessToken == "" || !session.ExpiresAt.Equal(now.Add(time.Hour)) || session.User.Email != "admin@energia.demo" {
		t.Fatalf("unexpected session %+v", session)
	}

	user, err := svc.Verify(session.AccessToken)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if user.Email != "admin@energia.demo" || user.Name != "Ana Operaciones" || user.Role == "" {
		t.Fatalf("unexpected user %+v", user)
	}
}

func TestLoginRejectsInvalidCredentials(t *testing.T) {
	svc := newTestService()
	for _, c := range [][2]string{{"admin@energia.demo", "wrong"}, {"other@energia.demo", "demo1234"}, {"", ""}} {
		if _, err := svc.Login(c[0], c[1]); !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("login(%q) expected ErrInvalidCredentials, got %v", c[0], err)
		}
	}
}

func TestVerifyRejectsExpiredToken(t *testing.T) {
	svc := newTestService()
	session, _ := svc.Login("admin@energia.demo", "demo1234")
	svc.now = func() time.Time { return now.Add(2 * time.Hour) }

	if _, err := svc.Verify(session.AccessToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestVerifyRejectsTamperedToken(t *testing.T) {
	svc := newTestService()
	session, _ := svc.Login("admin@energia.demo", "demo1234")

	other := NewService(Config{Email: "admin@energia.demo", Password: "demo1234", Secret: []byte("another-secret-with-enough-length"), TTL: time.Hour})
	if _, err := other.Verify(session.AccessToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for foreign signature, got %v", err)
	}
	if _, err := svc.Verify("not-a-token"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for garbage, got %v", err)
	}
}

func TestMiddleware(t *testing.T) {
	svc := newTestService()
	session, _ := svc.Login("admin@energia.demo", "demo1234")
	protected := Middleware(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := UserFrom(r.Context())
		if !ok {
			t.Fatal("user missing from context")
		}
		w.Write([]byte(user.Email))
	}))

	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/meters", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("missing token status = %d", rec.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/meters", nil)
	req.Header.Set("Authorization", "Bearer invalid")
	rec = httptest.NewRecorder()
	protected.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("invalid token status = %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/meters", nil)
	req.Header.Set("Authorization", "Bearer "+session.AccessToken)
	rec = httptest.NewRecorder()
	protected.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "admin@energia.demo" {
		t.Fatalf("valid token response %d %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerLogin(t *testing.T) {
	svc := newTestService()
	mux := http.NewServeMux()
	NewHandler(svc).RegisterPublic(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"email":"admin@energia.demo","password":"demo1234"}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var session Session
	if err := json.Unmarshal(rec.Body.Bytes(), &session); err != nil || session.AccessToken == "" {
		t.Fatalf("unexpected body %s", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"email":"admin@energia.demo","password":"nope"}`)))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password status = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad json status = %d", rec.Code)
	}
}

func TestHandlerMe(t *testing.T) {
	svc := newTestService()
	session, _ := svc.Login("admin@energia.demo", "demo1234")
	mux := http.NewServeMux()
	NewHandler(svc).RegisterProtected(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+session.AccessToken)
	rec := httptest.NewRecorder()
	Middleware(svc)(mux).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Ana Operaciones") {
		t.Fatalf("unexpected response %d %s", rec.Code, rec.Body.String())
	}
}
