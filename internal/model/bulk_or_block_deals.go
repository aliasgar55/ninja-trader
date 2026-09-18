package models

import (
	"time"

	"gorm.io/gorm"
)

type DealType string

const (
	DealTypeBulk  DealType = "BULK"
	DealTypeBlock DealType = "BLOCK"
)

type BulkBlockDeal struct {
	gorm.Model
	TradingSymbol string `gorm:"index"`
	ScripName     string
	ClientName    string
	DealType      string
	BuySell       string `gorm:"index"` // "BUY" or "SELL"
	Quantity      int64
	Price         float64
	Date          time.Time `gorm:"index"`
	Remarks       string
}
