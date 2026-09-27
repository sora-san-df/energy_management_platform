package auth

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
)

const issuer = "ai-energy-management-platform"

type Config struct {
	Email    string
	Password string
	Name     string
	Secret   []byte
	TTL      time.Duration
}

type User struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

type Session struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresAt   time.Time `json:"expires_at"`
	User        User      `json:"user"`
}

type claims struct {
	Name string `json:"name"`
	Role string `json:"role"`
	jwt.RegisteredClaims
}

type Service struct {
	cfg Config
	now func() time.Time
}

func NewService(cfg Config) *Service {
	cfg.Email = normalizeEmail(cfg.Email)
	if cfg.Name == "" {
		cfg.Name = "Energy Manager"
	}
	return &Service{cfg: cfg, now: time.Now}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (s *Service) user() User {
	return User{Email: s.cfg.Email, Name: s.cfg.Name, Role: "Energy Manager"}
}

func (s *Service) Login(email, password string) (Session, error) {
	emailOK := subtle.ConstantTimeCompare([]byte(normalizeEmail(email)), []byte(s.cfg.Email)) == 1
	passwordOK := subtle.ConstantTimeCompare([]byte(password), []byte(s.cfg.Password)) == 1
	if !emailOK || !passwordOK || s.cfg.Email == "" {
		return Session{}, ErrInvalidCredentials
	}
	now := s.now()
	expires := now.Add(s.cfg.TTL)
	u := s.user()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		Name: u.Name,
		Role: u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.Email,
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expires),
		},
	})
	signed, err := token.SignedString(s.cfg.Secret)
	if err != nil {
		return Session{}, fmt.Errorf("sign token: %w", err)
	}
	return Session{AccessToken: signed, TokenType: "Bearer", ExpiresAt: expires, User: u}, nil
}

func (s *Service) Verify(raw string) (User, error) {
	var c claims
	_, err := jwt.ParseWithClaims(raw, &c, func(t *jwt.Token) (any, error) {
		return s.cfg.Secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer(issuer), jwt.WithTimeFunc(s.now), jwt.WithExpirationRequired())
	if err != nil {
		return User{}, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	return User{Email: c.Subject, Name: c.Name, Role: c.Role}, nil
}
