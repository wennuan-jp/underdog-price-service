package service

import (
	"context"
	"fmt"
	"log"

	"humanpatch.com/underdog/asset-service/infra"
	"humanpatch.com/underdog/asset-service/model"
	"cloud.google.com/go/firestore"
)

// FirebaseService handles data operations for Firebase Firestore
type FirebaseService struct {
	Client *infra.FirebaseClient
}

// NewFirebaseService creates a new instance of FirebaseService
func NewFirebaseService(client *infra.FirebaseClient) *FirebaseService {
	return &FirebaseService{
		Client: client,
	}
}

// UpdatePrice pushes a single price pair to the Firestore 'prices' collection.
// It uses the asset's Code as the document ID (e.g., 'BTC', 'USD').
func (s *FirebaseService) UpdatePrice(ctx context.Context, price model.PricePair) error {
	if s.Client == nil || s.Client.Firestore == nil {
		return fmt.Errorf("firebase client or firestore is not initialized")
	}

	// Use asset code as the unique document identifier
	docRef := s.Client.Firestore.Collection("prices").Doc(price.Code)

	// Set the data.
	_, err := docRef.Set(ctx, price)
	if err != nil {
		return fmt.Errorf("failed to push price to firestore for %s: %w", price.Code, err)
	}

	log.Printf("✅ Pushed to Firebase: %s = $%.4f", price.Code, price.PriceInUSD)
	return nil
}

// UpdatePrices pushes multiple prices in a single batch (optional, but good for efficiency)
func (s *FirebaseService) UpdatePrices(ctx context.Context, prices []model.PricePair) error {
	batch := s.Client.Firestore.Batch()

	for _, price := range prices {
		docRef := s.Client.Firestore.Collection("prices").Doc(price.Code)
		batch.Set(docRef, price)
	}

	_, err := batch.Commit(ctx)

	if err != nil {
		return fmt.Errorf("failed to commit batch update to firestore: %w", err)
	}

	log.Printf("🚀 Successfully pushed %d prices to Firebase in a single batch", len(prices))
	return nil
}

// WatchPricesChanges starts a real-time listener on the 'prices' collection and logs changes to the console.
// This is useful for verifying that data is being correctly pushed without a separate frontend.
func (s *FirebaseService) WatchPricesChanges(ctx context.Context) {
	if s.Client == nil || s.Client.Firestore == nil {
		log.Println("❌ Watcher failed: Firestore client is not initialized")
		return
	}

	// Create an iterator that yields snapshots as data changes
	iter := s.Client.Firestore.Collection("prices").Snapshots(ctx)
	defer iter.Stop()

	log.Println("👀 Firestore Price Watcher started... waiting for changes.")

	// Continuous loop to process snapshots
	for {
		snap, err := iter.Next()
		if err != nil {
			// Check if context was cancelled
			if ctx.Err() != nil {
				log.Println("🛑 Firestore Price Watcher stopped.")
				return
			}
			log.Printf("❌ Watcher error: %v", err)
			return
		}

		// Process each document change in the snapshot
		for _, change := range snap.Changes {
			var p model.PricePair
			if err := change.Doc.DataTo(&p); err != nil {
				log.Printf("❌ Failed to parse Firestore data for document %s: %v", change.Doc.Ref.ID, err)
				continue
			}

			switch change.Kind {
			case firestore.DocumentAdded:
				// On first run, this logs all existing prices. Afterwards, only new ones.
				log.Printf("📥 [Firestore] NEW -> %s: $%.4f (at %s)", p.Code, p.PriceInUSD, p.LastUpdated.Format("15:04:05"))
			case firestore.DocumentModified:
				log.Printf("🔄 [Firestore] UPD -> %s: $%.4f (at %s)", p.Code, p.PriceInUSD, p.LastUpdated.Format("15:04:05"))
			case firestore.DocumentRemoved:
				log.Printf("🗑️ [Firestore] DEL -> %s", p.Code)
			}
		}
	}
}

