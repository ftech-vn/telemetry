package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	CheckInterval   string   `yaml:"check_interval"`
	ServerName      string   `yaml:"server_name"`
	DiskThreshold   *float64 `yaml:"disk_threshold"`
	ExcludedDirs    []string `yaml:"excluded_dirs"`
	CPUThreshold    *float64 `yaml:"cpu_threshold"`
	MemoryThreshold *float64 `yaml:"memory_threshold"`
	HealthChecks    []string `yaml:"health_checks"`
	DBChecks        []string `yaml:"db_checks"`
	LarkWebhookURL  string   `yaml:"lark_webhook_url"`
	WebhookURL      string   `yaml:"webhook_url"`
	WebhookInterval string   `yaml:"webhook_interval"`
	AutoUpdate      bool     `yaml:"auto_update"`
	ServerID        string   `yaml:"server_id"`
	ServerKey       string   `yaml:"server_key"`
	GeminiAPIKey    string   `yaml:"gemini_api_key"`
}

func Load() (*Config, error) {
	// Load from YAML config (required)
	cfg, err := loadFromYAML()
	if err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

func loadFromYAML() (*Config, error) {
	// Try to load from ~/.telemetry/config.yaml
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not determine home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".telemetry", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("could not read config file at %s: %w (run install script to create it)", configPath, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid YAML in config file: %w", err)
	}

	// Apply defaults for empty values
	if cfg.CheckInterval == "" {
		cfg.CheckInterval = "60s"
	}
	if cfg.WebhookInterval == "" {
		cfg.WebhookInterval = "1s"
	}
	if cfg.ServerName == "" {
		cfg.ServerName = "unknown"
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	// Validate check_interval
	interval, err := time.ParseDuration(c.CheckInterval)
	if err != nil {
		return fmt.Errorf("invalid check_interval '%s': %w", c.CheckInterval, err)
	}
	if interval < 1*time.Second {
		return fmt.Errorf("check_interval must be at least 1 second, got %v", interval)
	}
	if interval > 24*time.Hour {
		return fmt.Errorf("check_interval must be at most 24 hours, got %v", interval)
	}

	// Validate webhook_interval
	if c.WebhookURL != "" {
		interval, err = time.ParseDuration(c.WebhookInterval)
		if err != nil {
			return fmt.Errorf("invalid webhook_interval '%s': %w", c.WebhookInterval, err)
		}
		if interval < 1*time.Second {
			return fmt.Errorf("webhook_interval must be at least 1 second, got %v", interval)
		}
		if interval > 24*time.Hour {
			return fmt.Errorf("webhook_interval must be at most 24 hours, got %v", interval)
		}
	}

	// Validate that at least one notification method is configured
	if c.LarkWebhookURL == "" && c.WebhookURL == "" {
		return fmt.Errorf("at least one of lark_webhook_url or webhook_url must be provided")
	}

	// Validate thresholds if they are provided
	if c.DiskThreshold != nil && (*c.DiskThreshold < 1 || *c.DiskThreshold > 100) {
		return fmt.Errorf("disk_threshold must be between 1 and 100, got %.2f", *c.DiskThreshold)
	}
	if c.CPUThreshold != nil && (*c.CPUThreshold < 1 || *c.CPUThreshold > 100) {
		return fmt.Errorf("cpu_threshold must be between 1 and 100, got %.2f", *c.CPUThreshold)
	}
	if c.MemoryThreshold != nil && (*c.MemoryThreshold < 1 || *c.MemoryThreshold > 100) {
		return fmt.Errorf("memory_threshold must be between 1 and 100, got %.2f", *c.MemoryThreshold)
	}

	return nil
}



