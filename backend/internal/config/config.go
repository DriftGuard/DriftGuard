package config

import (
	"fmt"
	"os"
	"time"

	"github.com/DriftGuard/core/pkg/models"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server       ServerConfig         `yaml:"server"`
	Database     DatabaseConfig       `yaml:"database"`
	Kubernetes   KubernetesConfig     `yaml:"kubernetes"`
	Git          GitConfig            `yaml:"git"`
	Environments []models.Environment `yaml:"environments"`
	Logging      LoggingConfig        `yaml:"logging"`
}

type ServerConfig struct {
	Port            int           `yaml:"port"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	IdleTimeout     time.Duration `yaml:"idle_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	DBName   string `yaml:"dbname"`
	Password string `yaml:"password"`
}

type KubernetesConfig struct {
	ConfigPath string            `yaml:"config_path"`
	Context    string            `yaml:"context"`
	Namespaces []string          `yaml:"namespaces"`
	Resources  []string          `yaml:"resources"`
	Labels     map[string]string `yaml:"labels"`
}

type GitConfig struct {
	RepositoryURL string        `yaml:"repository_url"`
	DefaultBranch string        `yaml:"default_branch"`
	CloneTimeout  time.Duration `yaml:"clone_timeout"`
	PullTimeout   time.Duration `yaml:"pull_timeout"`
}

type LoggingConfig struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"`
	OutputPath string `yaml:"output_path"`
}

func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	config.setDefaults()

	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

func (c *Config) setDefaults() {
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

	if c.Database.Port == 0 {
		c.Database.Port = 27017
	}

	if c.Kubernetes.ConfigPath == "" {
		c.Kubernetes.ConfigPath = os.Getenv("KUBECONFIG")
	}
	if len(c.Kubernetes.Resources) == 0 {
		c.Kubernetes.Resources = []string{"deployments", "services", "configmaps", "secrets"}
	}

	if c.Git.DefaultBranch == "" {
		c.Git.DefaultBranch = "main"
	}
	if c.Git.CloneTimeout == 0 {
		c.Git.CloneTimeout = 5 * time.Minute
	}
	if c.Git.PullTimeout == 0 {
		c.Git.PullTimeout = 2 * time.Minute
	}

	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "json"
	}
}

func (c *Config) validate() error {
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Database.DBName == "" {
		return fmt.Errorf("database name is required")
	}

	return nil
}

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

func (c *Config) GetEnvironmentByName(name string) (*models.Environment, error) {
	for _, env := range c.Environments {
		if env.Name == name {
			return &env, nil
		}
	}
	return nil, fmt.Errorf("environment '%s' not found", name)
}
