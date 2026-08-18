package config

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server   ServerConfig   `json:"server"`
	Storage  StorageConfig  `json:"storage"`
	Detector DetectorConfig `json:"detector"`
	Security Security      `json:"security"`
	Log      LogConfig      `json:"log"`
	Alert    AlertConfig    `json:"alert"`
}

type ServerConfig struct {
	Host              string `json:"host"`
	Port              int    `json:"port"`
	StaticDir         string `json:"static_dir,omitempty"`
	ReadTimeout       int    `json:"read_timeout_seconds"`
	WriteTimeout      int    `json:"write_timeout_seconds"`
	IdleTimeout       int    `json:"idle_timeout_seconds"`
	MaxHeaderBytes    int    `json:"max_header_bytes"`
	GracefulTimeout   int    `json:"graceful_timeout_seconds"`
}

type StorageConfig struct {
	DBPath          string `json:"db_path"`
	Retention       string `json:"retention"`
	MaxSizeMB       int    `json:"max_size_mb"`
	MaxLogs         int    `json:"max_logs"`
	CleanupInterval int    `json:"cleanup_interval_seconds"`
}

type DetectorConfig struct {
	WindowSize     int     `json:"window_size"`
	ErrorRate      float64 `json:"error_rate_threshold"`
	RateMultiplier float64 `json:"rate_multiplier_threshold"`
	BurstCount     int     `json:"burst_threshold"`
}

type Security struct {
	APIKeys            []string `json:"api_keys"`
	AllowedOrigins     []string `json:"allowed_origins"`
	RateLimitPerMinute int      `json:"rate_limit_per_minute"`
	MaxBodySize        int64    `json:"max_body_size"`
	MaxWSConnections   int      `json:"max_ws_connections"`
	CORSAllowedOrigin  string   `json:"cors_allowed_origin"`
	CORSAllowedHeaders string   `json:"cors_allowed_headers"`
	CORSAllowedMethods string   `json:"cors_allowed_methods"`
}

type LogConfig struct {
	Level  string `json:"level"`
	Format string `json:"format"`
	Output string `json:"output"`
}

type AlertConfig struct {
	Enabled          bool     `json:"enabled"`
	WebhookURL       string   `json:"webhook_url"`
	WebhookHeaders   string   `json:"webhook_headers"`
	CooldownSeconds  int      `json:"cooldown_seconds"`
	Severities       []string `json:"severities"`
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:            "0.0.0.0",
			Port:            8080,
			ReadTimeout:     15,
			WriteTimeout:    15,
			IdleTimeout:     60,
			MaxHeaderBytes:  1 << 20, // 1MB
			GracefulTimeout: 30,
		},
		Storage: StorageConfig{
			DBPath:          "pulse.db",
			Retention:       "7d",
			MaxSizeMB:       500,
			MaxLogs:         1000000,
			CleanupInterval: 3600,
		},
		Detector: DetectorConfig{
			WindowSize:     60,
			ErrorRate:      0.3,
			RateMultiplier: 3.0,
			BurstCount:     100,
		},
		Security: Security{
			APIKeys:            []string{"pulse-dev-key-2024"},
			AllowedOrigins:     []string{"http://localhost:3000", "http://localhost:5173", "http://localhost:8080"},
			RateLimitPerMinute: 100,
			MaxBodySize:        10 * 1024 * 1024, // 10MB
			MaxWSConnections:   100,
			CORSAllowedOrigin:  "*",
			CORSAllowedHeaders: "Content-Type, Authorization, X-API-Key",
			CORSAllowedMethods: "GET, POST, PUT, DELETE, OPTIONS",
		},
		Log: LogConfig{
			Level:  "info",
			Format: "text",
			Output: "stdout",
		},
		Alert: AlertConfig{
			Enabled:         false,
			CooldownSeconds: 300,
			Severities:      []string{"critical", "high"},
		},
	}
}

func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
		} else {
			if err := json.Unmarshal(data, cfg); err != nil {
				return nil, err
			}
		}
	}

	// Override with environment variables
	cfg.loadFromEnv()

	return cfg, nil
}

