package config

import (
	"os"
	"testing"
)

func setenvs(t *testing.T, envs map[string]string) {
	t.Helper()
	for k, v := range envs {
		t.Setenv(k, v)
	}
}

func clearEnvForConfig(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"APP_ENV", "HTTP_ADDR", "DATABASE_URL", "JWT_SECRET",
		"JWT_TTL_HOURS", "LOG_LEVEL", "CORS_ALLOWED_ORIGINS",
		"STORAGE_DIR", "STORAGE_MAX_UPLOAD_BYTES",
	} {
		t.Setenv(key, "")
		os.Unsetenv(key)
	}
}

func TestLoad_Defaults(t *testing.T) {
	clearEnvForConfig(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Env != "development" {
		t.Errorf("Env = %q, want development", cfg.Env)
	}
	if cfg.HTTP.Addr != ":8080" {
		t.Errorf("HTTP.Addr = %q, want :8080", cfg.HTTP.Addr)
	}
	if cfg.DB.URL != "postgres://postgres:postgres@localhost:5432/ctf01d_development?sslmode=disable" {
		t.Errorf("DB.URL = %q, want default", cfg.DB.URL)
	}
	if cfg.JWT.TTLHours != 24 {
		t.Errorf("JWT.TTLHours = %d, want 24", cfg.JWT.TTLHours)
	}
	if cfg.Log.Level != "info" {
		t.Errorf("Log.Level = %q, want info", cfg.Log.Level)
	}
	if cfg.CORS.AllowedOrigins != "http://localhost:5173" {
		t.Errorf("CORS.AllowedOrigins = %q, want http://localhost:5173", cfg.CORS.AllowedOrigins)
	}
	if cfg.Storage.Dir != "./storage" {
		t.Errorf("Storage.Dir = %q, want ./storage", cfg.Storage.Dir)
	}
	if cfg.Storage.MaxUploadBytes != 209715200 {
		t.Errorf("Storage.MaxUploadBytes = %d, want 209715200", cfg.Storage.MaxUploadBytes)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	clearEnvForConfig(t)
	setenvs(t, map[string]string{
		"APP_ENV":                  "staging",
		"HTTP_ADDR":                ":9090",
		"DATABASE_URL":             "postgres://user:pass@db:5432/mydb",
		"JWT_SECRET":               "secret123",
		"JWT_TTL_HOURS":            "48",
		"LOG_LEVEL":                "debug",
		"CORS_ALLOWED_ORIGINS":     "http://localhost:3000",
		"STORAGE_DIR":              "/data/storage",
		"STORAGE_MAX_UPLOAD_BYTES": "104857600",
	})
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Env != "staging" {
		t.Errorf("Env = %q, want staging", cfg.Env)
	}
	if cfg.HTTP.Addr != ":9090" {
		t.Errorf("HTTP.Addr = %q, want :9090", cfg.HTTP.Addr)
	}
	if cfg.DB.URL != "postgres://user:pass@db:5432/mydb" {
		t.Errorf("DB.URL = %q, want custom", cfg.DB.URL)
	}
	if cfg.JWT.Secret != "secret123" {
		t.Errorf("JWT.Secret = %q, want secret123", cfg.JWT.Secret)
	}
	if cfg.JWT.TTLHours != 48 {
		t.Errorf("JWT.TTLHours = %d, want 48", cfg.JWT.TTLHours)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("Log.Level = %q, want debug", cfg.Log.Level)
	}
	if cfg.CORS.AllowedOrigins != "http://localhost:3000" {
		t.Errorf("CORS.AllowedOrigins = %q, want http://localhost:3000", cfg.CORS.AllowedOrigins)
	}
	if cfg.Storage.Dir != "/data/storage" {
		t.Errorf("Storage.Dir = %q, want /data/storage", cfg.Storage.Dir)
	}
	if cfg.Storage.MaxUploadBytes != 104857600 {
		t.Errorf("Storage.MaxUploadBytes = %d, want 104857600", cfg.Storage.MaxUploadBytes)
	}
}

func TestLoad_ProductionRequiresJWTSecret(t *testing.T) {
	clearEnvForConfig(t)
	t.Setenv("APP_ENV", "production")
	_, err := Load()
	if err == nil {
		t.Fatal("Load() should fail when JWT_SECRET is empty in production")
	}
}

func TestLoad_ProductionRequiresDatabaseURL(t *testing.T) {
	clearEnvForConfig(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "some-secret")
	t.Setenv("DATABASE_URL", "")
	os.Unsetenv("DATABASE_URL")
	_, err := Load()
	if err == nil {
		t.Fatal("Load() should fail when DATABASE_URL is empty in production")
	}
}

func TestLoad_ProductionValid(t *testing.T) {
	clearEnvForConfig(t)
	setenvs(t, map[string]string{
		"APP_ENV":      "production",
		"JWT_SECRET":   "super-secret-key",
		"DATABASE_URL": "postgres://user:pass@prod-db:5432/prod",
	})
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Env != "production" {
		t.Errorf("Env = %q, want production", cfg.Env)
	}
	if cfg.JWT.Secret != "super-secret-key" {
		t.Errorf("JWT.Secret = %q, want super-secret-key", cfg.JWT.Secret)
	}
	if cfg.DB.URL != "postgres://user:pass@prod-db:5432/prod" {
		t.Errorf("DB.URL = %q, want custom", cfg.DB.URL)
	}
}
