package models

import "gorm.io/gorm"

type KiteAlert struct {
	gorm.Model
	AlertID       string `gorm:"uniqueIndex"`
	TradingSymbol string `gorm:"index"`
	Exchange      string
	Operator      string // >= or <=
	TriggerValue  float64
	Status        string `gorm:"index"` // active, triggered, deleted
}
