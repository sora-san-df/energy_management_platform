package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const devJWTSecret = "dev-only-secret-change-me-in-production"

type Config struct {
	Env               string
	Port              string
	DatabasePath      string
	DataDir           string
	StaticDir         string
	GeminiAPIKey      string
	GeminiModel       string
	GeminiTimeout     time.Duration
	GeminiMaxRetries  int
	JWTSecret         string
	TokenTTL          time.Duration
	DemoEmail         string
	DemoPassword      string
	DemoName          string
	AllowedOrigins    []string
	AnalysisStepDelay time.Duration
}

func (c Config) IsProduction() bool {
	return c.Env == "production"
}

func Load() (Config, error) {
	return LoadFrom(os.Getenv)
}

func LoadFrom(getenv func(string) string) (Config, error) {
	get := func(key, fallback string) string {
		if v := strings.TrimSpace(getenv(key)); v != "" {
			return v
		}
		return fallback
	}
	var errs []error
	duration := func(key string, fallback time.Duration) time.Duration {
		raw := getenv(key)
		if strings.TrimSpace(raw) == "" {
			return fallback
		}
		d, err := time.ParseDuration(strings.TrimSpace(raw))
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", key, err))
			return fallback
		}
		return d
	}
	cfg := Config{
		Env:               get("APP_ENV", "development"),
		Port:              get("PORT", "8080"),
		DatabasePath:      get("DATABASE_PATH", "energy.db"),
		DataDir:           get("DATA_DIR", "data"),
		StaticDir:         get("STATIC_DIR", filepath.Join("web", "dist")),
		GeminiAPIKey:      get("GEMINI_API_KEY", ""),
		GeminiModel:       get("GEMINI_MODEL", "gemini-2.5-flash"),
		GeminiTimeout:     duration("GEMINI_TIMEOUT", 20*time.Second),
		GeminiMaxRetries:  2,
		JWTSecret:         get("JWT_SECRET", ""),
		TokenTTL:          duration("TOKEN_TTL", 12*time.Hour),
		DemoEmail:         get("DEMO_EMAIL", "admin@energia.demo"),
		DemoPassword:      get("DEMO_PASSWORD", "demo1234"),
		DemoName:          get("DEMO_NAME", "Diego Mera"),
		AnalysisStepDelay: duration("ANALYSIS_STEP_DELAY", 400*time.Millisecond),
	}
	for _, origin := range strings.Split(getenv("ALLOWED_ORIGINS"), ",") {
		if o := strings.TrimSpace(origin); o != "" {
			cfg.AllowedOrigins = append(cfg.AllowedOrigins, o)
		}
	}
	if cfg.JWTSecret == "" {
		if cfg.IsProduction() {
			errs = append(errs, errors.New("JWT_SECRET is required in production"))
		}
		cfg.JWTSecret = devJWTSecret
	}
	if cfg.IsProduction() && len(cfg.JWTSecret) < 32 {
		errs = append(errs, errors.New("JWT_SECRET must have at least 32 characters in production"))
	}
	return cfg, errors.Join(errs...)
}

func LoadDotEnv(path string) error {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(strings.TrimPrefix(key, "export "))
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}
	return scanner.Err()
}
