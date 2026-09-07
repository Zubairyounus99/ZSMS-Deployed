package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config represents the application configuration.
type Config struct {
	// App Environment
	AppEnv   string
	AppName  string
	AppDebug bool
	Timezone string

	// Domains & URLs
	AppURL  string
	WebURL  string
	APIURL  string
	DocsURL string

	// Logging & Debug
	LogLevel string

	// Server Binding
	Port               int
	Host               string
	CORSAllowedOrigins []string
	AllowHTTPLocal     bool

	// PostgreSQL
	DatabaseURL          string
	DBMaxOpenConns       int
	DBMaxIdleConns       int
	DBConnMaxLifetimeMin int

	// Redis
	RedisURL        string
	RedisMaxRetries int

	// Auth & Security
	AuthDisabled               bool
	JWTSecret                  string
	JWTExpirationHours         int
	PairingCodeExpirationMin   int
	DeviceTokenPrefix          string
	EncryptionKey              string

	// Firebase (FCM v1)
	FirebaseProjectID   string
	FirebaseClientEmail string
	FirebasePrivateKey  string

	// Storage
	StorageProvider string // "minio" or "r2"
	S3Endpoint      string
	S3AccessKey     string
	S3SecretKey     string
	S3Bucket        string
	S3Region        string
	S3UseSSL        bool
	R2Endpoint      string
	R2AccessKey     string
	R2SecretKey     string
	R2Bucket        string

	// Webhooks
	WebhookSigningSecret  string
	WebhookTimeoutSeconds int
	WebhookMaxRetries     int

	// SMTP
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string
	SMTPFromName string
	SMTPTLS      bool

	// Turnstile
	TurnstileSiteKey   string
	TurnstileSecretKey string

	// Limits
	MaxBulkBatchSize             int
	DefaultMessageExpiryHours    int
	PhoneOfflineThresholdSeconds int
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	appEnv := getEnv("APP_ENV", "development")
	isProd := appEnv == "production"

	defaultAppURL := "http://localhost:3000"
	defaultWebURL := "http://localhost:3000"
	defaultAPIURL := "http://localhost:8080"
	defaultDocsURL := "http://localhost:8080/docs"
	defaultAllowHTTP := true
	defaultPGHost := "localhost"
	defaultRedisHost := "localhost"

	if isProd {
		defaultAppURL = "https://sms.ztechai.us"
		defaultWebURL = "https://sms.ztechai.us"
		defaultAPIURL = "https://sms-api.ztechai.us"
		defaultDocsURL = "https://sms-api.ztechai.us/docs"
		defaultAllowHTTP = false
		defaultPGHost = "postgres"
		defaultRedisHost = "redis"
	}

	cfg := &Config{
		AppEnv:   appEnv,
		AppName:  getEnv("APP_NAME", "ZSMS"),
		AppDebug: getEnvAsBool("APP_DEBUG", !isProd),
		Timezone: getEnv("TZ", "UTC"),
		LogLevel: getEnv("LOG_LEVEL", "info"),

		AppURL:  getEnv("APP_URL", defaultAppURL),
		WebURL:  getEnv("WEB_URL", defaultWebURL),
		APIURL:  getEnv("API_URL", defaultAPIURL),
		DocsURL: getEnv("DOCS_URL", defaultDocsURL),

		Port: getEnvAsInt("PORT", 8080),
		Host: getEnv("HOST", "0.0.0.0"),
		CORSAllowedOrigins: func() []string {
			if origins := getEnvAsSlice("CORS_ALLOW_ORIGINS", nil); len(origins) > 0 {
				return origins
			}
			if origins := getEnvAsSlice("CORS_ALLOWED_ORIGINS", nil); len(origins) > 0 {
				return origins
			}
			if isProd {
				return []string{"https://sms.ztechai.us"}
			}
			return []string{"http://localhost:3000", "https://sms.ztechai.us"}
		}(),
		AllowHTTPLocal: getEnvAsBool("ALLOW_HTTP_LOCAL", defaultAllowHTTP),

		DatabaseURL: func() string {
			if explicit := getEnv("DATABASE_URL", ""); explicit != "" {
				return explicit
			}
			pgUser := getEnv("POSTGRES_USER", getEnv("DB_USER", "zsms_user"))
			pgPass := getEnv("POSTGRES_PASSWORD", getEnv("DB_PASSWORD", "zsms_dev_password"))
			pgHost := getEnv("POSTGRES_HOST", getEnv("DB_HOST", defaultPGHost))
			pgPort := getEnv("POSTGRES_PORT", getEnv("DB_PORT", "5432"))
			pgDB := getEnv("POSTGRES_DB", getEnv("DB_NAME", "zsms_db"))
			if pgPass != "" {
				return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", pgUser, pgPass, pgHost, pgPort, pgDB)
			}
			return fmt.Sprintf("postgres://%s@%s:%s/%s?sslmode=disable", pgUser, pgHost, pgPort, pgDB)
		}(),
		DBMaxOpenConns:       getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:       getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
		DBConnMaxLifetimeMin: getEnvAsInt("DB_CONN_MAX_LIFETIME_MINUTES", 30),

		RedisURL: func() string {
			if explicit := getEnv("REDIS_URL", ""); explicit != "" {
				return explicit
			}
			redisHost := getEnv("REDIS_HOST", defaultRedisHost)
			redisPort := getEnv("REDIS_PORT", "6379")
			redisPass := getEnv("REDIS_PASSWORD", "")
			if redisPass != "" {
				return fmt.Sprintf("redis://:%s@%s:%s/0", redisPass, redisHost, redisPort)
			}
			return fmt.Sprintf("redis://%s:%s/0", redisHost, redisPort)
		}(),
		RedisMaxRetries: getEnvAsInt("REDIS_MAX_RETRIES", 3),

		AuthDisabled:             getEnvAsBool("AUTH_DISABLED", false),
		JWTSecret:                getEnv("JWT_SECRET", "dev_jwt_secret_do_not_use_in_production_min_64_bytes_required_random_string"),
		JWTExpirationHours:       getEnvAsInt("JWT_EXPIRATION_HOURS", 72),
		PairingCodeExpirationMin: getEnvAsInt("PAIRING_CODE_EXPIRATION", 10),
		DeviceTokenPrefix:        getEnv("DEVICE_TOKEN_PREFIX", "zsms_dev_"),
		EncryptionKey:            getEnv("ENCRYPTION_KEY", ""),

		FirebaseProjectID:   getEnv("FIREBASE_PROJECT_ID", getEnv("GCP_PROJECT_ID", "")),
		FirebaseClientEmail: getEnv("FIREBASE_CLIENT_EMAIL", ""),
		FirebasePrivateKey:  getEnv("FIREBASE_PRIVATE_KEY", ""),

		StorageProvider: getEnv("STORAGE_PROVIDER", getEnv("STORAGE_DRIVER", "minio")),
		S3Endpoint:      getEnv("S3_ENDPOINT", "localhost:9000"),
		S3AccessKey:     getEnv("S3_ACCESS_KEY", "zsms_minio_admin"),
		S3SecretKey:     getEnv("S3_SECRET_KEY", "zsms_minio_secret_key"),
		S3Bucket:        getEnv("S3_BUCKET", "zsms-media"),
		S3Region:        getEnv("S3_REGION", "us-east-1"),
		S3UseSSL:        getEnvAsBool("S3_USE_SSL", false),
		R2Endpoint:      getEnv("R2_ENDPOINT", ""),
		R2AccessKey:     getEnv("R2_ACCESS_KEY", ""),
		R2SecretKey:     getEnv("R2_SECRET_KEY", ""),
		R2Bucket:        getEnv("R2_BUCKET", ""),

		WebhookSigningSecret:  getEnv("WEBHOOK_SIGNING_SECRET", "dev_webhook_signing_secret"),
		WebhookTimeoutSeconds: getEnvAsInt("WEBHOOK_TIMEOUT_SECONDS", 10),
		WebhookMaxRetries:     getEnvAsInt("WEBHOOK_MAX_RETRIES", 5),

		SMTPHost:     getEnv("SMTP_HOST", ""),
		SMTPPort:     getEnvAsInt("SMTP_PORT", 587),
		SMTPUsername: getEnv("SMTP_USERNAME", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:     getEnv("SMTP_FROM", getEnv("SMTP_FROM_EMAIL", "no-reply@sms.ztechai.us")),
		SMTPFromName: getEnv("SMTP_FROM_NAME", "ZTechAI ZSMS"),
		SMTPTLS:      getEnvAsBool("SMTP_TLS", true),

		TurnstileSiteKey:   getEnv("TURNSTILE_SITE_KEY", getEnv("CLOUDFLARE_TURNSTILE_SITE_KEY", "")),
		TurnstileSecretKey: getEnv("TURNSTILE_SECRET_KEY", getEnv("CLOUDFLARE_TURNSTILE_SECRET_KEY", "")),

		MaxBulkBatchSize:             getEnvAsInt("MAX_BULK_BATCH_SIZE", 1000),
		DefaultMessageExpiryHours:    getEnvAsInt("DEFAULT_MESSAGE_EXPIRY_HOURS", 24),
		PhoneOfflineThresholdSeconds: getEnvAsInt("PHONE_OFFLINE_THRESHOLD_SECONDS", 120),
	}

	return cfg, cfg.Validate()
}

// Validate checks essential configuration constraints.
func (c *Config) Validate() error {
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("invalid PORT: %d", c.Port)
	}

	if c.AppEnv == "production" {
		if !c.AuthDisabled && (c.JWTSecret == "" || strings.HasPrefix(c.JWTSecret, "dev_")) {
			return errors.New("production environment requires a secure, non-default JWT_SECRET")
		}
		if c.DatabaseURL == "" {
			return errors.New("production environment requires DATABASE_URL")
		}
		if c.RedisURL == "" {
			return errors.New("production environment requires REDIS_URL")
		}
	}

	return nil
}

