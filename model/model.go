package model

import (
	"time"
)

type AssetType string

const (
	AssetTypeCrypto AssetType = "Crypto"
	AssetTypeStock  AssetType = "Stock"
	AssetTypeFX     AssetType = "FX"
)

type PricePair struct {
	ID          string    `firestore:"id"`
	Name        string    `firestore:"name"`
	AssetType   AssetType `firestore:"asset_type"`
	Code        string    `firestore:"code"`
	PriceInUSD  float64   `firestore:"price_usd"`
	LastUpdated time.Time `firestore:"last_updated"`
}


// type PriceSource interface {
// 	ID          string
// 	URLEndpoint string

// }

// type PriceSource struct {
// 	ID          string
// 	URLEndpoint string // ✅ Exported (Capital 'U')
// }
