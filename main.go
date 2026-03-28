package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"humanpatch.com/underdog/asset-service/config"
	"humanpatch.com/underdog/asset-service/service"
)

func main() {
	// Load configuration from file without extension
	config, err := config.LoadConfig("_secret")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Use the configuration
	fmt.Printf("Config loaded successfully: %s\n", config)
	fmt.Printf("Masked ExchangeRateAPIKey Key: %s\n", config.MaskedAPIKey())

	// Create admin service
	adminService := &service.AdminService{
		Config: config,
	}

	// Refresh currency list
	// With context and timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := adminService.RefreshSupportedPricePairFXWithCTX(ctx); err != nil {
		log.Printf("Refresh failed: %v", err)
	}

	// Get all supported currencies
	currencies := adminService.GetSupportedCurrencies()
	fmt.Printf("Loaded %d currencies\n", len(currencies))

	// Print first 10 currencies
	for i, currency := range currencies {
		if i >= 10 {
			break
		}
		fmt.Printf("%s: %s\n", currency.Code, currency.Name)
	}

	// Check if a currency is supported
	if adminService.IsCurrencySupported("USD") {
		fmt.Println("USD is supported")
	}

	// Get currency name
	if name, exists := adminService.GetCurrencyName("EUR"); exists {
		fmt.Printf("EUR: %s\n", name)
	}

}
