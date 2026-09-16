package models

import (
	"time"

	"gorm.io/gorm"
)

type InsiderTradeTransaction struct {
	ID uint `gorm:"primaryKey"`

	// Filing from which this transaction was extracted.
	FilingID uint `gorm:"not null;uniqueIndex"`

	// Company / security identification.
	Symbol string `gorm:"not null"`

	// Person / entity involved in the transaction.
	EntityID uint `gorm:"not null"`
	AcqMode  string

	// Security information.
	SecurityType string `gorm:"type:varchar(30);not null"`

	// Transaction classification.
	TransactionType string `gorm:"type:varchar(100);not null;index"`
	TransactionMode string `gorm:"type:varchar(100);not null;index"`

	// Actual transaction date.
	TransactionDate time.Time `gorm:"not null"` // acqToDate
	IntimationDate  time.Time `gorm:"not null"` // date on which exchange was intimated
	NseDate         time.Time `gorm:"not null"` // nse date of publishing

	// Number of securities involved.
	Quantity int64 `gorm:"not null"`

	// Reported transaction value.
	// Use decimal rather than float for financial data.
	Value float64 `gorm:"type:decimal(20,4)"`

	// Holding before transaction.
	HoldingBefore int64 `gorm:"not null;default:0"`
	HoldingAfter  int64 `gorm:"not null;default:0"`

	// Holding after transaction.
	HoldingBeforePct float64 `gorm:"type:decimal(12,6)"`
	HoldingAfterPct  float64 `gorm:"type:decimal(12,6)"`

	// Exchange reported value for trade execution, if applicable.
	Exchange string `gorm:"type:varchar(20)"`

	// Optional free-text notes from the filing.
	Notes string `gorm:"type:text"`

	DerevativeType   string `gorm:"type:varchar(255)"`
	SecurityTypePost string `gorm:"type:varchar(255)"`

	// Audit timestamps.
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type InsiderTradeEntity struct {
	ID       uint   `gorm:"primaryKey"`
	Name     string `gorm:"not null;uniqueIndex"`
	Category string `gorm:"index"`
}

type InsiderTradeWithEntity struct {
	InsiderTradeTransaction
	EntityName     string
	EntityCategory string
}
