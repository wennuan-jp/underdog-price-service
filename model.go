package main

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
	ID          string
	Name        string
	AssetType   AssetType
	Code        string
	PriceInUSD  float64
	LastUpdated time.Time
}

// type PriceSource interface {
// 	ID          string
// 	URLEndpoint string

// }

// type PriceSource struct {
// 	ID          string
// 	URLEndpoint string // ✅ Exported (Capital 'U')
// }
