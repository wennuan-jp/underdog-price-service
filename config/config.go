package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration values
type Config struct {
	ExchangeRateAPIKey   string `yaml:"exchange_rate_api_key"`
	MetadataOutdatedDays int    `yaml:"metadata_outdated_days"`
}

// LoadConfig reads configuration from a YAML file
func LoadConfig(filename string) (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file '%s' does not exist", filename)
	}

	// Open/Read the file
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := &Config{
		MetadataOutdatedDays: 50, // Default value
	}
	
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	// Validate required fields
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate checks if all required configuration fields are set
func (c *Config) Validate() error {
	if c.ExchangeRateAPIKey == "" {
		return fmt.Errorf("exchange_rate_api_key is required but not set")
	}
	if c.MetadataOutdatedDays <= 0 {
		return fmt.Errorf("metadata_outdated_days must be a positive integer")
	}
	return nil
}

// String returns a string representation of the config (with sensitive data masked)
func (c *Config) String() string {
	maskedKey := "****"
	if len(c.ExchangeRateAPIKey) > 8 {
		maskedKey = c.ExchangeRateAPIKey[:4] + "****" + c.ExchangeRateAPIKey[len(c.ExchangeRateAPIKey)-4:]
	}
	return fmt.Sprintf("Config{ExchangeRateAPIKey: %s, MetadataOutdatedDays: %d}", maskedKey, c.MetadataOutdatedDays)
}

