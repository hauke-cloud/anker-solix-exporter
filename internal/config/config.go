package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Anker    AnkerConfig    `yaml:"anker"`
	Database DatabaseConfig `yaml:"database"`
	Exporter ExporterConfig `yaml:"exporter"`
}

type AnkerConfig struct {
	Email        string `yaml:"email"`
	Password     string `yaml:"password"`
	Country      string `yaml:"country"`
	PollInterval string `yaml:"poll_interval"`
}

type DatabaseConfig struct {
	Host           string `yaml:"host"`
	Port           int    `yaml:"port"`
	User           string `yaml:"user"`
	Password       string `yaml:"password"`
	Database       string `yaml:"database"`
	SSLMode        string `yaml:"sslmode"`
	MigrationsPath string `yaml:"migrations_path"`
	// TLS/SSL certificate configuration (optional)
	SSLCert       string `yaml:"sslcert"`        // Path to client certificate
	SSLKey        string `yaml:"sslkey"`         // Path to client key
	SSLRootCert   string `yaml:"sslrootcert"`    // Path to CA certificate
}

type DatabaseTLSConfig struct {
	Enabled     bool
	CertFile    string
	KeyFile     string
	RootCertFile string
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
		Database: DatabaseConfig{
			Host:           "localhost",
			Port:           5432,
			SSLMode:        "disable",
			MigrationsPath: "/etc/anker-solix-exporter/migrations",
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
	if v := os.Getenv("DB_HOST"); v != "" {
		config.Database.Host = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		var port int
		if _, err := fmt.Sscanf(v, "%d", &port); err == nil {
			config.Database.Port = port
		}
	}
	if v := os.Getenv("DB_USER"); v != "" {
		config.Database.User = v
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		config.Database.Password = v
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		config.Database.Database = v
	}
	if v := os.Getenv("DB_SSLMODE"); v != "" {
		config.Database.SSLMode = v
	}
	if v := os.Getenv("DB_MIGRATIONS_PATH"); v != "" {
		config.Database.MigrationsPath = v
	}
	if v := os.Getenv("DB_SSLCERT"); v != "" {
		config.Database.SSLCert = v
	}
	if v := os.Getenv("DB_SSLKEY"); v != "" {
		config.Database.SSLKey = v
	}
	if v := os.Getenv("DB_SSLROOTCERT"); v != "" {
		config.Database.SSLRootCert = v
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
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	// When using client certificate authentication, user and password may be derived from the certificate
	usingClientCert := c.Database.SSLCert != "" && c.Database.SSLKey != ""
	
	if !usingClientCert {
		if c.Database.User == "" {
			return fmt.Errorf("database user is required")
		}
		if c.Database.Password == "" {
			return fmt.Errorf("database password is required")
		}
	}
	if c.Database.Database == "" {
		return fmt.Errorf("database name is required")
	}
	
	// Validate SSL certificate configuration
	if c.Database.SSLCert != "" || c.Database.SSLKey != "" || c.Database.SSLRootCert != "" {
		// If any certificate is provided, validate the configuration
		if c.Database.SSLMode == "disable" {
			return fmt.Errorf("sslmode cannot be 'disable' when SSL certificates are configured")
		}
		
		// For client certificate auth, both cert and key are required
		if (c.Database.SSLCert != "" && c.Database.SSLKey == "") || 
		   (c.Database.SSLCert == "" && c.Database.SSLKey != "") {
			return fmt.Errorf("both sslcert and sslkey must be provided for client certificate authentication")
		}
		
		// Validate certificate files exist
		if c.Database.SSLCert != "" {
			if _, err := os.Stat(c.Database.SSLCert); err != nil {
				return fmt.Errorf("SSL certificate file not found: %w", err)
			}
		}
		if c.Database.SSLKey != "" {
			if _, err := os.Stat(c.Database.SSLKey); err != nil {
				return fmt.Errorf("SSL key file not found: %w", err)
			}
		}
		if c.Database.SSLRootCert != "" {
			if _, err := os.Stat(c.Database.SSLRootCert); err != nil {
				return fmt.Errorf("SSL root certificate file not found: %w", err)
			}
		}
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

// GetDSN returns the PostgreSQL DSN connection string
func (c *Config) GetDSN() string {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Database,
		c.Database.SSLMode,
	)
	
	// Add SSL certificate parameters if configured
	if c.Database.SSLCert != "" {
		dsn += fmt.Sprintf(" sslcert=%s", c.Database.SSLCert)
	}
	if c.Database.SSLKey != "" {
		dsn += fmt.Sprintf(" sslkey=%s", c.Database.SSLKey)
	}
	if c.Database.SSLRootCert != "" {
		dsn += fmt.Sprintf(" sslrootcert=%s", c.Database.SSLRootCert)
	}
	
	return dsn
}

// GetTLSConfig returns the TLS configuration if certificates are provided
func (c *Config) GetTLSConfig() *DatabaseTLSConfig {
	if c.Database.SSLCert == "" && c.Database.SSLKey == "" && c.Database.SSLRootCert == "" {
		return &DatabaseTLSConfig{Enabled: false}
	}
	
	return &DatabaseTLSConfig{
		Enabled:      true,
		CertFile:     c.Database.SSLCert,
		KeyFile:      c.Database.SSLKey,
		RootCertFile: c.Database.SSLRootCert,
	}
}
