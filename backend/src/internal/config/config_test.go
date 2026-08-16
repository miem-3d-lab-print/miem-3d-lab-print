package config

import (
	"strings"
	"testing"
	"time"
)

func setRequiredEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("JWT_SECRET", strings.Repeat("s", 32))
	t.Setenv("MINIO_ENDPOINT", "localhost:9000")
	t.Setenv("MINIO_ACCESS_KEY", "access")
	t.Setenv("MINIO_SECRET_KEY", "secret")
	t.Setenv("SMTP_USERNAME", "")
	t.Setenv("SMTP_PASSWORD", "")
	t.Setenv("TRUST_PROXY", "")
	t.Setenv("MINIO_USE_SSL", "")
	t.Setenv("DB_MAX_OPEN_CONNS", "")
	t.Setenv("DB_MAX_IDLE_CONNS", "")
	t.Setenv("DB_CONN_MAX_LIFETIME", "")
	t.Setenv("DB_CONN_MAX_IDLE_TIME", "")
	t.Setenv("HTTP_ADDRESS", "")
	t.Setenv("SERVER_ADDRESS", "")
	t.Setenv("HTTP_READ_HEADER_TIMEOUT", "")
	t.Setenv("HTTP_READ_TIMEOUT", "")
	t.Setenv("HTTP_WRITE_TIMEOUT", "")
	t.Setenv("HTTP_IDLE_TIMEOUT", "")
}

func TestLoadUsesDevelopmentDefaults(t *testing.T) {
	setRequiredEnvironment(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an error: %v", err)
	}
	if cfg.Server.Address != ":8080" {
		t.Errorf("Server.Address = %q, want %q", cfg.Server.Address, ":8080")
	}
	if cfg.SMTP.Host != "localhost" || cfg.SMTP.Port != "1025" {
		t.Errorf("SMTP address = %s:%s, want localhost:1025", cfg.SMTP.Host, cfg.SMTP.Port)
	}
	if cfg.MinIO.UseSSL {
		t.Error("MinIO.UseSSL = true, want false")
	}
	if cfg.DatabasePool.MaxOpenConns != 25 || cfg.DatabasePool.MaxIdleConns != 10 {
		t.Errorf("database pool = %d/%d, want 25/10", cfg.DatabasePool.MaxOpenConns, cfg.DatabasePool.MaxIdleConns)
	}
	if cfg.Server.ReadHeaderTimeout != 5*time.Second || cfg.Server.IdleTimeout != time.Minute {
		t.Errorf("unexpected HTTP timeouts: %+v", cfg.Server)
	}
}

func TestLoadRejectsShortJWTSecret(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("JWT_SECRET", "short")

	if _, err := Load(); err == nil {
		t.Fatal("Load() returned nil error for a short JWT secret")
	}
}

func TestLoadRejectsPartialSMTPCredentials(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("SMTP_USERNAME", "user")

	if _, err := Load(); err == nil {
		t.Fatal("Load() returned nil error for partial SMTP credentials")
	}
}

func TestLoadRejectsInvalidBoolean(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("TRUST_PROXY", "sometimes")

	if _, err := Load(); err == nil {
		t.Fatal("Load() returned nil error for an invalid boolean")
	}
}

func TestLoadRejectsInvalidDatabasePool(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("DB_MAX_OPEN_CONNS", "5")
	t.Setenv("DB_MAX_IDLE_CONNS", "6")

	if _, err := Load(); err == nil {
		t.Fatal("Load() returned nil error for an invalid database pool")
	}
}
