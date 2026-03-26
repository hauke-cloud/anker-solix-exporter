package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Anker    AnkerConfig    `yaml:"anker"`
	InfluxDB InfluxDBConfig `yaml:"influxdb"`
	Exporter ExporterConfig `yaml:"exporter"`
}

type AnkerConfig struct {
	Email        string `yaml:"email"`
	Password     string `yaml:"password"`
	Country      string `yaml:"country"`
	PollInterval string `yaml:"poll_interval"`
}

type InfluxDBConfig struct {
	URL          string `yaml:"url"`
	Token        string `yaml:"token"`
	Org          string `yaml:"org"`
	Bucket       string `yaml:"bucket"`
	Measurement  string `yaml:"measurement"`
}

type ExporterConfig struct {
	ResumeFile   string `yaml:"resume_file"`
	LogLevel     string `yaml:"log_level"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	config := &Config{
		Anker: AnkerConfig{
			Country:      "DE",
			PollInterval: "5m",
		},
		InfluxDB: InfluxDBConfig{
			Measurement: "solix_energy",
		},
		Exporter: ExporterConfig{
			ResumeFile: "/data/last_export.json",
			LogLevel:   "info",
		},
	}

	// Load from file if exists
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		if err == nil {
			if err := yaml.Unmarshal(data, config); err != nil {
				return nil, fmt.Errorf("failed to parse config file: %w", err)
			}
		}
	}

	// Override with environment variables
	if v := os.Getenv("ANKER_EMAIL"); v != "" {
		config.Anker.Email = v
	}
	if v := os.Getenv("ANKER_PASSWORD"); v != "" {
		config.Anker.Password = v
	}
	if v := os.Getenv("ANKER_COUNTRY"); v != "" {
		config.Anker.Country = v
	}
	if v := os.Getenv("ANKER_POLL_INTERVAL"); v != "" {
		config.Anker.PollInterval = v
	}
	if v := os.Getenv("INFLUXDB_URL"); v != "" {
		config.InfluxDB.URL = v
	}
	if v := os.Getenv("INFLUXDB_TOKEN"); v != "" {
		config.InfluxDB.Token = v
	}
	if v := os.Getenv("INFLUXDB_ORG"); v != "" {
		config.InfluxDB.Org = v
	}
	if v := os.Getenv("INFLUXDB_BUCKET"); v != "" {
		config.InfluxDB.Bucket = v
	}
	if v := os.Getenv("INFLUXDB_MEASUREMENT"); v != "" {
		config.InfluxDB.Measurement = v
	}
	if v := os.Getenv("RESUME_FILE"); v != "" {
		config.Exporter.ResumeFile = v
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		config.Exporter.LogLevel = v
	}

	// Validate
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

func (c *Config) Validate() error {
	if c.Anker.Email == "" {
		return fmt.Errorf("anker email is required")
	}
	if c.Anker.Password == "" {
		return fmt.Errorf("anker password is required")
	}
	if c.InfluxDB.URL == "" {
		return fmt.Errorf("influxdb url is required")
	}
	if c.InfluxDB.Token == "" {
		return fmt.Errorf("influxdb token is required")
	}
	if c.InfluxDB.Org == "" {
		return fmt.Errorf("influxdb org is required")
	}
	if c.InfluxDB.Bucket == "" {
		return fmt.Errorf("influxdb bucket is required")
	}
	
	// Validate poll interval
	if _, err := time.ParseDuration(c.Anker.PollInterval); err != nil {
		return fmt.Errorf("invalid poll interval: %w", err)
	}

	return nil
}

func (c *Config) GetPollInterval() time.Duration {
	d, _ := time.ParseDuration(c.Anker.PollInterval)
	return d
}
