package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"humanpatch.com/underdog/asset-service/config"
)

// CurrencyResponse represents the API response structure
type CurrencyResponse struct {
	Success    bool              `json:"success"`
	Terms      string            `json:"terms"`
	Privacy    string            `json:"privacy"`
	Currencies map[string]string `json:"currencies"`
}

// SupportedCurrency represents a currency with its code and name
type SupportedCurrency struct {
	Code string
	Name string
}

// Add this to your AdminService struct
type AdminService struct {
	Config              *config.Config
	supportedCurrencies []SupportedCurrency
	currenciesMap       map[string]string
	mu                  sync.RWMutex
	lastRefreshTime     time.Time
}

// refreshSupportedPricePairFXWithContext fetches currencies with context support
func (s *AdminService) RefreshSupportedPricePairFXWithCTX(ctx context.Context) error {
	// Construct the URL with API key
	fetchURL := FXHost + "/list?access_key=" + s.Config.ExchangeRateAPIKey

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", fetchURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", "Apifox/1.0.0 (https://apifox.com)")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Host", "api.exchangerate.host")
	req.Header.Set("Connection", "keep-alive")

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch currency list: %w", err)
	}
	defer resp.Body.Close()

	// Read response body with size limit (1MB)
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var currencyResp CurrencyResponse
	if err := json.Unmarshal(body, &currencyResp); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Check success flag
	if !currencyResp.Success {
		return fmt.Errorf("API returned failure: %s", string(body))
	}

	// Convert to slice and map
	currencies := make([]SupportedCurrency, 0, len(currencyResp.Currencies))
	currenciesMap := make(map[string]string, len(currencyResp.Currencies))

	for code, name := range currencyResp.Currencies {
		currencies = append(currencies, SupportedCurrency{
			Code: code,
			Name: name,
		})
		currenciesMap[code] = name
	}

	// Update with lock
	s.mu.Lock()
	s.supportedCurrencies = currencies
	s.currenciesMap = currenciesMap
	s.lastRefreshTime = time.Now()
	s.mu.Unlock()

	return nil
}

// GetSupportedCurrencies returns a copy of supported currencies
func (s *AdminService) GetSupportedCurrencies() []SupportedCurrency {
	s.mu.RLock()
	defer s.mu.RUnlock()

	currencies := make([]SupportedCurrency, len(s.supportedCurrencies))
	copy(currencies, s.supportedCurrencies)
	return currencies
}

// GetCurrencyName returns the name for a currency code
func (s *AdminService) GetCurrencyName(code string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	name, exists := s.currenciesMap[code]
	return name, exists
}

// IsCurrencySupported checks if a currency code is supported
func (s *AdminService) IsCurrencySupported(code string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.currenciesMap[code]
	return exists
}

// GetLastRefreshTime returns when the currency list was last refreshed
func (s *AdminService) GetLastRefreshTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.lastRefreshTime
}
