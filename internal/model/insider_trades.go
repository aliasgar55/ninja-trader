package models

import (
	"time"

	"gorm.io/gorm"
)

type InsiderTradeTransaction struct {
	ID uint `gorm:"primaryKey"`

	// Filing from which this transaction was extracted.
	FilingID uint `gorm:"not null;index"`

	// Company / security identification.
	CompanyID uint   `gorm:"not null;index"`
	ISIN      string `gorm:"type:varchar(12);not null;index"`

	// Person / entity involved in the transaction.
	EntityID uint `gorm:"not null;index"`

	// Security information.
	SecurityType string `gorm:"type:varchar(30);not null"`

	// Transaction classification.
	TransactionType string `gorm:"type:varchar(30);not null;index"`
	TransactionMode string `gorm:"type:varchar(30);not null;index"`

	// Actual transaction date.
	TransactionDate time.Time `gorm:"not null;index"`

	// Number of securities involved.
	Quantity int64 `gorm:"not null"`

	// Reported transaction value.
	// Use decimal rather than float for financial data.
	Value    *float64 `gorm:"type:decimal(20,4)"`
	Currency string   `gorm:"type:char(3);default:'INR'"`

	// Holding before transaction.
	HoldingBefore    int64    `gorm:"not null;default:0"`
	HoldingBeforePct *float64 `gorm:"type:decimal(12,6)"`

	// Holding after transaction.
	HoldingAfter    int64    `gorm:"not null;default:0"`
	HoldingAfterPct *float64 `gorm:"type:decimal(12,6)"`

	// Derived ownership change.
	OwnershipChange    int64    `gorm:"not null;default:0"`
	OwnershipPctChange *float64 `gorm:"type:decimal(12,6)"`

	// Original values exactly as reported by NSE.
	RawTransactionType string `gorm:"type:varchar(100)"`
	RawTransactionMode string `gorm:"type:varchar(100)"`

	// Exchange reported value for trade execution, if applicable.
	Exchange string `gorm:"type:varchar(20)"`

	// Optional free-text notes from the filing.
	Notes *string `gorm:"type:text"`

	// Audit timestamps.
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type InsiderEntity struct {
    ID       uint   `gorm:"primaryKey"`
    Name     string `gorm:"not null;index"`
    Category string `gorm:"index"`
    CINDIN   string `gorm:"index"`
}
