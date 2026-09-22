package repo

import (
	models "ninja-trader/internal/model"

	"gorm.io/gorm"
)

type TradeRepo struct {
	Db *gorm.DB
}

func (repo *TradeRepo) GetTradeBySymbol(symbol string) (*models.PaperTrade, error) {
	var trade models.PaperTrade
	err := repo.Db.Where("trading_symbol = ?", symbol).First(&trade).Error
	if err != nil {
		return nil, err
	}
	return &trade, nil
}

func (repo *TradeRepo) GetHoldingByInstrumentID(instrumentID uint) (int, error) {
	var trade models.PaperTrade
	err := repo.Db.Where("instrument_id = ?", instrumentID).First(&trade).Error
	if err != nil {
		return 0, err
	}
	return trade.Quantity, nil
}

func (repo *TradeRepo) GetAllTrades() ([]models.PaperTrade, error) {
	var trades []models.PaperTrade
	err := repo.Db.Order("quantity DESC").Find(&trades).Error
	return trades, err
}

type TradeWithPnL struct {
	models.PaperTrade
	CurrentPrice   float64
	PnL            float64
	InvestedAmount float64
	CurrentValue   float64
	PnLPct         float64
	Tag            string
	BasicIndustry  string
	Index          string
	VptScore       float64
}

func (repo *TradeRepo) GetAllTradesWithPnL() ([]TradeWithPnL, error) {
	var results []TradeWithPnL
	err := repo.Db.
		Model(&models.PaperTrade{}).
		Select(`paper_trades.*,
      h.c AS current_price,
      paper_trades.average_price * ABS(paper_trades.quantity) AS invested_amount,
      h.c * ABS(paper_trades.quantity) AS current_value,
      (h.c - paper_trades.average_price) * paper_trades.quantity AS pn_l,
      CASE WHEN paper_trades.quantity != 0 THEN SIGN(paper_trades.quantity)*((h.c - paper_trades.average_price) / paper_trades.average_price) * 100 ELSE 0 END AS pn_l_pct,
      i.tag AS tag,
			i.basic_industry AS basic_industry,
      i."index" AS "index",
      COALESCE(h.vpt_score, 0) AS vpt_score`).
		Joins(`JOIN LATERAL (
      SELECT c, vpt_score FROM historicaldata
      WHERE symbol = paper_trades.trading_symbol
      ORDER BY date DESC LIMIT 1
    ) h ON true`).
		Joins(`LEFT JOIN instruments i ON i.id = paper_trades.instrument_id`).
		Order("pn_l_pct DESC").
		Scan(&results).Error
	return results, err
}

func (repo *TradeRepo) GetLogsBySymbol(symbol string) ([]models.PaperTradeLog, error) {
	var logs []models.PaperTradeLog
	err := repo.Db.Where("trading_symbol = ?", symbol).Order("created_at DESC").Find(&logs).Error
	return logs, err
}

func (repo *TradeRepo) UpsertTrade(trade *models.PaperTrade, log *models.PaperTradeLog) error {
	return repo.Db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(trade).Error; err != nil {
			return err
		}
		return tx.Create(log).Error
	})
}
