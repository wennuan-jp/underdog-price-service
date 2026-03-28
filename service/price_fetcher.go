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

// FXLiveResponse represents the live exchange rates API response
type FXLiveResponse struct {
	Success   bool               `json:"success"`
	Terms     string             `json:"terms"`
	Privacy   string             `json:"privacy"`
	Timestamp int64              `json:"timestamp"`
	Source    string             `json:"source"`
	Quotes    map[string]float64 `json:"quotes"`
}

// FXRate represents a single exchange rate
type FXRate struct {
	Pair      string    `json:"pair"`      // e.g., "USDEUR"
	Base      string    `json:"base"`      // e.g., "USD"
	Target    string    `json:"target"`    // e.g., "EUR"
	Rate      float64   `json:"rate"`      // Exchange rate
	Timestamp time.Time `json:"timestamp"` // When this rate was fetched
}

// PriceFetcherService handles fetching exchange rates
type PriceFetcherService struct {
	config config.Config

	mu            sync.RWMutex
	latestRates   map[string]*FXRate // key: currency pair (e.g., "USDEUR")
	lastFetchTime time.Time
	httpClient    *http.Client
}

// NewPriceFetcherService creates a new price fetcher service
func NewPriceFetcherService(config config.Config) *PriceFetcherService {
	return &PriceFetcherService{
		config:      config,
		latestRates: make(map[string]*FXRate),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// FetchFXPricePair fetches exchange rates for specified currency pairs
func (s *PriceFetcherService) FetchFXPricePair() error {
	return s.FetchFXPricePairWithContext(context.Background())
}

// FetchFXPricePairWithContext fetches exchange rates with context support
func (s *PriceFetcherService) FetchFXPricePairWithContext(ctx context.Context) error {
	apiKey := s.config.ExchangeRateAPIKey

	// Build URL for live endpoint (not list)
	// Using symbols parameter to filter only needed currencies
	symbols := "USD,EUR,JPY,CNY,GBP" // Add more as needed
	fetchURL := FXHost + "/live?access_key=" + apiKey + "&source=USD&symbols=" + symbols

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

	// Execute request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch exchange rates: %w", err)
	}
	defer resp.Body.Close()

	// Read response body with size limit
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse JSON response
	var fxResp FXLiveResponse
	if err := json.Unmarshal(body, &fxResp); err != nil {
		return fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Check success flag
	if !fxResp.Success {
		return fmt.Errorf("API returned success=false: %s", string(body))
	}

	// Convert quotes to FXRate objects
	rates := make(map[string]*FXRate)
	timestamp := time.Unix(fxResp.Timestamp, 0)

	for pair, rate := range fxResp.Quotes {
		// Pair format is like "USDEUR" (base + target)
		if len(pair) >= 6 {
			base := pair[:3]
			target := pair[3:]

			rates[pair] = &FXRate{
				Pair:      pair,
				Base:      base,
				Target:    target,
				Rate:      rate,
				Timestamp: timestamp,
			}
		}
	}

	// Update with lock
	s.mu.Lock()
	s.latestRates = rates
	s.lastFetchTime = time.Now()
	s.mu.Unlock()

	return nil
}

// GetRate returns the exchange rate for a specific currency pair
func (s *PriceFetcherService) GetRate(base, target string) (*FXRate, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pair := base + target
	rate, exists := s.latestRates[pair]
	return rate, exists
}

// GetAllRates returns all fetched rates
func (s *PriceFetcherService) GetAllRates() map[string]*FXRate {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to avoid race conditions
	rates := make(map[string]*FXRate, len(s.latestRates))
	for k, v := range s.latestRates {
		rates[k] = &FXRate{
			Pair:      v.Pair,
			Base:      v.Base,
			Target:    v.Target,
			Rate:      v.Rate,
			Timestamp: v.Timestamp,
		}
	}
	return rates
}

// GetLastFetchTime returns when rates were last fetched
func (s *PriceFetcherService) GetLastFetchTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastFetchTime
}

// Convert performs a currency conversion
func (s *PriceFetcherService) Convert(amount float64, from, to string) (float64, error) {
	if from == to {
		return amount, nil
	}

	// Try direct conversion
	rate, exists := s.GetRate(from, to)
	if exists {
		return amount * rate.Rate, nil
	}

	// Try inverse conversion
	rate, exists = s.GetRate(to, from)
	if exists {
		return amount / rate.Rate, nil
	}

	// Try via USD (if available)
	usdToFrom, existsFrom := s.GetRate("USD", from)
	usdToTo, existsTo := s.GetRate("USD", to)

	if existsFrom && existsTo {
		// Convert to USD first, then to target
		usdAmount := amount / usdToFrom.Rate
		return usdAmount * usdToTo.Rate, nil
	}

	return 0, fmt.Errorf("cannot convert from %s to %s: no direct or indirect rate available", from, to)
}
