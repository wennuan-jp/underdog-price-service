package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Config holds all configuration values
type Config struct {
	ExchangeRateAPIKey string
	// Add other configuration fields as needed
	// DatabaseURL     string
	// APISecret       string
	// JWTSecret       string
}

// LoadConfig reads configuration from a file without extension
func LoadConfig(filename string) (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file '%s' does not exist", filename)
	}

	// Open the file
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	config := &Config{}
	scanner := bufio.NewScanner(file)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key=value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("line %d: invalid format (expected key=value): %s", lineNum, line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		value = strings.Trim(value, `"'`)

		// Set configuration values
		switch key {
		case "exchangeRateApiKey":
			config.ExchangeRateAPIKey = value
		// Add more cases as needed
		// case "databaseUrl":
		//     config.DatabaseURL = value
		// case "apiSecret":
		//     config.APISecret = value
		default:
			return nil, fmt.Errorf("line %d: unknown key '%s'", lineNum, key)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
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
		return fmt.Errorf("exchangeRateApiKey is required but not set")
	}
	// Add more validations as needed
	// if c.DatabaseURL == "" {
	//     return fmt.Errorf("databaseUrl is required but not set")
	// }
	return nil
}

// GetExchangeRateAPIKey returns the API key (with optional masking for logging)
func (c *Config) GetExchangeRateAPIKey() string {
	return c.ExchangeRateAPIKey
}

// MaskedAPIKey returns a masked version of the API key for logging
func (c *Config) MaskedAPIKey() string {
	if len(c.ExchangeRateAPIKey) <= 8 {
		return "****"
	}
	return c.ExchangeRateAPIKey[:4] + "****" + c.ExchangeRateAPIKey[len(c.ExchangeRateAPIKey)-4:]
}

// String returns a string representation of the config (with sensitive data masked)
func (c *Config) String() string {
	return fmt.Sprintf("Config{ExchangeRateAPIKey: %s}", c.MaskedAPIKey())
}
