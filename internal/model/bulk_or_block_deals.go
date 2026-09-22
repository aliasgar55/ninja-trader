package models

import (
	"time"

	"gorm.io/gorm"
)

type BulkBlockDeal struct {
	gorm.Model
	TradingSymbol string `gorm:"index"`
	ScripName     string
	ClientName    string
	DealType      DealType
	BuySell       TradeType `gorm:"index"` // "BUY" or "SELL"
	Quantity      int64
	Price         float64
	Date          time.Time `gorm:"index"`
	Remarks       string
}
