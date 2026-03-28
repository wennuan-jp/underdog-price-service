package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
	Code      string
	Name      string
	UpdatedAt time.Time
}

// AdminService handles administrative tasks and metadata
type AdminService struct {
	Config              *config.Config
	DB                  *sql.DB
	supportedCurrencies []SupportedCurrency
	currenciesMap       map[string]string
	mu                  sync.RWMutex
	lastRefreshTime     time.Time
}

// SyncWithAPI fetches latest currencies and saves to memory + SQLite
func (s *AdminService) SyncWithAPI(ctx context.Context) error {
	if err := s.RefreshSupportedPricePairFXWithCTX(ctx); err != nil {
		return err
	}
	// Persist to SQLite
	return s.SaveToSQLite()
}

// RefreshSupportedPricePairFXWithCTX fetches currencies from API
func (s *AdminService) RefreshSupportedPricePairFXWithCTX(ctx context.Context) error {
	log.Println("🌐 Fetching latest currency names from external API...")
	
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
	now := time.Now()
	currencies := make([]SupportedCurrency, 0, len(currencyResp.Currencies))
	currenciesMap := make(map[string]string, len(currencyResp.Currencies))

	for code, name := range currencyResp.Currencies {
		currencies = append(currencies, SupportedCurrency{
			Code:      code,
			Name:      name,
			UpdatedAt: now,
		})
		currenciesMap[code] = name
	}

	// Update with lock
	s.mu.Lock()
	s.supportedCurrencies = currencies
	s.currenciesMap = currenciesMap
	s.lastRefreshTime = now
	s.mu.Unlock()

	return nil
}

// SaveToSQLite persists the in-memory currency list to the SQLite database
func (s *AdminService) SaveToSQLite() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.DB == nil {
		return fmt.Errorf("sqlite database not initialized in AdminService")
	}

	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO supported_fx_currencies (code, name, updated_at) 
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(code) DO UPDATE SET 
			name = excluded.name,
			updated_at = CURRENT_TIMESTAMP
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, cur := range s.supportedCurrencies {
		if _, err := stmt.Exec(cur.Code, cur.Name); err != nil {
			return err
		}
	}

	log.Printf("📥 Persisted %d currencies to SQLite", len(s.supportedCurrencies))
	return tx.Commit()
}

// LoadFromSQLite loads the currency list from SQLite into memory
func (s *AdminService) LoadFromSQLite() (int, error) {
	if s.DB == nil {
		return 0, fmt.Errorf("sqlite database not initialized in AdminService")
	}

	rows, err := s.DB.Query("SELECT code, name, updated_at FROM supported_fx_currencies")
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var currencies []SupportedCurrency
	currenciesMap := make(map[string]string)
	var latestUpdate time.Time

	for rows.Next() {
		var cur SupportedCurrency
		if err := rows.Scan(&cur.Code, &cur.Name, &cur.UpdatedAt); err != nil {
			return 0, err
		}
		currencies = append(currencies, cur)
		currenciesMap[cur.Code] = cur.Name
		if cur.UpdatedAt.After(latestUpdate) {
			latestUpdate = cur.UpdatedAt
		}
	}

	if len(currencies) == 0 {
		return 0, nil
	}

	s.mu.Lock()
	s.supportedCurrencies = currencies
	s.currenciesMap = currenciesMap
	s.lastRefreshTime = latestUpdate
	s.mu.Unlock()

	return len(currencies), nil
}

// IsCacheOutdated checks if the data in SQLite is older than the given days
func (s *AdminService) IsCacheOutdated(days int) (bool, error) {
	var count int
	err := s.DB.QueryRow("SELECT COUNT(*) FROM supported_fx_currencies").Scan(&count)
	if err != nil || count == 0 {
		return true, nil // Empty or error, consider outdated
	}

	var latestUpdateStr string
	err = s.DB.QueryRow("SELECT MAX(updated_at) FROM supported_fx_currencies").Scan(&latestUpdateStr)
	if err != nil {
		return true, err
	}

	// Parsing sqlite datetime (format depends on how it's stored, usually RFC3339 or similar)
	formats := []string{"2006-01-02 15:04:05", time.RFC3339}
	var latestUpdate time.Time
	var parseErr error
	for _, f := range formats {
		latestUpdate, parseErr = time.Parse(f, latestUpdateStr)
		if parseErr == nil {
			break
		}
	}

	if parseErr != nil {
		// Try to parse as unix timestamp or just assume outdated
		return true, nil
	}

	if time.Since(latestUpdate).Hours() > float64(days*24) {
		return true, nil
	}

	return false, nil
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

// SetSupportedCurrencies manually updates the internal currency list (useful for loading from cache or DB)
func (s *AdminService) SetSupportedCurrencies(currencies []SupportedCurrency) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.supportedCurrencies = currencies
	s.currenciesMap = make(map[string]string, len(currencies))
	for _, cur := range currencies {
		s.currenciesMap[cur.Code] = cur.Name
	}
	s.lastRefreshTime = time.Now()
}

