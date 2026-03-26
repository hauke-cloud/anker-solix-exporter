package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Set required env vars
	os.Setenv("ANKER_EMAIL", "test@example.com")
	os.Setenv("ANKER_PASSWORD", "password123")
	os.Setenv("INFLUXDB_URL", "http://localhost:8086")
	os.Setenv("INFLUXDB_TOKEN", "token123")
	os.Setenv("INFLUXDB_ORG", "test-org")
	os.Setenv("INFLUXDB_BUCKET", "test-bucket")
	defer func() {
		os.Unsetenv("ANKER_EMAIL")
		os.Unsetenv("ANKER_PASSWORD")
		os.Unsetenv("INFLUXDB_URL")
		os.Unsetenv("INFLUXDB_TOKEN")
		os.Unsetenv("INFLUXDB_ORG")
		os.Unsetenv("INFLUXDB_BUCKET")
	}()

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Anker.Email != "test@example.com" {
		t.Errorf("Expected email test@example.com, got %s", cfg.Anker.Email)
	}

	if cfg.Anker.Country != "DE" {
		t.Errorf("Expected default country DE, got %s", cfg.Anker.Country)
	}

	if cfg.InfluxDB.URL != "http://localhost:8086" {
		t.Errorf("Expected URL http://localhost:8086, got %s", cfg.InfluxDB.URL)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Anker: AnkerConfig{
					Email:        "test@example.com",
					Password:     "password",
					PollInterval: "5m",
				},
				InfluxDB: InfluxDBConfig{
					URL:    "http://localhost:8086",
					Token:  "token",
					Org:    "org",
					Bucket: "bucket",
				},
			},
			wantErr: false,
		},
		{
			name: "missing email",
			config: Config{
				Anker: AnkerConfig{
					Password:     "password",
					PollInterval: "5m",
				},
				InfluxDB: InfluxDBConfig{
					URL:    "http://localhost:8086",
					Token:  "token",
					Org:    "org",
					Bucket: "bucket",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid poll interval",
			config: Config{
				Anker: AnkerConfig{
					Email:        "test@example.com",
					Password:     "password",
					PollInterval: "invalid",
				},
				InfluxDB: InfluxDBConfig{
					URL:    "http://localhost:8086",
					Token:  "token",
					Org:    "org",
					Bucket: "bucket",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetPollInterval(t *testing.T) {
	cfg := Config{
		Anker: AnkerConfig{
			PollInterval: "10m",
		},
	}

	interval := cfg.GetPollInterval()
	expected := 10 * time.Minute

	if interval != expected {
		t.Errorf("Expected interval %v, got %v", expected, interval)
	}
}
