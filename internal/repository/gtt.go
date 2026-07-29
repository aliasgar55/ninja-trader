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
	DiffPercent        float64
}

func (r *GTTRepo) GetAll() ([]models.GTTOrder, error) {
	var orders []models.GTTOrder
	err := r.Db.Order("created_at DESC").Find(&orders).Error
	return orders, err
}

func (r *GTTRepo) GetAllWithPrice() ([]GTTOrderWithPrice, error) {
	var orders []GTTOrderWithPrice
	err := r.Db.Raw(`
		SELECT g.*, COALESCE(h.adjusted_close_price, 0) AS adjusted_close_price,
			CASE WHEN COALESCE(h.adjusted_close_price, 0) > 0
				THEN ((g.trigger_price - h.adjusted_close_price) / h.adjusted_close_price) * 100
				ELSE 0 END AS diff_percent
		FROM gtt_orders g
		LEFT JOIN historicaldata h ON h.symbol = g.trading_symbol
			AND h.date = (SELECT MAX(date) FROM historicaldata WHERE symbol = g.trading_symbol)
		ORDER BY CASE g.status WHEN 'active' THEN 0 WHEN 'triggered' THEN 1 ELSE 2 END ASC,
			ABS(CASE WHEN COALESCE(h.adjusted_close_price, 0) > 0
			THEN ((g.trigger_price - h.adjusted_close_price) / h.adjusted_close_price) * 100
			ELSE 0 END) ASC
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

func (r *GTTRepo) Delete(triggerID int) error {
	return r.Db.Where("trigger_id = ?", triggerID).Delete(&models.GTTOrder{}).Error
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
		"status":              order.Status,
		"last_price":          order.LastPrice,
		"trigger_price":       order.TriggerPrice,
		"limit_price":         order.LimitPrice,
		"quantity":            order.Quantity,
		"upper_trigger_price": order.UpperTriggerPrice,
		"upper_limit_price":   order.UpperLimitPrice,
		"upper_quantity":      order.UpperQuantity,
	}).Error
}
