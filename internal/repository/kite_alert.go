package repo

import (
	models "ninja-trader/internal/model"

	"gorm.io/gorm"
)

type KiteAlertRepo struct {
	Db *gorm.DB
}

func (r *KiteAlertRepo) Create(alert *models.KiteAlert) error {
	return r.Db.Create(alert).Error
}

func (r *KiteAlertRepo) GetAll() ([]models.KiteAlert, error) {
	var alerts []models.KiteAlert
	err := r.Db.Where("status != ?", "deleted").Order("created_at DESC").Find(&alerts).Error
	return alerts, err
}

func (r *KiteAlertRepo) GetBySymbol(symbol string) ([]models.KiteAlert, error) {
	var alerts []models.KiteAlert
	err := r.Db.Where("trading_symbol = ? AND status != ?", symbol, "deleted").Order("created_at DESC").Find(&alerts).Error
	return alerts, err
}

func (r *KiteAlertRepo) UpdateStatus(alertID string, status string) error {
	return r.Db.Model(&models.KiteAlert{}).Where("alert_id = ?", alertID).Update("status", status).Error
}

func (r *KiteAlertRepo) Delete(alertID string) error {
	return r.Db.Where("alert_id = ?", alertID).Delete(&models.KiteAlert{}).Error
}

func (r *KiteAlertRepo) Upsert(alert *models.KiteAlert) error {
	var existing models.KiteAlert
	err := r.Db.Where("alert_id = ?", alert.AlertID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.Db.Create(alert).Error
	}
	return r.Db.Model(&existing).Updates(map[string]interface{}{
		"status":        alert.Status,
		"trigger_value": alert.TriggerValue,
		"operator":      alert.Operator,
	}).Error
}
