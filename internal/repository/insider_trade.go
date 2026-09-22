package repo

import (
	models "ninja-trader/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type InsiderTradesRepo struct {
	Db *gorm.DB
}

func (repo *InsiderTradesRepo) CreateInsiderTrade(transaction *models.InsiderTradeTransaction, entity *models.InsiderTradeEntity) error {
	return repo.Db.Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "name"}},
			DoNothing: true,
		}).Create(entity)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			if err := tx.Where("name = ?", entity.Name).First(entity).Error; err != nil {
				return err
			}
		}
		transaction.EntityID = entity.ID
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "filing_id"}},
			DoNothing: true,
		}).Create(transaction).Error
	})
}

func (repo *InsiderTradesRepo) GetBySymbol(symbol string) ([]models.InsiderTradeWithEntity, error) {
	var results []models.InsiderTradeWithEntity
	err := repo.Db.Table("insider_trade_transactions").
		Select("insider_trade_transactions.*, insider_trade_entities.name as entity_name, insider_trade_entities.category as entity_category").
		Joins("LEFT JOIN insider_trade_entities ON insider_trade_entities.id = insider_trade_transactions.entity_id").
		Where("insider_trade_transactions.symbol = ? AND insider_trade_transactions.deleted_at IS NULL", symbol).
		Order("insider_trade_transactions.transaction_date DESC").
		Find(&results).Error
	return results, err
}

func (repo *InsiderTradesRepo) GetAll(limit, offset int) ([]models.InsiderTradeWithEntity, int64, error) {
	var total int64
	repo.Db.Table("insider_trade_transactions").Where("deleted_at IS NULL").Count(&total)
	var results []models.InsiderTradeWithEntity
	err := repo.Db.Table("insider_trade_transactions").
		Select("insider_trade_transactions.*, insider_trade_entities.name as entity_name, insider_trade_entities.category as entity_category").
		Joins("LEFT JOIN insider_trade_entities ON insider_trade_entities.id = insider_trade_transactions.entity_id").
		Where("insider_trade_transactions.deleted_at IS NULL").
		Order("insider_trade_transactions.transaction_date DESC").
		Limit(limit).Offset(offset).
		Find(&results).Error
	return results, total, err
}
