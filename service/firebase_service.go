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

// UpdatePrice pushes a single price pair to the correct Firestore collection based on asset type.
func (s *FirebaseService) UpdatePrice(ctx context.Context, price model.PricePair) error {
	if s.Client == nil || s.Client.Firestore == nil {
		return fmt.Errorf("firebase client or firestore is not initialized")
	}

	collectionName := s.getCollectionName(price.AssetType)
	docRef := s.Client.Firestore.Collection(collectionName).Doc(price.Code)

	// Set the data.
	_, err := docRef.Set(ctx, price)
	if err != nil {
		return fmt.Errorf("failed to push price to %s for %s: %w", collectionName, price.Code, err)
	}

	log.Printf("✅ Pushed to Firebase [%s]: %s = $%.4f", collectionName, price.Code, price.PriceInUSD)
	return nil
}

// UpdatePrices pushes multiple prices in a single batch to their respective collections.
func (s *FirebaseService) UpdatePrices(ctx context.Context, prices []model.PricePair) error {
	batch := s.Client.Firestore.Batch()

	for _, price := range prices {
		collectionName := s.getCollectionName(price.AssetType)
		docRef := s.Client.Firestore.Collection(collectionName).Doc(price.Code)
		batch.Set(docRef, price)
	}

	_, err := batch.Commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to commit batch update to firestore: %w", err)
	}

	log.Printf("🚀 Successfully pushed %d prices across specific collections in a single batch", len(prices))
	return nil
}

// getCollectionName returns the Firestore collection name based on the asset type
func (s *FirebaseService) getCollectionName(assetType model.AssetType) string {
	switch assetType {
	case model.AssetTypeFX:
		return "fx_prices"
	case model.AssetTypeStock:
		return "stock_prices"
	case model.AssetTypeCrypto:
		return "crypto_prices"
	default:
		return "asset_prices"
	}
}

// WatchPricesChanges starts a real-time listener on a specific collection.
func (s *FirebaseService) WatchPricesChanges(ctx context.Context, collectionName string) {
	if s.Client == nil || s.Client.Firestore == nil {
		log.Println("❌ Watcher failed: Firestore client is not initialized")
		return
	}

	iter := s.Client.Firestore.Collection(collectionName).Snapshots(ctx)
	defer iter.Stop()

	log.Printf("👀 Firestore Price Watcher started for [%s]... waiting for changes.", collectionName)

	for {
		snap, err := iter.Next()
		if err != nil {
			if ctx.Err() != nil {
				log.Printf("🛑 [%s] Watcher stopped.", collectionName)
				return
			}
			log.Printf("❌ [%s] Watcher error: %v", collectionName, err)
			return
		}

		for _, change := range snap.Changes {
			var p model.PricePair
			if err := change.Doc.DataTo(&p); err != nil {
				log.Printf("❌ Failed to parse data in [%s] for document %s: %v", collectionName, change.Doc.Ref.ID, err)
				continue
			}

			switch change.Kind {
			case firestore.DocumentAdded:
				log.Printf("📥 [%s] NEW -> %s: $%.4f", collectionName, p.Code, p.PriceInUSD)
			case firestore.DocumentModified:
				log.Printf("🔄 [%s] UPD -> %s: $%.4f", collectionName, p.Code, p.PriceInUSD)
			case firestore.DocumentRemoved:
				log.Printf("🗑️ [%s] DEL -> %s", collectionName, p.Code)
			}
		}
	}
}





