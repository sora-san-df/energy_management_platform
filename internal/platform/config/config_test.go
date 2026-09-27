package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func env(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := LoadFrom(env(nil))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != "8080" || cfg.DatabasePath != "energy.db" || cfg.DataDir != "data" || cfg.StaticDir != filepath.Join("web", "dist") {
		t.Fatalf("unexpected defaults %+v", cfg)
	}
	if cfg.GeminiModel != "gemini-2.5-flash" || cfg.GeminiTimeout != 20*time.Second || cfg.GeminiAPIKey != "" {
		t.Fatalf("unexpected gemini defaults %+v", cfg)
	}
	if cfg.DemoEmail == "" || cfg.DemoPassword == "" || cfg.JWTSecret == "" || cfg.TokenTTL != 12*time.Hour {
		t.Fatalf("unexpected auth defaults %+v", cfg)
	}
	if cfg.IsProduction() {
		t.Fatal("default environment must be development")
	}
}

func TestLoadOverrides(t *testing.T) {
	cfg, err := LoadFrom(env(map[string]string{
		"PORT":                "9000",
		"DATABASE_PATH":       "/tmp/x.db",
		"GEMINI_API_KEY":      "key",
		"GEMINI_MODEL":        "gemini-pro",
		"GEMINI_TIMEOUT":      "5s",
		"ALLOWED_ORIGINS":     "http://localhost:5173, https://app.example.com",
		"ANALYSIS_STEP_DELAY": "0s",
		"TOKEN_TTL":           "1h",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != "9000" || cfg.DatabasePath != "/tmp/x.db" || cfg.GeminiAPIKey != "key" || cfg.GeminiModel != "gemini-pro" || cfg.GeminiTimeout != 5*time.Second {
		t.Fatalf("overrides not applied %+v", cfg)
	}
	if len(cfg.AllowedOrigins) != 2 || cfg.AllowedOrigins[1] != "https://app.example.com" {
		t.Fatalf("unexpected origins %v", cfg.AllowedOrigins)
	}
	if cfg.AnalysisStepDelay != 0 || cfg.TokenTTL != time.Hour {
		t.Fatalf("unexpected durations %+v", cfg)
	}
}

func TestLoadRejectsInvalidDuration(t *testing.T) {
	if _, err := LoadFrom(env(map[string]string{"GEMINI_TIMEOUT": "soon"})); err == nil {
		t.Fatal("expected error for invalid duration")
	}
}

func TestProductionRequiresSecrets(t *testing.T) {
	if _, err := LoadFrom(env(map[string]string{"APP_ENV": "production"})); err == nil {
		t.Fatal("production must require an explicit JWT_SECRET")
	}
	cfg, err := LoadFrom(env(map[string]string{"APP_ENV": "production", "JWT_SECRET": "a-very-long-production-secret-value!!"}))
	if err != nil || !cfg.IsProduction() {
		t.Fatalf("expected valid production config, got %v", err)
	}
}

func TestLoadDotEnvDoesNotOverrideExistingVariables(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	content := "# comment\nDOTENV_TEST_A=from-file\nDOTENV_TEST_B=\"quoted value\"\n\nDOTENV_TEST_C=from-file\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOTENV_TEST_C", "from-env")
	os.Unsetenv("DOTENV_TEST_A")
	os.Unsetenv("DOTENV_TEST_B")
	t.Cleanup(func() {
		os.Unsetenv("DOTENV_TEST_A")
		os.Unsetenv("DOTENV_TEST_B")
	})

	if err := LoadDotEnv(path); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("DOTENV_TEST_A") != "from-file" || os.Getenv("DOTENV_TEST_B") != "quoted value" || os.Getenv("DOTENV_TEST_C") != "from-env" {
		t.Fatalf("unexpected env A=%q B=%q C=%q", os.Getenv("DOTENV_TEST_A"), os.Getenv("DOTENV_TEST_B"), os.Getenv("DOTENV_TEST_C"))
	}
}

func TestLoadDotEnvMissingFileIsIgnored(t *testing.T) {
	if err := LoadDotEnv(filepath.Join(t.TempDir(), "missing.env")); err != nil {
		t.Fatalf("missing .env must be ignored, got %v", err)
	}
}
