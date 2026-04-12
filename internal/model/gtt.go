package models

import "gorm.io/gorm"

type GTTOrder struct {
	gorm.Model
	TriggerID         int    `gorm:"uniqueIndex"` // Kite's trigger ID
	TradingSymbol     string `gorm:"index"`
	Exchange          string
	TransactionType   string // BUY or SELL
	TriggerType       string // single or oco
	Status            string `gorm:"index"` // active, triggered, cancelled, deleted
	LastPrice         float64
	TriggerPrice      float64
	LimitPrice        float64
	Quantity          float64
	UpperTriggerPrice float64 // OCO upper leg (zero if single)
	UpperLimitPrice   float64
	UpperQuantity     float64
}