func (c *Config) loadFromEnv() {
	// Server
	if v := os.Getenv("PULSE_HOST"); v != "" {
		c.Server.Host = v
	}
	if v := os.Getenv("PULSE_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Server.Port = port
		}
	}
	if v := os.Getenv("PULSE_STATIC_DIR"); v != "" {
		c.Server.StaticDir = v
	}
	if v := os.Getenv("PULSE_READ_TIMEOUT"); v != "" {
		if timeout, err := strconv.Atoi(v); err == nil {
			c.Server.ReadTimeout = timeout
		}
	}
	if v := os.Getenv("PULSE_WRITE_TIMEOUT"); v != "" {
		if timeout, err := strconv.Atoi(v); err == nil {
			c.Server.WriteTimeout = timeout
		}
	}
	if v := os.Getenv("PULSE_GRACEFUL_TIMEOUT"); v != "" {
		if timeout, err := strconv.Atoi(v); err == nil {
			c.Server.GracefulTimeout = timeout
		}
	}

	// Storage
	if v := os.Getenv("PULSE_DB_PATH"); v != "" {
		c.Storage.DBPath = v
	}
	if v := os.Getenv("PULSE_RETENTION_HOURS"); v != "" {
		if hours, err := strconv.Atoi(v); err == nil {
			c.Storage.Retention = strconv.Itoa(hours) + "h"
		}
	}
	if v := os.Getenv("PULSE_MAX_LOGS"); v != "" {
		if max, err := strconv.Atoi(v); err == nil {
			c.Storage.MaxLogs = max
		}
	}
	if v := os.Getenv("PULSE_CLEANUP_INTERVAL"); v != "" {
		if interval, err := strconv.Atoi(v); err == nil {
			c.Storage.CleanupInterval = interval
		}
	}

	// Security
	if v := os.Getenv("PULSE_API_KEY"); v != "" {
		c.Security.APIKeys = []string{v}
	}
	if v := os.Getenv("PULSE_ALLOWED_ORIGINS"); v != "" {
		if v == "*" {
			c.Security.AllowedOrigins = []string{"*"}
		} else {
			c.Security.AllowedOrigins = strings.Split(v, ",")
		}
	}
	if v := os.Getenv("PULSE_RATE_LIMIT"); v != "" {
		if limit, err := strconv.Atoi(v); err == nil {
			c.Security.RateLimitPerMinute = limit
		}
	}
	if v := os.Getenv("PULSE_WS_MAX_CONNECTIONS"); v != "" {
		if max, err := strconv.Atoi(v); err == nil {
			c.Security.MaxWSConnections = max
		}
	}
	if v := os.Getenv("PULSE_MAX_BODY_SIZE"); v != "" {
		if size, err := strconv.ParseInt(v, 10, 64); err == nil {
			c.Security.MaxBodySize = size
		}
	}
	if v := os.Getenv("PULSE_CORS_ORIGIN"); v != "" {
		c.Security.CORSAllowedOrigin = v
	}

	// Log
	if v := os.Getenv("PULSE_LOG_LEVEL"); v != "" {
		c.Log.Level = v
	}
	if v := os.Getenv("PULSE_LOG_FORMAT"); v != "" {
		c.Log.Format = v
	}
	if v := os.Getenv("PULSE_LOG_OUTPUT"); v != "" {
		c.Log.Output = v
	}

	// Alert
	if v := os.Getenv("PULSE_ALERT_ENABLED"); v != "" {
		c.Alert.Enabled = v == "true" || v == "1"
	}
	if v := os.Getenv("PULSE_ALERT_WEBHOOK_URL"); v != "" {
		c.Alert.WebhookURL = v
	}
	if v := os.Getenv("PULSE_ALERT_COOLDOWN"); v != "" {
		if cooldown, err := strconv.Atoi(v); err == nil {
			c.Alert.CooldownSeconds = cooldown
		}
	}
}

func (c *Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (c *Config) RetentionDuration() time.Duration {
	d, err := time.ParseDuration(c.Storage.Retention)
	if err != nil {
		return 7 * 24 * time.Hour
	}
	return d
}

func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return ErrInvalid("server.port", c.Server.Port)
	}
	if c.Detector.ErrorRate < 0 || c.Detector.ErrorRate > 1 {
		return ErrInvalid("detector.error_rate_threshold", c.Detector.ErrorRate)
	}
	if c.Detector.RateMultiplier < 1 {
		return ErrInvalid("detector.rate_multiplier_threshold", c.Detector.RateMultiplier)
	}
	if c.Security.RateLimitPerMinute < 1 {
		return ErrInvalid("security.rate_limit_per_minute", c.Security.RateLimitPerMinute)
	}
	return nil
}

// APICheck validates API key
func (c *Config) IsValidAPIKey(key string) bool {
	for _, k := range c.Security.APIKeys {
		if k == key {
			return true
		}
	}
	return false
}

// IsOriginAllowed checks if the origin is allowed
func (c *Config) IsOriginAllowed(origin string) bool {
	for _, o := range c.Security.AllowedOrigins {
		if o == "*" || o == origin {
			return true
		}
	}
	return false
}

// ShouldCORS returns whether CORS should be enabled
func (c *Config) ShouldCORS() bool {
	return c.Security.CORSAllowedOrigin != ""
}

type ConfigError struct {
	Field string
	Value interface{}
}

func (e *ConfigError) Error() string {
	return "invalid config: " + e.Field + " = " + strconv.FormatFloat(toFloat64(e.Value), 'f', -1, 64)
}

func ErrInvalid(field string, value interface{}) *ConfigError {
	return &ConfigError{Field: field, Value: value}
}

func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return 0
	}
}
