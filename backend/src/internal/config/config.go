package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"

	"github.com/miem-3d-lab-print/miem-3d-lab-print/backend/src/internal/db"
)

type Config struct {
	SMTP         SMTPConfig
	SiteURL      string
	Database     DatabaseConfig
	JWT          JWTConfig
	MinIO        MinIOConfig
	Server       ServerConfig
	DatabasePool db.PoolConfig
	CORSOrigins  string
	TrustProxy   bool
}

type ServerConfig struct {
	Address           string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

type DatabaseConfig struct {
	URL string
}

type JWTConfig struct {
	Secret string
}

type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

func Load() (Config, error) {
	_ = godotenv.Load()

	smtpUsername := getenv("SMTP_USERNAME", "")
	smtpPassword := getenv("SMTP_PASSWORD", "")
	if (smtpUsername == "") != (smtpPassword == "") {
		return Config{}, fmt.Errorf("SMTP_USERNAME and SMTP_PASSWORD must be set together")
	}
	smtpFrom := getenv("SMTP_FROM", smtpUsername)
	if smtpFrom == "" {
		smtpFrom = "noreply@miem-3d-lab.local"
	}

	databaseURL := getenv("DATABASE_URL", "")
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	jwtSecret := getenv("JWT_SECRET", "")
	if len(jwtSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}

	minioEndpoint := getenv("MINIO_ENDPOINT", "")
	if minioEndpoint == "" {
		return Config{}, fmt.Errorf("MINIO_ENDPOINT is required")
	}
	minioAccessKey := getenv("MINIO_ACCESS_KEY", "")
	if minioAccessKey == "" {
		return Config{}, fmt.Errorf("MINIO_ACCESS_KEY is required")
	}
	minioSecretKey := getenv("MINIO_SECRET_KEY", "")
	if minioSecretKey == "" {
		return Config{}, fmt.Errorf("MINIO_SECRET_KEY is required")
	}

	trustProxy, err := getenvBool("TRUST_PROXY", false)
	if err != nil {
		return Config{}, err
	}
	minioUseSSL, err := getenvBool("MINIO_USE_SSL", false)
	if err != nil {
		return Config{}, err
	}
	databasePool, err := loadDatabasePool()
	if err != nil {
		return Config{}, err
	}
	readHeaderTimeout, err := getenvDuration("HTTP_READ_HEADER_TIMEOUT", 5*time.Second)
	if err != nil {
		return Config{}, err
	}
	readTimeout, err := getenvDuration("HTTP_READ_TIMEOUT", 2*time.Minute)
	if err != nil {
		return Config{}, err
	}
	writeTimeout, err := getenvDuration("HTTP_WRITE_TIMEOUT", 2*time.Minute)
	if err != nil {
		return Config{}, err
	}
	idleTimeout, err := getenvDuration("HTTP_IDLE_TIMEOUT", time.Minute)
	if err != nil {
		return Config{}, err
	}

	return Config{
		CORSOrigins: getenv("CORS_ALLOWED_ORIGINS", ""),
		TrustProxy:  trustProxy,
		SiteURL:     getenv("SITE_URL", "http://localhost:3000"),
		SMTP: SMTPConfig{
			Host:     getenv("SMTP_HOST", "localhost"),
			Port:     getenv("SMTP_PORT", "1025"),
			Username: smtpUsername,
			Password: smtpPassword,
			From:     smtpFrom,
		},
		Database: DatabaseConfig{
			URL: databaseURL,
		},
		JWT: JWTConfig{
			Secret: jwtSecret,
		},
		MinIO: MinIOConfig{
			Endpoint:  minioEndpoint,
			AccessKey: minioAccessKey,
			SecretKey: minioSecretKey,
			Bucket:    getenv("MINIO_BUCKET", "3d-files"),
			UseSSL:    minioUseSSL,
		},
		Server: ServerConfig{
			Address:           getenv("HTTP_ADDRESS", getenv("SERVER_ADDRESS", ":8080")),
			ReadHeaderTimeout: readHeaderTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
		},
		DatabasePool: databasePool,
	}, nil
}

func loadDatabasePool() (db.PoolConfig, error) {
	maxOpen, err := getenvInt("DB_MAX_OPEN_CONNS", 25)
	if err != nil {
		return db.PoolConfig{}, err
	}
	maxIdle, err := getenvInt("DB_MAX_IDLE_CONNS", 10)
	if err != nil {
		return db.PoolConfig{}, err
	}
	maxLifetime, err := getenvDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute)
	if err != nil {
		return db.PoolConfig{}, err
	}
	maxIdleTime, err := getenvDuration("DB_CONN_MAX_IDLE_TIME", 5*time.Minute)
	if err != nil {
		return db.PoolConfig{}, err
	}
	if maxOpen < 1 || maxIdle < 0 || maxIdle > maxOpen {
		return db.PoolConfig{}, fmt.Errorf("database pool limits are invalid")
	}
	return db.PoolConfig{
		MaxOpenConns:    maxOpen,
		MaxIdleConns:    maxIdle,
		ConnMaxLifetime: maxLifetime,
		ConnMaxIdleTime: maxIdleTime,
	}, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvBool(key string, fallback bool) (bool, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s must be a boolean: %w", key, err)
	}
	return parsed, nil
}

func getenvInt(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	return parsed, nil
}

func getenvDuration(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration: %w", key, err)
	}
	return parsed, nil
}
