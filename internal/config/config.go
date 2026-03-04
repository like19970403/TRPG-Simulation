package config

import (
	"fmt"
	"log/slog"

	"github.com/caarlos0/env/v11"
)

// Config holds all configuration values loaded from environment variables.
type Config struct {
	Port        int    `env:"PORT"                  envDefault:"8080"`
	DatabaseURL string `env:"DATABASE_URL,required,notEmpty"`
	LogLevel    string `env:"LOG_LEVEL"             envDefault:"info"`

	JWTSecret          string `env:"JWT_SECRET,required,notEmpty"`
	JWTAccessTokenTTL  int    `env:"JWT_ACCESS_TOKEN_TTL"  envDefault:"900"`
	JWTRefreshTokenTTL int    `env:"JWT_REFRESH_TOKEN_TTL" envDefault:"604800"`
	BcryptCost         int    `env:"BCRYPT_COST"           envDefault:"12"`
	CookieSecure       bool   `env:"COOKIE_SECURE"         envDefault:"true"`
	UploadDir          string `env:"UPLOAD_DIR"            envDefault:"./uploads"`
	AllowedOrigins     string `env:"ALLOWED_ORIGINS"       envDefault:""`
}

// Load parses environment variables into a Config struct.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("config: parse env: %w", err)
	}
	return cfg, nil
}

// SlogLevel converts the string log level to slog.Level.
func (c *Config) SlogLevel() slog.Level {
	switch c.LogLevel {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
