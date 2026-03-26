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
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_USER", "test_user")
	os.Setenv("DB_PASSWORD", "test_password")
	os.Setenv("DB_NAME", "test_db")
	defer func() {
		os.Unsetenv("ANKER_EMAIL")
		os.Unsetenv("ANKER_PASSWORD")
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("DB_NAME")
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

	if cfg.Database.Host != "localhost" {
		t.Errorf("Expected host localhost, got %s", cfg.Database.Host)
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
				Database: DatabaseConfig{
					Host:     "localhost",
					User:     "test_user",
					Password: "test_password",
					Database: "test_db",
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
				Database: DatabaseConfig{
					Host:     "localhost",
					User:     "test_user",
					Password: "test_password",
					Database: "test_db",
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
				Database: DatabaseConfig{
					Host:     "localhost",
					User:     "test_user",
					Password: "test_password",
					Database: "test_db",
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
