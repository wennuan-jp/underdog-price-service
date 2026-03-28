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
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("❌ Critical Failure: Failed to load config: %v", err)
	}

	// 2. Initialize Database Connections
	firebaseClient, err := infra.InitFirebase(ctx, "_firebase_credentials.json")
	if err != nil {
		log.Fatalf("❌ Critical Failure: Failed to initialize Firebase: %v", err)
	}
	defer firebaseClient.Close()

	sqliteDB, err := infra.InitSQLite("metadata.db")
	if err != nil {
		log.Fatalf("❌ Critical Failure: Failed to initialize SQLite: %v", err)
	}
	defer sqliteDB.Close()

	// 3. Initialize Services
	adminService := &service.AdminService{Config: cfg, DB: sqliteDB}
	fetcher := service.NewPriceFetcherService(*cfg)
	firebaseService := service.NewFirebaseService(firebaseClient)

	// 🛠️ BLOCKING STARTUP LOGIC: Ensure metadata is ready before accepting requests
	log.Println("🛠️  Preparing service metadata...")
	outdated, err := adminService.IsCacheOutdated(cfg.MetadataOutdatedDays)

	if err != nil || outdated {
		if err != nil {
			log.Printf("⚠️  SQLite cache check failed: %v", err)
		}
		log.Println("🔄 Local metadata outdated or missing. Syncing with external API...")
		if err := adminService.SyncWithAPI(ctx); err != nil {
			log.Fatalf("❌ Critical Failure: Failed to initialize currency metadata from API: %v", err)
		}
	} else {
		log.Println("📂 Loading currency metadata from local SQLite cache...")
		if count, err := adminService.LoadFromSQLite(); err != nil || count == 0 {
			log.Printf("⚠️  Failed to load from SQLite (%v). Falling back to API...", err)
			if err := adminService.SyncWithAPI(ctx); err != nil {
				log.Fatalf("❌ Critical Failure: Failed to initialize metadata: %v", err)
			}
		} else {
			log.Printf("✅ Successfully loaded %d currency names from SQLite", count)
		}
	}

	// 4. Start Firestore real-time listener (Watch mechanism)
	// We watch the live prices in Firestore for verification
	go firebaseService.WatchPricesChanges(ctx, "fx_prices")

	// 5. Setup Gin HTTP Server
	r := gin.Default()

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "up",
			"time":   time.Now().Format(time.RFC3339),
			"cache_age": adminService.GetLastRefreshTime().Format(time.RFC3339),
		})
	})

	// 1. Refresh Supported Currencies (Metadata Maintenance)
	r.POST("/v1/admin/refresh-supported", func(c *gin.Context) {
		reqCtx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()

		if err := adminService.SyncWithAPI(reqCtx); err != nil {
			log.Printf("❌ Currency list refresh failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh metadata", "info": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": fmt.Sprintf("Successfully refreshed %d names in SQLite cache", len(adminService.GetSupportedCurrencies())),
			"count":   len(adminService.GetSupportedCurrencies()),
		})
	})

	// 2. Synchronize FX Rates with Firebase
	r.POST("/v1/price/sync-fx", func(c *gin.Context) {
		reqCtx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()

		// Fetch prices from source
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

			// Retrieve descriptive name from SQLite cache
			name, _ := adminService.GetCurrencyName(rate.Target)
			if name == "" {
				name = rate.Target
			}

			ratePairs = append(ratePairs, model.PricePair{
				Name:        name,
				AssetType:   model.AssetTypeFX,
				Code:        rate.Target,
				PriceInUSD:  priceInUSD,
				LastUpdated: rate.Timestamp,
			})
		}

		// Batch push to Firebase (using names fetched from SQLite)
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

	// 6. Run the server
	port := "9333"
	log.Printf("🚀 Asset Price Service starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}

}
