package database

import (
	"errors"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewDatabase(dsn string) (*gorm.DB, error) {
	if dsn == "" {
		return nil, errors.New("database: DSN must not be empty")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:      logger.Default.LogMode(logger.Warn),
		PrepareStmt: true,
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// Max open connections — tune to (2 * num_cores) or based on Postgres max_connections
	sqlDB.SetMaxOpenConns(25)
	// Idle connections kept alive to avoid reconnect overhead
	sqlDB.SetMaxIdleConns(10)
	// Recycle connections to avoid stale/broken connections from the server side
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	// Evict idle connections that haven't been used recently
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, err
	}
	return db, nil
}