// MaskedSummary returns a map of configuration parameters with sensitive values redacted.
// This allows logging configuration safely on boot without secret leakage.
func (c *Config) MaskedSummary() map[string]interface{} {
	return map[string]interface{}{
		"app_env":             c.AppEnv,
		"app_name":            c.AppName,
		"app_debug":           c.AppDebug,
		"auth_disabled":       c.AuthDisabled,
		"port":                c.Port,
		"host":                c.Host,
		"app_url":             c.AppURL,
		"api_url":             c.APIURL,
		"web_url":             c.WebURL,
		"storage_provider":    c.StorageProvider,
		"s3_bucket":           c.S3Bucket,
		"database_configured": c.DatabaseURL != "",
		"redis_configured":    c.RedisURL != "",
		"firebase_configured": c.FirebaseProjectID != "" && c.FirebasePrivateKey != "",
		"smtp_configured":     c.SMTPHost != "",
		"turnstile_enabled":   c.TurnstileSiteKey != "",
		"jwt_secret":          "[REDACTED]",
		"db_password":         "[REDACTED]",
		"redis_password":      "[REDACTED]",
		"encryption_key":      "[REDACTED]",
	}
}

// DBConnMaxLifetime returns connection lifetime as a time.Duration.
func (c *Config) DBConnMaxLifetime() time.Duration {
	return time.Duration(c.DBConnMaxLifetimeMin) * time.Minute
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val)
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	valStr := getEnv(key, "")
	if val, err := strconv.Atoi(valStr); err == nil {
		return val
	}
	return defaultVal
}

func getEnvAsBool(key string, defaultVal bool) bool {
	valStr := getEnv(key, "")
	if val, err := strconv.ParseBool(valStr); err == nil {
		return val
	}
	return defaultVal
}

func getEnvAsSlice(key string, defaultVal []string) []string {
	valStr := getEnv(key, "")
	if valStr == "" {
		return defaultVal
	}
	parts := strings.Split(valStr, ",")
	var result []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
