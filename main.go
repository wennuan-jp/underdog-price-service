package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"humanpatch.com/underdog/asset-service/config"
	"humanpatch.com/underdog/asset-service/infra"
	"humanpatch.com/underdog/asset-service/model"
	"humanpatch.com/underdog/asset-service/service"

	"github.com/gin-gonic/gin"
)

func main() {
	// Root context
	ctx := context.Background()

	// 1. Load configuration
	cfg, err := config.LoadConfig("_secret")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Initialize Firebase
	firebaseClient, err := infra.InitFirebase(ctx, "_firebase_credentials.json")
	if err != nil {
		log.Fatalf("Failed to initialize Firebase: %v", err)
	}
	defer firebaseClient.Close()

	// 3. Initialize Services
	adminService := &service.AdminService{Config: cfg}
	fetcher := service.NewPriceFetcherService(*cfg)
	firebaseService := service.NewFirebaseService(firebaseClient)

	// 4. Start Firestore real-time listener (Watch mechanism)
	// Runs in a separate goroutine to avoid blocking the API server
	go firebaseService.WatchPricesChanges(ctx)

	// 5. Setup Gin HTTP Server
	r := gin.Default()

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "up",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// 1. Refresh Supported Currencies (Admin/Metadata)
	r.POST("/v1/admin/refresh-supported", func(c *gin.Context) {
		reqCtx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		if err := adminService.RefreshSupportedPricePairFXWithCTX(reqCtx); err != nil {
			log.Printf("❌ Currency list refresh failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh supported currencies"})
			return
		}

		currencies := adminService.GetSupportedCurrencies()
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": fmt.Sprintf("Successfully refreshed %d currency names", len(currencies)),
			"count":   len(currencies),
		})
	})

	// 2. Synchronize FX Rates with Firebase
	r.POST("/v1/price/sync-fx", func(c *gin.Context) {
		reqCtx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()

		// Fetch prices
		if err := fetcher.FetchFXPricePairWithContext(reqCtx); err != nil {
			log.Printf("❌ Failed to fetch live prices: %v", err)
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Failed to fetch prices from source", "info": err.Error()})
			return
		}

		rates := fetcher.GetAllRates()
		var ratePairs []model.PricePair

		for _, rate := range rates {
			priceInUSD := 0.0
			if rate.Rate != 0 {
				priceInUSD = 1.0 / rate.Rate
			}

			name, _ := adminService.GetCurrencyName(rate.Target)
			if name == "" {
				name = rate.Target
			}

			ratePairs = append(ratePairs, model.PricePair{
				ID:          rate.Target,
				Name:        name,
				AssetType:   model.AssetTypeFX,
				Code:        rate.Target,
				PriceInUSD:  priceInUSD,
				LastUpdated: rate.Timestamp,
			})
		}

		// Batch push to Firebase
		if err := firebaseService.UpdatePrices(reqCtx, ratePairs); err != nil {
			log.Printf("❌ Failed to push prices to Firebase: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync with Firebase"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": fmt.Sprintf("Synchronized %d FX rates with Firebase", len(ratePairs)),
			"count":   len(ratePairs),
		})
	})

	// 5. Run the server
	port := "9333"
	log.Printf("🚀 Asset Price Service starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
