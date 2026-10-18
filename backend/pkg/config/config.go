package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Build-time version information. Injected via -ldflags at Docker build time.
var (
        Version   = "dev"
        Commit    = "none"
        BuildTime = "none"
)

// Config holds all application configuration.
type Config struct {
        Server        ServerConfig        `mapstructure:"server"`
        MongoDB       MongoDBConfig       `mapstructure:"mongodb"`
        NATS          NATSConfig          `mapstructure:"nats"`
        NATSKV        NATSKVConfig        `mapstructure:"nats_kv"`
        Logging       LoggingConfig       `mapstructure:"logging"`
        Security      SecurityConfig      `mapstructure:"security"`
        JWT           JWTConfig           `mapstructure:"jwt"`
        Email         EmailConfig         `mapstructure:"email"`
        Observability ObservabilityConfig `mapstructure:"observability"`
}

type ServerConfig struct {
        Host         string        `mapstructure:"host"`
        Port         int           `mapstructure:"port"`
        ReadTimeout  time.Duration `mapstructure:"read_timeout"`
        WriteTimeout time.Duration `mapstructure:"write_timeout"`
        IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

type MongoDBConfig struct {
        URI            string        `mapstructure:"uri"`
        Database       string        `mapstructure:"database"`
        MaxPoolSize    uint64        `mapstructure:"max_pool_size"`
        MinPoolSize    uint64        `mapstructure:"min_pool_size"`
        ConnectTimeout time.Duration `mapstructure:"connect_timeout"`
}

type NATSConfig struct {
        URL            string        `mapstructure:"url"`
        ClusterName    string        `mapstructure:"cluster_name"`
        ReconnectWait  time.Duration `mapstructure:"reconnect_wait"`
        MaxReconnects  int           `mapstructure:"max_reconnects"`
        ConnectTimeout time.Duration `mapstructure:"connect_timeout"`
}

type NATSKVConfig struct {
        Buckets []KVBucketConfig `mapstructure:"buckets"`
}

type KVBucketConfig struct {
        Name    string `mapstructure:"name"`
        History uint8  `mapstructure:"history"`
}

type LoggingConfig struct {
        Level    string `mapstructure:"level"`
        Encoding string `mapstructure:"encoding"`
        Output   string `mapstructure:"output"`
}

type SecurityConfig struct {
        CORSAllowedOrigins []string `mapstructure:"cors_allowed_origins"`
        CORSAllowedMethods []string `mapstructure:"cors_allowed_methods"`
        CORSAllowedHeaders []string `mapstructure:"cors_allowed_headers"`
        RateLimitPerMinute int      `mapstructure:"rate_limit_per_minute"`
}

type JWTConfig struct {
        Secret     string        `mapstructure:"secret"`
        AccessTTL  time.Duration `mapstructure:"access_ttl"`
        RefreshTTL time.Duration `mapstructure:"refresh_ttl"`
        Issuer     string        `mapstructure:"issuer"`
}

type EmailConfig struct {
        SMTPHost     string        `mapstructure:"smtp_host"`
        SMTPPort     int           `mapstructure:"smtp_port"`
        SMTPUsername string        `mapstructure:"smtp_username"`
        SMTPPassword string        `mapstructure:"smtp_password"`
        SMTPFrom     string        `mapstructure:"smtp_from"`
        SMTPStartTLS bool          `mapstructure:"smtp_starttls"`
        SMTPTimeout  time.Duration `mapstructure:"smtp_timeout"`
}

type ObservabilityConfig struct {
        ServiceName    string `mapstructure:"service_name"`
        Enabled        bool   `mapstructure:"enabled"`
        PrometheusPort int    `mapstructure:"prometheus_port"`
}

// Load reads configuration from environment variables with sensible defaults.
// Environment variables take precedence for secrets and deployment-specific values.
func Load() (*Config, error) {
        cfg := &Config{
                Server: ServerConfig{
                        Host:         getEnv("SERVER_HOST", "0.0.0.0"),
                        Port:         getEnvInt("SERVER_PORT", 8080),
                        ReadTimeout:  getEnvDuration("SERVER_READ_TIMEOUT", 15*time.Second),
                        WriteTimeout: getEnvDuration("SERVER_WRITE_TIMEOUT", 15*time.Second),
                        IdleTimeout:  getEnvDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
                },
                MongoDB: MongoDBConfig{
                        URI:            getEnv("MONGODB_URI", "mongodb://localhost:27017"),
                        Database:       getEnv("MONGODB_DATABASE", "ecommerce"),
                        MaxPoolSize:    uint64(getEnvInt("MONGODB_MAX_POOL_SIZE", 100)),
                        MinPoolSize:    uint64(getEnvInt("MONGODB_MIN_POOL_SIZE", 5)),
                        ConnectTimeout: getEnvDuration("MONGODB_CONNECT_TIMEOUT", 10*time.Second),
                },
                NATS: NATSConfig{
                        URL:            getEnv("NATS_URL", "nats://localhost:4222"),
                        ClusterName:    getEnv("NATS_CLUSTER_NAME", "ecommerce-cluster"),
                        ReconnectWait:  getEnvDuration("NATS_RECONNECT_WAIT", 2*time.Second),
                        MaxReconnects:  getEnvInt("NATS_MAX_RECONNECTS", 5),
                        ConnectTimeout: getEnvDuration("NATS_CONNECT_TIMEOUT", 5*time.Second),
                },
                NATSKV: NATSKVConfig{
                        Buckets: []KVBucketConfig{
                                {Name: "PRODUCT_CACHE", History: 1},
                                {Name: "CATEGORY_CACHE", History: 1},
                                {Name: "SESSION_STATE", History: 0},
                                {Name: "IDEMPOTENCY", History: 0},
                                {Name: "RATE_LIMIT", History: 0},
                                {Name: "CHECKOUT_STATE", History: 0},
                                {Name: "DISTRIBUTED_LOCKS", History: 0},
                        },
                },
                Logging: LoggingConfig{
                        Level:    getEnv("LOG_LEVEL", "info"),
                        Encoding: getEnv("LOG_ENCODING", "json"),
                        Output:   getEnv("LOG_OUTPUT", "stdout"),
                },
                Security: SecurityConfig{
                        CORSAllowedOrigins: getEnvStringSlice("CORS_ALLOWED_ORIGINS", "http://localhost:3000"),
                        CORSAllowedMethods: getEnvStringSlice("CORS_ALLOWED_METHODS", "GET,POST,PUT,PATCH,DELETE,OPTIONS"),
                        CORSAllowedHeaders: getEnvStringSlice("CORS_ALLOWED_HEADERS", "Content-Type,Authorization,X-Request-ID"),
                        RateLimitPerMinute: getEnvInt("RATE_LIMIT_PER_MINUTE", 100),
                },
                JWT: JWTConfig{
                        Secret:     os.Getenv("JWT_SECRET"),
                        AccessTTL:  getEnvDuration("JWT_ACCESS_TTL", 15*time.Minute),
                        RefreshTTL: getEnvDuration("JWT_REFRESH_TTL", 168*time.Hour),
                        Issuer:     getEnv("JWT_ISSUER", "ecommerce-api"),
                },
                Email: EmailConfig{
                        SMTPHost:     getEnv("SMTP_HOST", "localhost"),
                        SMTPPort:     getEnvInt("SMTP_PORT", 1025),
                        SMTPUsername: os.Getenv("SMTP_USERNAME"),
                        SMTPPassword: os.Getenv("SMTP_PASSWORD"),
                        SMTPFrom:     getEnv("SMTP_FROM", "no-reply@ecommerce.local"),
                        SMTPStartTLS: getEnvBool("SMTP_STARTTLS", false),
                        SMTPTimeout:  getEnvDuration("SMTP_TIMEOUT", 10*time.Second),
                },
                Observability: ObservabilityConfig{
                        ServiceName:    getEnv("OTEL_SERVICE_NAME", "ecommerce-api"),
                        Enabled:        getEnvBool("OTEL_ENABLED", true),
                        PrometheusPort: getEnvInt("OTEL_PROMETHEUS_PORT", 9090),
                },
        }

        if err := cfg.validate(); err != nil {
                return nil, fmt.Errorf("config validation failed: %w", err)
        }
        return cfg, nil
}

func (c *Config) validate() error {
        if c.Server.Port <= 0 || c.Server.Port > 65535 {
                return fmt.Errorf("invalid server port: %d", c.Server.Port)
        }
        if c.MongoDB.URI == "" {
                return fmt.Errorf("mongodb URI is required")
        }
        if c.MongoDB.Database == "" {
                return fmt.Errorf("mongodb database name is required")
        }
        if c.NATS.URL == "" {
                return fmt.Errorf("NATS URL is required")
        }
        if c.JWT.Secret == "" {
                return fmt.Errorf("JWT_SECRET environment variable is required (must be at least 32 characters)")
        }
        if len(c.JWT.Secret) < 32 {
                return fmt.Errorf("JWT_SECRET must be at least 32 characters long")
        }
        if c.Email.SMTPHost == "" {
                return fmt.Errorf("SMTP_HOST is required")
        }
        if c.Email.SMTPPort <= 0 || c.Email.SMTPPort > 65535 {
                return fmt.Errorf("invalid SMTP_PORT: %d", c.Email.SMTPPort)
        }
        return nil
}

// ServerAddr returns the host:port string for the HTTP server.
func (c *Config) ServerAddr() string {
        return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

func getEnv(key, fallback string) string {
        if v := os.Getenv(key); v != "" {
                return v
        }
        return fallback
}

func getEnvInt(key string, fallback int) int {
        if v := os.Getenv(key); v != "" {
                if i, err := strconv.Atoi(v); err == nil {
                        return i
                }
        }
        return fallback
}

func getEnvBool(key string, fallback bool) bool {
        if v := os.Getenv(key); v != "" {
                b, err := strconv.ParseBool(v)
                if err == nil {
                        return b
                }
        }
        return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
        if v := os.Getenv(key); v != "" {
                d, err := time.ParseDuration(v)
                if err == nil {
                        return d
                }
        }
        return fallback
}

func getEnvStringSlice(key, fallback string) []string {
        if v := os.Getenv(key); v != "" {
                return strings.Split(v, ",")
        }
        return strings.Split(fallback, ",")
}
