package repo

import (
	models "ninja-trader/internal/model"
	"time"

	"gorm.io/gorm"
)

type BulkBlockDealRepo struct {
	Db *gorm.DB
}

func (repo *BulkBlockDealRepo) Create(deal *models.BulkBlockDeal) error {
	return repo.Db.Create(deal).Error
}

func (repo *BulkBlockDealRepo) GetBySymbol(symbol string) ([]models.BulkBlockDeal, error) {
	var results []models.BulkBlockDeal
	err := repo.Db.Where("trading_symbol = ?", symbol).
		Order("date DESC").
		Find(&results).Error
	return results, err
}

func (repo *BulkBlockDealRepo) GetByDateRange(from, to time.Time) ([]models.BulkBlockDeal, error) {
	var results []models.BulkBlockDeal
	err := repo.Db.Where("date BETWEEN ? AND ?", from, to).
		Order("date DESC, trading_symbol ASC").
		Find(&results).Error
	return results, err
}

func (repo *BulkBlockDealRepo) GetBySymbolAndType(symbol, dealType string) ([]models.BulkBlockDeal, error) {
	var results []models.BulkBlockDeal
	err := repo.Db.Where("trading_symbol = ? AND deal_type = ?", symbol, dealType).
		Order("date DESC").
		Find(&results).Error
	return results, err
}

func (repo *BulkBlockDealRepo) DeleteByDateRangeAndType(from, to time.Time, dealType string) error {
	return repo.Db.Where("date BETWEEN ? AND ? AND deal_type = ?", from, to, dealType).
		Delete(&models.BulkBlockDeal{}).Error
}

func (repo *BulkBlockDealRepo) GetLastDealDate(dealType string) (time.Time, error) {
	var deal models.BulkBlockDeal
	err := repo.Db.Where("deal_type = ?", dealType).Order("date DESC").First(&deal).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}
	return deal.Date, nil
}
