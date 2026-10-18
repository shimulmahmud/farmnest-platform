package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoad_Defaults(t *testing.T) {
	os.Clearenv()
	os.Setenv("JWT_SECRET", "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4") // 32-char minimum
	defer os.Clearenv()
	cfg, err := Load()
	assert.NoError(t, err)
	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "mongodb://localhost:27017", cfg.MongoDB.URI)
	assert.Equal(t, "ecommerce", cfg.MongoDB.Database)
	assert.Equal(t, "nats://localhost:4222", cfg.NATS.URL)
	assert.Equal(t, uint64(100), cfg.MongoDB.MaxPoolSize)
	assert.Equal(t, uint64(5), cfg.MongoDB.MinPoolSize)
	assert.Equal(t, 15*time.Second, cfg.Server.ReadTimeout)
	assert.Equal(t, 100, cfg.Security.RateLimitPerMinute)
	assert.Equal(t, 7, len(cfg.NATSKV.Buckets))
	assert.Equal(t, []string{"http://localhost:3000"}, cfg.Security.CORSAllowedOrigins)
	assert.Equal(t, []string{"Content-Type", "Authorization", "X-Request-ID"}, cfg.Security.CORSAllowedHeaders)
}

func TestLoad_EnvOverrides(t *testing.T) {
	os.Clearenv()
	os.Setenv("JWT_SECRET", "super-secret-key-that-is-long-enough")
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("MONGODB_URI", "mongodb://prod:27017")
	os.Setenv("MONGODB_DATABASE", "ecommerce-prod")
	os.Setenv("NATS_URL", "nats://nats-prod:4222")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("CORS_ALLOWED_ORIGINS", "https://example.com,https://app.example.com")
	defer os.Clearenv()

	cfg, err := Load()
	assert.NoError(t, err)
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, "mongodb://prod:27017", cfg.MongoDB.URI)
	assert.Equal(t, "ecommerce-prod", cfg.MongoDB.Database)
	assert.Equal(t, "nats://nats-prod:4222", cfg.NATS.URL)
	assert.Equal(t, "debug", cfg.Logging.Level)
	assert.Equal(t, "super-secret-key-that-is-long-enough", cfg.JWT.Secret)
	assert.Equal(t, []string{"https://example.com", "https://app.example.com"}, cfg.Security.CORSAllowedOrigins)
}

func TestLoad_Validation_InvalidPort(t *testing.T) {
	os.Clearenv()
	os.Setenv("JWT_SECRET", "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4")
	os.Setenv("SERVER_PORT", "99999")
	defer os.Clearenv()

	_, err := Load()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid server port")
}

func TestLoad_Validation_MissingJWTSecret(t *testing.T) {
	os.Clearenv()
	defer os.Clearenv()

	_, err := Load()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET environment variable is required")
}

func TestLoad_Validation_ShortJWTSecret(t *testing.T) {
	os.Clearenv()
	os.Setenv("JWT_SECRET", "too-short")
	defer os.Clearenv()

	_, err := Load()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least 32 characters")
}

func TestServerAddr(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Host: "127.0.0.1", Port: 3000},
	}
	assert.Equal(t, "127.0.0.1:3000", cfg.ServerAddr())
}

func TestGetEnvHelpers(t *testing.T) {
	os.Clearenv()
	// getEnv
	assert.Equal(t, "default", getEnv("MISSING_KEY", "default"))
	os.Setenv("TEST_KEY", "value")
	assert.Equal(t, "value", getEnv("TEST_KEY", "default"))

	// getEnvInt
	assert.Equal(t, 42, getEnvInt("MISSING_INT", 42))
	os.Setenv("TEST_INT", "123")
	assert.Equal(t, 123, getEnvInt("TEST_INT", 42))

	// getEnvBool
	assert.True(t, getEnvBool("MISSING_BOOL", true))
	os.Setenv("TEST_BOOL", "false")
	assert.False(t, getEnvBool("TEST_BOOL", true))

	// getEnvDuration
	assert.Equal(t, 5*time.Second, getEnvDuration("MISSING_DUR", 5*time.Second))
	os.Setenv("TEST_DUR", "30s")
	assert.Equal(t, 30*time.Second, getEnvDuration("TEST_DUR", 5*time.Second))

	// getEnvStringSlice
	slice := getEnvStringSlice("MISSING_SLICE", "a,b,c")
	assert.Equal(t, []string{"a", "b", "c"}, slice)
	os.Setenv("TEST_SLICE", "x,y")
	slice = getEnvStringSlice("TEST_SLICE", "a,b,c")
	assert.Equal(t, []string{"x", "y"}, slice)

	defer os.Clearenv()
}
