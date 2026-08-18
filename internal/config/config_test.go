package config

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("host = %s, want 0.0.0.0", cfg.Server.Host)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("port = %d, want 8080", cfg.Server.Port)
	}
	if cfg.Storage.DBPath != "pulse.db" {
		t.Errorf("db path = %s, want pulse.db", cfg.Storage.DBPath)
	}
	if cfg.Detector.ErrorRate != 0.3 {
		t.Errorf("error rate = %f, want 0.3", cfg.Detector.ErrorRate)
	}
}

func TestLoadConfigNonExistent(t *testing.T) {
	cfg, err := LoadConfig("/nonexistent/path.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Error("expected default config for non-existent path")
	}
}

func TestLoadConfigExisting(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "pulse_config_*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	configJSON := `{
		"server": {
			"host": "127.0.0.1",
			"port": 9090
		},
		"detector": {
			"error_rate_threshold": 0.5
		}
	}`

	if _, err := tmpFile.WriteString(configJSON); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	cfg, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("load config failed: %v", err)
	}

	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("host = %s, want 127.0.0.1", cfg.Server.Host)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("port = %d, want 9090", cfg.Server.Port)
	}
	if cfg.Detector.ErrorRate != 0.5 {
		t.Errorf("error rate = %f, want 0.5", cfg.Detector.ErrorRate)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Server.Port = 7777
	cfg.Detector.ErrorRate = 0.42

	tmpFile, err := os.CreateTemp("", "pulse_save_*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	if err := cfg.Save(tmpFile.Name()); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if loaded.Server.Port != 7777 {
		t.Errorf("port = %d, want 7777", loaded.Server.Port)
	}
	if loaded.Detector.ErrorRate != 0.42 {
		t.Errorf("error rate = %f, want 0.42", loaded.Detector.ErrorRate)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  func() *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: func() *Config {
				return DefaultConfig()
			},
			wantErr: false,
		},
		{
			name: "invalid port (too low)",
			config: func() *Config {
				c := DefaultConfig()
				c.Server.Port = 0
				return c
			},
			wantErr: true,
		},
		{
			name: "invalid port (too high)",
			config: func() *Config {
				c := DefaultConfig()
				c.Server.Port = 70000
				return c
			},
			wantErr: true,
		},
		{
			name: "invalid error rate (negative)",
			config: func() *Config {
				c := DefaultConfig()
				c.Detector.ErrorRate = -0.1
				return c
			},
			wantErr: true,
		},
		{
			name: "invalid error rate (over 1)",
			config: func() *Config {
				c := DefaultConfig()
				c.Detector.ErrorRate = 1.5
				return c
			},
			wantErr: true,
		},
		{
			name: "invalid rate multiplier",
			config: func() *Config {
				c := DefaultConfig()
				c.Detector.RateMultiplier = 0.5
				return c
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config().Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRetentionDuration(t *testing.T) {
	cfg := DefaultConfig()

	d := cfg.RetentionDuration()
	expected := 7 * 24 * time.Hour
	if d != expected {
		t.Errorf("retention = %v, want %v", d, expected)
	}

	cfg.Storage.Retention = "24h"
	d = cfg.RetentionDuration()
	if d != 24*time.Hour {
		t.Errorf("retention = %v, want 24h", d)
	}

	cfg.Storage.Retention = "invalid"
	d = cfg.RetentionDuration()
	if d != expected {
		t.Errorf("default retention for invalid = %v, want %v", d, expected)
	}
}

func TestConfigError(t *testing.T) {
	err := ErrInvalid("port", 70000)
	expected := "invalid config: port = 70000"
	if err.Error() != expected {
		t.Errorf("error = %q, want %q", err.Error(), expected)
	}
}
