package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Anker    AnkerConfig    `mapstructure:"anker"`
	Database DatabaseConfig `mapstructure:"database"`
	Exporter ExporterConfig `mapstructure:"exporter"`
}

type AnkerConfig struct {
	Email         string  `mapstructure:"email"`
	Password      string  `mapstructure:"password"`
	Country       string  `mapstructure:"country"`
	PollInterval  string  `mapstructure:"poll_interval"`
	Debug         bool    `mapstructure:"debug"`
	EndpointLimit int     `mapstructure:"endpoint_limit"`
	RequestDelay  float64 `mapstructure:"request_delay"`
}

type DatabaseConfig struct {
	Host           string `mapstructure:"host"`
	Port           int    `mapstructure:"port"`
	User           string `mapstructure:"user"`
	Password       string `mapstructure:"password"`
	Database       string `mapstructure:"database"`
	SSLMode        string `mapstructure:"sslmode"`
	MigrationsPath string `mapstructure:"migrations_path"`
	// TLS/SSL certificate configuration (optional)
	SSLCert     string `mapstructure:"sslcert"`
	SSLKey      string `mapstructure:"sslkey"`
	SSLRootCert string `mapstructure:"sslrootcert"`
}

type DatabaseTLSConfig struct {
	Enabled      bool
	CertFile     string
	KeyFile      string
	RootCertFile string
}

type ExporterConfig struct {
	ResumeFile string `mapstructure:"resume_file"`
	LogLevel   string `mapstructure:"log_level"`
}

// LoadConfig loads configuration from file and environment variables using Viper
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Set default values for all configuration options
	setDefaults(v)

	// Configure Viper
	v.SetConfigType("yaml")
	
	// If a config path is provided, use it
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// Search for config in multiple paths
		v.SetConfigName("config")
		v.AddConfigPath("/etc/anker-solix-exporter/")
		v.AddConfigPath("$HOME/.anker-solix-exporter")
		v.AddConfigPath(".")
	}

	// Read config file (optional - will use defaults if not found)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file was found but another error was produced
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found; will use defaults and environment variables
	}

	// Enable environment variable override
	v.SetEnvPrefix("") // No prefix, use exact names
	v.AutomaticEnv()
	
	// Replace dots and dashes in env var names with underscores
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	// Bind specific environment variables with custom names
	bindEnvVariables(v)

	// Unmarshal config into struct
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// setDefaults sets all default configuration values
func setDefaults(v *viper.Viper) {
	// Determine default migrations path
	migrationsPath := detectMigrationsPath()
	
	// Determine default resume file path
	resumeFile := detectResumeFilePath()

	// Anker defaults
	v.SetDefault("anker.country", "DE")
	v.SetDefault("anker.poll_interval", "15s")
	v.SetDefault("anker.debug", false)
	v.SetDefault("anker.endpoint_limit", 10)
	v.SetDefault("anker.request_delay", 0.3)

	// Database defaults
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.migrations_path", migrationsPath)

	// Exporter defaults
	v.SetDefault("exporter.resume_file", resumeFile)
	v.SetDefault("exporter.log_level", "info")
}

// bindEnvVariables binds environment variables to config keys
func bindEnvVariables(v *viper.Viper) {
	// Anker configuration
	v.BindEnv("anker.email", "ANKER_EMAIL")
	v.BindEnv("anker.password", "ANKER_PASSWORD")
	v.BindEnv("anker.country", "ANKER_COUNTRY")
	v.BindEnv("anker.poll_interval", "ANKER_POLL_INTERVAL")
	v.BindEnv("anker.debug", "ANKER_DEBUG")
	v.BindEnv("anker.endpoint_limit", "ANKER_ENDPOINT_LIMIT")
	v.BindEnv("anker.request_delay", "ANKER_REQUEST_DELAY")

	// Database configuration
	v.BindEnv("database.host", "DB_HOST")
	v.BindEnv("database.port", "DB_PORT")
	v.BindEnv("database.user", "DB_USER")
	v.BindEnv("database.password", "DB_PASSWORD")
	v.BindEnv("database.database", "DB_NAME")
	v.BindEnv("database.sslmode", "DB_SSLMODE")
	v.BindEnv("database.migrations_path", "DB_MIGRATIONS_PATH")
	v.BindEnv("database.sslcert", "DB_SSLCERT")
	v.BindEnv("database.sslkey", "DB_SSLKEY")
	v.BindEnv("database.sslrootcert", "DB_SSLROOTCERT")

	// Exporter configuration
	v.BindEnv("exporter.resume_file", "RESUME_FILE")
	v.BindEnv("exporter.log_level", "LOG_LEVEL")
}

// detectMigrationsPath determines the default migrations path
func detectMigrationsPath() string {
	// Try container path first
	containerPath := "/etc/anker-solix-exporter/migrations"
	if _, err := os.Stat(containerPath); err == nil {
		return containerPath
	}

	// Try relative path for local development
	if _, err := os.Stat("migrations"); err == nil {
		return "migrations"
	}
	
	if _, err := os.Stat("./migrations"); err == nil {
		return "./migrations"
	}

	// Try to find migrations relative to executable
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		migrationsPath := filepath.Join(execDir, "migrations")
		if _, err := os.Stat(migrationsPath); err == nil {
			return migrationsPath
		}
	}

	// Fallback to container path
	return containerPath
}

// detectResumeFilePath determines the default resume file path
func detectResumeFilePath() string {
	// Use /data for container environment
	containerPath := "/data/last_export.json"
	if _, err := os.Stat("/data"); err == nil {
		return containerPath
	}

	// Local development - use current directory
	return "last_export.json"
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
	dsn := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Database,
		c.Database.SSLMode,
	)

	// Only add password if it's set (not needed for certificate auth)
	if c.Database.Password != "" {
		dsn += fmt.Sprintf(" password=%s", c.Database.Password)
	}

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
