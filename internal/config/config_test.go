package config

import (
	"log/slog"
	"testing"
)

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/testdb")
	t.Setenv("JWT_SECRET", "test-secret-key-at-least-32-chars!!")
}

func TestLoad_Defaults(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.DatabaseURL != "postgres://user:pass@localhost:5432/testdb" {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, "postgres://user:pass@localhost:5432/testdb")
	}
}

func TestLoad_JWTDefaults(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.JWTAccessTokenTTL != 900 {
		t.Errorf("JWTAccessTokenTTL = %d, want 900", cfg.JWTAccessTokenTTL)
	}
	if cfg.JWTRefreshTokenTTL != 604800 {
		t.Errorf("JWTRefreshTokenTTL = %d, want 604800", cfg.JWTRefreshTokenTTL)
	}
	if cfg.BcryptCost != 12 {
		t.Errorf("BcryptCost = %d, want 12", cfg.BcryptCost)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("PORT", "3000")
	t.Setenv("DATABASE_URL", "postgres://custom:pass@db:5432/custom")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("JWT_SECRET", "custom-secret-key-at-least-32-chars!!")
	t.Setenv("JWT_ACCESS_TOKEN_TTL", "600")
	t.Setenv("JWT_REFRESH_TOKEN_TTL", "86400")
	t.Setenv("BCRYPT_COST", "10")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != 3000 {
		t.Errorf("Port = %d, want 3000", cfg.Port)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
	if cfg.JWTSecret != "custom-secret-key-at-least-32-chars!!" {
		t.Errorf("JWTSecret = %q, want %q", cfg.JWTSecret, "custom-secret-key-at-least-32-chars!!")
	}
	if cfg.JWTAccessTokenTTL != 600 {
		t.Errorf("JWTAccessTokenTTL = %d, want 600", cfg.JWTAccessTokenTTL)
	}
	if cfg.JWTRefreshTokenTTL != 86400 {
		t.Errorf("JWTRefreshTokenTTL = %d, want 86400", cfg.JWTRefreshTokenTTL)
	}
	if cfg.BcryptCost != 10 {
		t.Errorf("BcryptCost = %d, want 10", cfg.BcryptCost)
	}
}

func TestLoad_MissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "some-secret")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for missing DATABASE_URL, got nil")
	}
}

func TestLoad_MissingJWTSecret(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/testdb")
	t.Setenv("JWT_SECRET", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for missing JWT_SECRET, got nil")
	}
}

func TestValidate_ShortJWTSecret(t *testing.T) {
	cfg := &Config{JWTSecret: "short", DatabaseURL: "postgres://x"}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for short JWT secret")
	}
}

func TestValidate_ValidJWTSecret(t *testing.T) {
	cfg := &Config{JWTSecret: "a]32-char-minimum-secret-string!", DatabaseURL: "postgres://x"}
	if err := cfg.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_ExactlyMinLength(t *testing.T) {
	cfg := &Config{JWTSecret: "abcdefghijklmnopqrstuvwxyz012345", DatabaseURL: "postgres://x"}
	if err := cfg.Validate(); err != nil {
		t.Errorf("32-char secret should pass: %v", err)
	}
}

func TestValidate_JustBelowMinLength(t *testing.T) {
	cfg := &Config{JWTSecret: "abcdefghijklmnopqrstuvwxyz01234", DatabaseURL: "postgres://x"}
	if err := cfg.Validate(); err == nil {
		t.Error("31-char secret should fail")
	}
}

func TestLoad_ShortJWTSecret(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/testdb")
	t.Setenv("JWT_SECRET", "too-short")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should fail for short JWT_SECRET")
	}
}

func TestSlogLevel(t *testing.T) {
	tests := []struct {
		level string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"unknown", slog.LevelInfo},
		{"", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			cfg := &Config{LogLevel: tt.level}
			if got := cfg.SlogLevel(); got != tt.want {
				t.Errorf("SlogLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}
