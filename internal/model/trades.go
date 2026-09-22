package models

import (
	"time"

	"gorm.io/gorm"
)


type PaperTrade struct {
	gorm.Model
	InstrumentID  uint `gorm:"not null;index"`
	Instrument    Instrument
	AveragePrice  float64
	Quantity      int
	TradingSymbol string
}

type PaperTradeLog struct {
	gorm.Model
	InstrumentID  uint `gorm:"not null;index"`
	Instrument    Instrument
	Quantity      uint
	TradeType     TradeType
	Date          time.Time
	AvgPrice      float64
	Amount        float64
	TradingSymbol string
	ExecutedAt    time.Time
}
