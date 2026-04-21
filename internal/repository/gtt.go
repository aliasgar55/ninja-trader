package repo

import (
	models "ninja-trader/internal/model"

	"gorm.io/gorm"
)

type GTTRepo struct {
	Db *gorm.DB
}

func (r *GTTRepo) Create(order *models.GTTOrder) error {
	return r.Db.Create(order).Error
}

type GTTOrderWithPrice struct {
	models.GTTOrder
	AdjustedClosePrice float64
}

func (r *GTTRepo) GetAll() ([]models.GTTOrder, error) {
	var orders []models.GTTOrder
	err := r.Db.Order("created_at DESC").Find(&orders).Error
	return orders, err
}

func (r *GTTRepo) GetAllWithPrice() ([]GTTOrderWithPrice, error) {
	var orders []GTTOrderWithPrice
	err := r.Db.Raw(`
		SELECT g.*, COALESCE(h.adjusted_close_price, 0) AS adjusted_close_price
		FROM gtt_orders g
		LEFT JOIN historicaldata h ON h.symbol = g.trading_symbol
			AND h.date = (SELECT MAX(date) FROM historicaldata WHERE symbol = g.trading_symbol)
		ORDER BY g.created_at DESC
	`).Scan(&orders).Error
	return orders, err
}

func (r *GTTRepo) GetBySymbol(symbol string) ([]models.GTTOrder, error) {
	var orders []models.GTTOrder
	err := r.Db.Where("trading_symbol = ?", symbol).Order("created_at DESC").Find(&orders).Error
	return orders, err
}

func (r *GTTRepo) GetByTriggerID(triggerID int) (*models.GTTOrder, error) {
	var order models.GTTOrder
	err := r.Db.Where("trigger_id = ?", triggerID).First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *GTTRepo) UpdateStatus(triggerID int, status string) error {
	return r.Db.Model(&models.GTTOrder{}).Where("trigger_id = ?", triggerID).Update("status", status).Error
}

func (r *GTTRepo) Update(triggerID int, updates map[string]interface{}) error {
	return r.Db.Model(&models.GTTOrder{}).Where("trigger_id = ?", triggerID).Updates(updates).Error
}

func (r *GTTRepo) Upsert(order *models.GTTOrder) error {
	var existing models.GTTOrder
	err := r.Db.Where("trigger_id = ?", order.TriggerID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.Db.Create(order).Error
	}
	if err != nil {
		return err
	}
	return r.Db.Model(&existing).Updates(map[string]interface{}{
		"status":     order.Status,
		"last_price": order.LastPrice,
	}).Error
}
