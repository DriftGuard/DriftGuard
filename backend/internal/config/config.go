package config

import (
	"fmt"
	"os"
	"time"

	"github.com/DriftGuard/core/pkg/models"
	"gopkg.in/yaml.v3"
)

// Config represents the main application configuration
type Config struct {
	Server       ServerConfig         `yaml:"server"`
	Database     DatabaseConfig       `yaml:"database"`
	Kubernetes   KubernetesConfig     `yaml:"kubernetes"`
	Git          GitConfig            `yaml:"git"`
	MCP          MCPConfig            `yaml:"mcp"`
	Environments []models.Environment `yaml:"environments"`
	Logging      LoggingConfig        `yaml:"logging"`
}

// ServerConfig represents HTTP server configuration
type ServerConfig struct {
	Port            int           `yaml:"port"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	IdleTimeout     time.Duration `yaml:"idle_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	DBName   string `yaml:"dbname"`
	Password string `yaml:"password"`
}

// KubernetesConfig represents Kubernetes client configuration
type KubernetesConfig struct {
	ConfigPath       string            `yaml:"config_path"`
	Context          string            `yaml:"context"`
	Namespaces       []string          `yaml:"namespaces"`
	Resources        []string          `yaml:"resources"`
	Labels           map[string]string `yaml:"labels"`
	EnableSnapshots  bool              `yaml:"enable_snapshots"`
	SnapshotInterval time.Duration     `yaml:"snapshot_interval"`
	SkipSystemNS     bool              `yaml:"skip_system_namespaces"`
	SkipFrequentPods bool              `yaml:"skip_frequent_pods"`
}

// GitConfig represents Git repository configuration
type GitConfig struct {
	DefaultBranch string        `yaml:"default_branch"`
	CloneTimeout  time.Duration `yaml:"clone_timeout"`
	PullTimeout   time.Duration `yaml:"pull_timeout"`
}

// MCPConfig represents Model Context Protocol configuration
type MCPConfig struct {
	Endpoint string        `yaml:"endpoint"`
	Timeout  time.Duration `yaml:"timeout"`
	Retries  int           `yaml:"retries"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"`
	OutputPath string `yaml:"output_path"`
}

// Load loads configuration from a YAML file
func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	config.setDefaults()

	// Validate configuration
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// setDefaults sets default values for configuration fields
func (c *Config) setDefaults() {
	// Server defaults
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = 30 * time.Second
	}
	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = 30 * time.Second
	}
	if c.Server.IdleTimeout == 0 {
		c.Server.IdleTimeout = 60 * time.Second
	}
	if c.Server.ShutdownTimeout == 0 {
		c.Server.ShutdownTimeout = 30 * time.Second
	}

	// Database defaults - MongoDB
	if c.Database.Port == 0 {
		c.Database.Port = 27017
	}

	// Kubernetes defaults
	if c.Kubernetes.ConfigPath == "" {
		c.Kubernetes.ConfigPath = os.Getenv("KUBECONFIG")
	}
	if len(c.Kubernetes.Resources) == 0 {
		c.Kubernetes.Resources = []string{"deployments", "services", "configmaps", "secrets"}
	}

	// Git defaults
	if c.Git.DefaultBranch == "" {
		c.Git.DefaultBranch = "main"
	}
	if c.Git.CloneTimeout == 0 {
		c.Git.CloneTimeout = 5 * time.Minute
	}
	if c.Git.PullTimeout == 0 {
		c.Git.PullTimeout = 2 * time.Minute
	}

	// MCP defaults
	if c.MCP.Endpoint == "" {
		c.MCP.Endpoint = "localhost:9000"
	}
	if c.MCP.Timeout == 0 {
		c.MCP.Timeout = 30 * time.Second
	}
	if c.MCP.Retries == 0 {
		c.MCP.Retries = 3
	}

	// Logging defaults
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "json"
	}
}

// validate validates the configuration
func (c *Config) validate() error {
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Database.DBName == "" {
		return fmt.Errorf("database name is required")
	}

	if len(c.Environments) == 0 {
		return fmt.Errorf("at least one environment must be configured")
	}

	for i, env := range c.Environments {
		if env.Name == "" {
			return fmt.Errorf("environment %d: name is required", i)
		}
		if env.GitRepo.URL == "" {
			return fmt.Errorf("environment %s: git repository URL is required", env.Name)
		}
	}

	return nil
}

// GetDatabaseURL returns the database connection URL
func (c *Config) GetDatabaseURL() string {
	if c.Database.User != "" && c.Database.Password != "" {
		return fmt.Sprintf("mongodb://%s:%s@%s:%d/%s",
			c.Database.User,
			c.Database.Password,
			c.Database.Host,
			c.Database.Port,
			c.Database.DBName,
		)
	}
	return fmt.Sprintf("mongodb://%s:%d/%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.DBName,
	)
}

// GetEnvironmentByName returns an environment by name
func (c *Config) GetEnvironmentByName(name string) (*models.Environment, error) {
	for _, env := range c.Environments {
		if env.Name == name {
			return &env, nil
		}
	}
	return nil, fmt.Errorf("environment '%s' not found", name)
}
