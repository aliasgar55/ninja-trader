package models

import (
	"time"

	"gorm.io/gorm"
)

type Instrument struct {
	gorm.Model
	ID                 uint `gorm:"primaryKey"`
	InstrumentToken    int  `gorm:"uniqueIndex"`
	InstrumentFullName string
	ExchangeToken      int
	TradingSymbol      string `gorm:"index"`
	Exchange           string
	Active             bool
	MarketCap          float64 `gorm:"index"`
	IsNav              bool
	PriceBand          string
	NseApiSuccess      bool
	NseApiError        string
	Watchlist          bool `gorm:"index"`
	BasicIndustry      string
	Index              string
	IsDailySyncFailed  bool
	DailySyncError     string
	LastDailySyncDate  time.Time
}

type EventType int
const (
    Dividen EventType = iota
    Split
)

type Event struct {
	gorm.Model
	EventType EventType
	Dividen   float32
	Numerator uint8
	Denominator uint8
}

type Historicaldata struct {
	gorm.Model
	O                               float64
	H                               float64
	L                               float64
	C                               float64
	AdjustedClosePrice              float64
	AlphaVantageWeeklyAdjustedClose float64
	LastTradedPrice                 float64
	Vwap                            float64
	Volume                          int64
	NoOfTrades                      int64
	Symbol                          string    `gorm:"index:,priority:1,unique,composite:idx_historicaldata_symbol_date"`
	Date                            time.Time `gorm:"index:,priority:2,unique,composite:idx_historicaldata_symbol_date"`
	YearHigh                        float64
	YearLow                         float64
	VolumePerTrade                  int64
	DeliveryPercentage              float32
}

type Shorts struct {
	gorm.Model
	TradingSymbol string `gorm:"index"`
	SecurityName  string
	Quantity      int64     `grom:"index"`
	Date          time.Time `gorm:"index"`
}

type AdjustedCloseUpdate struct {
	Symbol             string
	Date               time.Time
	AdjustedClosePrice float64
}
