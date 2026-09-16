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
	WatchListDate      time.Time
	NeedsAdjsutment    bool
	Tag                string
	IsFnoSec           bool
}

type TagHistory struct {
	gorm.Model
	InstrumentID  uint `gorm:"not null;index"`
	TradingSymbol string
	PreviousTag   string
	NextTag       string
	UpdatedOn     time.Time
}

type EventType int

const (
	Dividend EventType = iota
	Split
)

type Event struct {
	gorm.Model
	InstrumentID  uint `gorm:"not null;index"`
	Instrument    Instrument
	TradingSymbol string
	EventDate     time.Time
	EventType     EventType
	Dividend      float32
	Numerator     uint8
	Denominator   uint8
	Processed     bool
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
	Volume                          uint64
	NoOfTrades                      int64
	Symbol                          string    `gorm:"index:,priority:1,unique,composite:idx_historicaldata_symbol_date"`
	Date                            time.Time `gorm:"index:,priority:2,unique,composite:idx_historicaldata_symbol_date"`
	YearHigh                        float64
	YearLow                         float64
	VolumePerTrade                  int64
	DeliveryPercentage              float32
	AdjustementFactor               float64
	VptMa20                         float64
	VolumeMa20                      float64
	VptScore                        float64
	Divergence                      float64
	DivergenceMax3y                 float64
	DeliveryValue                   float64
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

type SignalUpdate struct {
	ID              uint
	VptMa20         float64
	VolumeMa20      float64
	VptScore        float64
	Divergence      float64
	DivergenceMax3y float64
}

type Note struct {
	gorm.Model
	InstrumentID  uint   `gorm:"not null;index"`
	TradingSymbol string `gorm:"index"`
	Date          time.Time
	Text          string
}

type AlertLog struct {
	gorm.Model
	AlertType     string `gorm:"index"` // "PRICE" or "VOLUME"
	TradingSymbol string `gorm:"index"`
	Price         float64
	Message       string
	AlertTime     time.Time `gorm:"index"`
}
