package config

import (
	"errors"
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env     string `env:"APP_ENV" env-default:"development"`
	HTTP    HTTPConfig
	DB      DBConfig
	JWT     JWTConfig
	Log     LogConfig
	CORS    CORSConfig
	Storage StorageConfig
}

type HTTPConfig struct {
	Addr string `env:"HTTP_ADDR" env-default:":8080"`
}

type DBConfig struct {
	URL string `env:"DATABASE_URL"`
}

type JWTConfig struct {
	Secret   string `env:"JWT_SECRET"`
	TTLHours int    `env:"JWT_TTL_HOURS" env-default:"24"`
}

type LogConfig struct {
	Level string `env:"LOG_LEVEL" env-default:"info"`
}

type CORSConfig struct {
	AllowedOrigins string `env:"CORS_ALLOWED_ORIGINS" env-default:"http://localhost:5173"`
}

type StorageConfig struct {
	Dir            string `env:"STORAGE_DIR" env-default:"./storage"`
	MaxUploadBytes int64  `env:"STORAGE_MAX_UPLOAD_BYTES" env-default:"209715200"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	if cfg.Env == "production" {
		if cfg.JWT.Secret == "" {
			return nil, errors.New("JWT_SECRET is required in production")
		}
		if cfg.DB.URL == "" {
			return nil, errors.New("DATABASE_URL is required in production")
		}
	}
	if cfg.DB.URL == "" {
		cfg.DB.URL = "postgres://postgres:postgres@localhost:5432/ctf01d_development?sslmode=disable"
	}
	return &cfg, nil
}
