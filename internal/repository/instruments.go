package repo

import (
	"fmt"
	models "ninja-trader/internal/model"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type InstrumentRepo struct {
	Db *gorm.DB
}

func (repo *InstrumentRepo) CreateInstrument(instrument *models.Instrument) error {
	return repo.Db.Clauses(
		clause.OnConflict{
			Columns: []clause.Column{
				{Name: "instrument_token"},
			},
			DoUpdates: clause.AssignmentColumns([]string{"market_cap", "nse_api_success", "is_nav", "instrument_full_name", "active"}),
		},
	).Create(instrument).Error
}

func (repo *InstrumentRepo) GetAllInstruments() ([]models.Instrument, error) {
	var instruments []models.Instrument
	err := repo.Db.
		Order(clause.OrderByColumn{Column: clause.Column{Name: "market_cap"}, Desc: true}).
		Where(&models.Instrument{Active: true}).
		Find(&instruments).
		Error
	return instruments, err
}

type InstrumentWithHolding struct {
	models.Instrument
	Holding            int
	DeliveryPercentage float32
	VolumePerTrade     int64
	VptScore           float64
	Divergence         float64
	DeliveryValue      float64
	VolRatio           float64
	Range3M            float64 `gorm:"column:range_3m"`
	PctFrom52WLow      float64 `gorm:"column:pct_from_52w_low"`
	PctChange          float64 `gorm:"column:pct_change"`
}

func (repo *InstrumentRepo) withHoldingQuery() *gorm.DB {
	return repo.Db.Model(&models.Instrument{}).
		Select("instruments.*, COALESCE(pt.quantity, 0) AS holding, COALESCE(hd.delivery_percentage, 0) AS delivery_percentage, COALESCE(hd.volume_per_trade, 0) AS volume_per_trade, COALESCE(hd.vpt_score, 0) AS vpt_score, COALESCE(hd.divergence, 0) AS divergence, COALESCE(hd.delivery_value, 0) AS delivery_value, CASE WHEN COALESCE(hd.volume_ma20, 0) > 0 THEN hd.volume::float / hd.volume_ma20 ELSE 0 END AS vol_ratio, COALESCE(r3.range_pct, 0) AS range_3m, CASE WHEN COALESCE(hd.year_low, 0) > 0 THEN (hd.adjusted_close_price - hd.year_low) / hd.year_low * 100 ELSE 0 END AS pct_from_52w_low").
		Joins("LEFT JOIN paper_trades pt ON pt.instrument_id = instruments.id").
		Joins("LEFT JOIN historicaldata hd ON hd.symbol = instruments.trading_symbol AND hd.date = (SELECT MAX(h2.date) FROM historicaldata h2 WHERE h2.symbol = instruments.trading_symbol)").
		Joins(`LEFT JOIN LATERAL (
			SELECT (MAX(adjusted_close_price) - MIN(adjusted_close_price)) / NULLIF(MIN(adjusted_close_price), 0) * 100 AS range_pct
			FROM historicaldata WHERE symbol = instruments.trading_symbol AND date >= NOW() - INTERVAL '90 days'
		) r3 ON true`)
}

var allowedSortColumns = map[string]string{
	"symbol":           "instruments.trading_symbol",
	"market_cap":       "instruments.market_cap",
	"industry":         "instruments.basic_industry",
	"index":            "instruments.\"index\"",
	"price_band":       "instruments.price_band",
	"delivery_pct":     "delivery_percentage",
	"tag":              "instruments.tag",
	"holding":          "holding",
	"name":             "instruments.instrument_full_name",
	"active":           "instruments.active",
	"vpt":              "volume_per_trade",
	"vpt_score":        "vpt_score",
	"divergence":       "divergence",
	"delivery_value":   "delivery_value",
	"vol_ratio":        "vol_ratio",
	"range_3m":         "range_3m",
	"pct_from_52w_low": "pct_from_52w_low",
	"pct_change":       "pct_change",
	"is_fno_sec":       "instruments.is_fno_sec",
}

func buildSortOrder(sort, order string) string {
	col, ok := allowedSortColumns[sort]
	if !ok {
		return "instruments.market_cap DESC, instruments.trading_symbol ASC"
	}
	dir := "ASC"
	if order == "desc" {
		dir = "DESC"
	}
	return col + " " + dir + ", instruments.market_cap DESC"
}

func (repo *InstrumentRepo) withHoldingQueryForDate(date string) *gorm.DB {
	dateClause := "hd.date = (SELECT MAX(h2.date) FROM historicaldata h2 WHERE h2.symbol = instruments.trading_symbol)"
	if date != "" {
		dateClause = "hd.date::date = '" + date + "'"
	}
	return repo.Db.Model(&models.Instrument{}).
		Select("instruments.*, COALESCE(pt.quantity, 0) AS holding, COALESCE(hd.delivery_percentage, 0) AS delivery_percentage, COALESCE(hd.volume_per_trade, 0) AS volume_per_trade, COALESCE(hd.vpt_score, 0) AS vpt_score, COALESCE(hd.divergence, 0) AS divergence, COALESCE(hd.delivery_value, 0) AS delivery_value, CASE WHEN COALESCE(hd.volume_ma20, 0) > 0 THEN hd.volume::float / hd.volume_ma20 ELSE 0 END AS vol_ratio, COALESCE(r3.range_pct, 0) AS range_3m, CASE WHEN COALESCE(hd.year_low, 0) > 0 THEN (hd.adjusted_close_price - hd.year_low) / hd.year_low * 100 ELSE 0 END AS pct_from_52w_low").
		Joins("LEFT JOIN paper_trades pt ON pt.instrument_id = instruments.id").
		Joins("LEFT JOIN historicaldata hd ON hd.symbol = instruments.trading_symbol AND " + dateClause).
		Joins(`LEFT JOIN LATERAL (
		  SELECT (MAX(adjusted_close_price) - MIN(adjusted_close_price)) / NULLIF(MIN(adjusted_close_price), 0) * 100 AS range_pct
		  FROM historicaldata WHERE symbol = instruments.trading_symbol AND date >= NOW() - INTERVAL '90 days'
		) r3 ON true`)
}

func (repo *InstrumentRepo) GetAllInstrumentsWithHoldingForDate(query, sort, order, date string) ([]InstrumentWithHolding, error) {
	var results []InstrumentWithHolding
	q := repo.withHoldingQueryForDate(date).Where("instruments.active = true")
	if query != "" {
		q = q.Where("instruments.trading_symbol ILIKE ?", "%"+query+"%")
	}
	err := q.Order(buildSortOrder(sort, order)).Scan(&results).Error
	return results, err
}

func (repo *InstrumentRepo) GetAllInstrumentsWithHolding(query, sort, order string) ([]InstrumentWithHolding, error) {
	var results []InstrumentWithHolding
	q := repo.withHoldingQuery().Where("instruments.active = true")
	if query != "" {
		q = q.Where("instruments.trading_symbol ILIKE ?", "%"+query+"%")
	}
	err := q.Order(buildSortOrder(sort, order)).Scan(&results).Error
	return results, err
}

func (repo *InstrumentRepo) GetWatchlistInstrumentsWithHolding(query, sort, order string) ([]InstrumentWithHolding, error) {
	var results []InstrumentWithHolding
	q := repo.withHoldingQuery().
		Joins(`LEFT JOIN LATERAL (
			SELECT adjusted_close_price AS wl_close
			FROM historicaldata
			WHERE symbol = instruments.trading_symbol AND date >= instruments.watch_list_date
			ORDER BY date ASC LIMIT 1
		) wl ON true`).
		Where("instruments.active = true AND instruments.watchlist = true AND instruments.is_nav = false")
	q = q.Select("instruments.*, COALESCE(pt.quantity, 0) AS holding, COALESCE(hd.delivery_percentage, 0) AS delivery_percentage, COALESCE(hd.volume_per_trade, 0) AS volume_per_trade, COALESCE(hd.vpt_score, 0) AS vpt_score, COALESCE(hd.divergence, 0) AS divergence, COALESCE(hd.delivery_value, 0) AS delivery_value, CASE WHEN COALESCE(hd.volume_ma20, 0) > 0 THEN hd.volume::float / hd.volume_ma20 ELSE 0 END AS vol_ratio, COALESCE(r3.range_pct, 0) AS range_3m, CASE WHEN COALESCE(hd.year_low, 0) > 0 THEN (hd.adjusted_close_price - hd.year_low) / hd.year_low * 100 ELSE 0 END AS pct_from_52w_low, CASE WHEN COALESCE(wl.wl_close, 0) > 0 THEN (hd.adjusted_close_price - wl.wl_close) / wl.wl_close * 100 ELSE 0 END AS pct_change")
	if query != "" {
		q = q.Where("instruments.trading_symbol ILIKE ?", "%"+query+"%")
	}
	sortOrder := buildSortOrder(sort, order)
	if sort == "" {
		sortOrder = "instruments.watch_list_date DESC NULLS LAST, instruments.trading_symbol ASC"
	}
	err := q.Order(sortOrder).Scan(&results).Error
	return results, err
}

type InstrumentWithRange struct {
	models.Instrument
	Range1M  float64 `gorm:"column:range_1m"`
	Range3M  float64 `gorm:"column:range_3m"`
	Range6M  float64 `gorm:"column:range_6m"`
	Range1Y  float64 `gorm:"column:range_1y"`
	VptScore float64 `gorm:"column:vpt_score"`
}

var rangeSortColumns = map[string]string{
	"symbol":     "instruments.trading_symbol",
	"market_cap": "instruments.market_cap",
	"industry":   "instruments.basic_industry",
	"index":      "instruments.\"index\"",
	"tag":        "instruments.tag",
	"range_1m":   "range_1m",
	"range_3m":   "range_3m",
	"range_6m":   "range_6m",
	"range_1y":   "range_1y",
	"vpt_score":  "vpt_score",
}

func buildRangeSortOrder(sort, order string) string {
	col, ok := rangeSortColumns[sort]
	if !ok {
		return "range_1y ASC, instruments.trading_symbol ASC"
	}
	dir := "ASC"
	if order == "desc" {
		dir = "DESC"
	}
	return col + " " + dir + ", instruments.trading_symbol ASC"
}

func (repo *InstrumentRepo) GetInstrumentsWithRange(query, sort, order string) ([]InstrumentWithRange, error) {
	var results []InstrumentWithRange
	q := repo.Db.Model(&models.Instrument{}).
		Select(`instruments.*,
      COALESCE(r1.range_pct, 0) AS range_1m,
      COALESCE(r3.range_pct, 0) AS range_3m,
      COALESCE(r6.range_pct, 0) AS range_6m,
      COALESCE(r12.range_pct, 0) AS range_1y,
      COALESCE(hd.vpt_score, 0) AS vpt_score`).
		Joins(`LEFT JOIN historicaldata hd ON hd.symbol = instruments.trading_symbol AND hd.date = (SELECT MAX(h2.date) FROM historicaldata h2 WHERE h2.symbol = instruments.trading_symbol)`).
		Joins(`LEFT JOIN LATERAL (
			SELECT (MAX(adjusted_close_price) - MIN(adjusted_close_price)) / NULLIF(MIN(adjusted_close_price), 0) * 100 AS range_pct
			FROM historicaldata WHERE symbol = instruments.trading_symbol AND date >= NOW() - INTERVAL '30 days'
		) r1 ON true`).
		Joins(`LEFT JOIN LATERAL (
			SELECT (MAX(adjusted_close_price) - MIN(adjusted_close_price)) / NULLIF(MIN(adjusted_close_price), 0) * 100 AS range_pct
			FROM historicaldata WHERE symbol = instruments.trading_symbol AND date >= NOW() - INTERVAL '90 days'
		) r3 ON true`).
		Joins(`LEFT JOIN LATERAL (
			SELECT (MAX(adjusted_close_price) - MIN(adjusted_close_price)) / NULLIF(MIN(adjusted_close_price), 0) * 100 AS range_pct
			FROM historicaldata WHERE symbol = instruments.trading_symbol AND date >= NOW() - INTERVAL '180 days'
		) r6 ON true`).
		Joins(`LEFT JOIN LATERAL (
			SELECT (MAX(adjusted_close_price) - MIN(adjusted_close_price)) / NULLIF(MIN(adjusted_close_price), 0) * 100 AS range_pct
			FROM historicaldata WHERE symbol = instruments.trading_symbol AND date >= NOW() - INTERVAL '365 days'
		) r12 ON true`).
		Where("instruments.active = true")
	if query != "" {
		q = q.Where("instruments.trading_symbol ILIKE ?", "%"+query+"%")
	}
	err := q.Order(buildRangeSortOrder(sort, order)).Scan(&results).Error
	return results, err
}

func (repo *InstrumentRepo) GetAdjacentRangeSymbols(symbol, sort, order string) (prev, next string) {
	var results []InstrumentWithRange
	q := repo.Db.Model(&models.Instrument{}).
		Select(`instruments.trading_symbol,
			COALESCE(r1.range_pct, 0) AS range_1m,
			COALESCE(r3.range_pct, 0) AS range_3m,
			COALESCE(r6.range_pct, 0) AS range_6m,
			COALESCE(r12.range_pct, 0) AS range_1y,
			COALESCE(hd.vpt_score, 0) AS vpt_score`).
		Joins(`LEFT JOIN historicaldata hd ON hd.symbol = instruments.trading_symbol AND hd.date = (SELECT MAX(h2.date) FROM historicaldata h2 WHERE h2.symbol = instruments.trading_symbol)`).
		Joins(`LEFT JOIN LATERAL (
			SELECT (MAX(adjusted_close_price) - MIN(adjusted_close_price)) / NULLIF(MIN(adjusted_close_price), 0) * 100 AS range_pct
			FROM historicaldata WHERE symbol = instruments.trading_symbol AND date >= NOW() - INTERVAL '30 days'
		) r1 ON true`).
		Joins(`LEFT JOIN LATERAL (
			SELECT (MAX(adjusted_close_price) - MIN(adjusted_close_price)) / NULLIF(MIN(adjusted_close_price), 0) * 100 AS range_pct
			FROM historicaldata WHERE symbol = instruments.trading_symbol AND date >= NOW() - INTERVAL '90 days'
		) r3 ON true`).
		Joins(`LEFT JOIN LATERAL (
			SELECT (MAX(adjusted_close_price) - MIN(adjusted_close_price)) / NULLIF(MIN(adjusted_close_price), 0) * 100 AS range_pct
			FROM historicaldata WHERE symbol = instruments.trading_symbol AND date >= NOW() - INTERVAL '180 days'
		) r6 ON true`).
		Joins(`LEFT JOIN LATERAL (
			SELECT (MAX(adjusted_close_price) - MIN(adjusted_close_price)) / NULLIF(MIN(adjusted_close_price), 0) * 100 AS range_pct
			FROM historicaldata WHERE symbol = instruments.trading_symbol AND date >= NOW() - INTERVAL '365 days'
		) r12 ON true`).
		Where("instruments.active = true")
	q.Order(buildRangeSortOrder(sort, order)).Find(&results)
	for i, r := range results {
		if r.TradingSymbol == symbol {
			if i > 0 {
				prev = results[i-1].TradingSymbol
			}
			if i < len(results)-1 {
				next = results[i+1].TradingSymbol
			}
			return
		}
	}
	return
}

func (repo *InstrumentRepo) BulkUpdateAdjustedClosePrice(updates []models.AdjustedCloseUpdate) error {
	for _, u := range updates {
		if err := repo.Db.Model(&models.Historicaldata{}).
			Where("symbol = ? AND date::date = ?::date", u.Symbol, u.Date).
			Update("adjusted_close_price", u.AdjustedClosePrice).Error; err != nil {
			return err
		}
	}
	return nil
}

func (repo *InstrumentRepo) BulkUpdateSignals(updates []models.SignalUpdate) error {
	batchSize := 500
	for i := 0; i < len(updates); i += batchSize {
		end := i + batchSize
		if end > len(updates) {
			end = len(updates)
		}
		batch := updates[i:end]

		valuesClauses := make([]string, len(batch))
		args := make([]interface{}, 0, len(batch)*6)
		for j, u := range batch {
			valuesClauses[j] = fmt.Sprintf("($%d::bigint, $%d::numeric, $%d::numeric, $%d::numeric, $%d::numeric, $%d::numeric)", j*6+1, j*6+2, j*6+3, j*6+4, j*6+5, j*6+6)
			args = append(args, u.ID, u.VptMa20, u.VptScore, u.Divergence, u.DivergenceMax3y, u.VolumeMa20)
		}

		sql := fmt.Sprintf(`UPDATE historicaldata AS h SET
			vpt_ma20 = v.vpt_ma20,
			vpt_score = v.vpt_score,
			divergence = v.divergence,
			divergence_max3y = v.divergence_max3y,
			volume_ma20 = v.volume_ma20
			FROM (VALUES %s) AS v(id, vpt_ma20, vpt_score, divergence, divergence_max3y, volume_ma20)
			WHERE h.id = v.id`, strings.Join(valuesClauses, ","))

		if err := repo.Db.Exec(sql, args...).Error; err != nil {
			return err
		}
	}
	return nil
}

func (repo *InstrumentRepo) BulkInsertTradeHistory(trades []models.Historicaldata) error {
	return repo.Db.Clauses(clause.OnConflict{
		DoNothing: true,
	}).CreateInBatches(&trades, 100).Error
}

func (repo *InstrumentRepo) BulkInsertEvents(events []models.Event) error {
	return repo.Db.Clauses(clause.OnConflict{
		DoNothing: true,
	}).CreateInBatches(&events, 100).Error
}

// func (repo *InstrumentRepo

func (repo *InstrumentRepo) GetEventsBySymbol(symbol string) ([]models.Event, error) {
	var events []models.Event
	err := repo.Db.Where("trading_symbol = ?", symbol).Order("event_date ASC").Find(&events).Error
	return events, err
}

func (repo *InstrumentRepo) GetPreviousTrade(symbol string, date time.Time) (*models.Historicaldata, error) {
	var data models.Historicaldata
	err := repo.Db.Where("symbol = ? AND date < ? AND adjusted_close_price IS NOT NULL AND adjusted_close_price != 0", symbol, date).Order("date DESC").First(&data).Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (repo *InstrumentRepo) GetTradesWithNullAdjustedClose(symbol string) ([]models.Historicaldata, error) {
	var data []models.Historicaldata
	err := repo.Db.Where("symbol = ? AND (adjusted_close_price IS NULL OR adjusted_close_price = 0)", symbol).Order("date ASC").Find(&data).Error
	return data, err
}

func (repo *InstrumentRepo) GetHistoricalDataBySymbol(symbol string) ([]models.Historicaldata, error) {
	var data []models.Historicaldata
	err := repo.Db.Where("symbol = ?", symbol).Order("date ASC").Find(&data).Error
	return data, err
}

func (repo *InstrumentRepo) GetHistoricalDataBySymbolAndDate(symbol string, date time.Time) (*models.Historicaldata, error) {
	var data models.Historicaldata
	err := repo.Db.Where("symbol = ? AND date::date = ?::date", symbol, date).First(&data).Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (repo *InstrumentRepo) GetHistoricalDataBySymbolSince(symbol string, since time.Time) ([]models.Historicaldata, error) {
	var data []models.Historicaldata
	err := repo.Db.Where("symbol = ? AND date >= ?", symbol, since).Order("date ASC").Find(&data).Error
	return data, err
}

func (repo *InstrumentRepo) GetLatestDeliveryPercentage(symbol string) (float32, error) {
	var data models.Historicaldata
	err := repo.Db.Where("symbol = ?", symbol).Order("date DESC").First(&data).Error
	if err != nil {
		return 0, err
	}
	return data.DeliveryPercentage, nil
}

type HistoricalDateRange struct {
	MinDate time.Time
	MaxDate time.Time
}

func (repo *InstrumentRepo) GetHistoricalDataDateRange(symbol string) (*HistoricalDateRange, error) {
	var result HistoricalDateRange
	err := repo.Db.Model(&models.Historicaldata{}).
		Select("MIN(date) as min_date, MAX(date) as max_date").
		Where("symbol = ?", symbol).
		Scan(&result).Error
	return &result, err
}

func (repo *InstrumentRepo) GetHistoricalYears(symbol string) ([]int, error) {
	var years []int
	err := repo.Db.Model(&models.Historicaldata{}).
		Select("DISTINCT EXTRACT(YEAR FROM date)::int AS year").
		Where("symbol = ?", symbol).
		Order("year DESC").
		Pluck("year", &years).Error
	return years, err
}

func (repo *InstrumentRepo) SearchInstruments(query string) ([]models.Instrument, error) {
	var instruments []models.Instrument
	err := repo.Db.
		Where("active = ? AND trading_symbol ILIKE ?", true, "%"+query+"%").
		Order("market_cap DESC").
		Find(&instruments).Error
	return instruments, err
}

func (repo *InstrumentRepo) GetInstrumentBySymbol(symbol string) (*models.Instrument, error) {
	var instrument models.Instrument
	err := repo.Db.Where("trading_symbol = ? AND active = ?", symbol, true).First(&instrument).Error
	if err != nil {
		return nil, err
	}
	return &instrument, nil
}

func (repo *InstrumentRepo) adjacentSymbols(symbol, sort, order string, where string, args ...interface{}) (prev, next string) {
	type result struct {
		TradingSymbol string
	}
	var results []result
	repo.withHoldingQuery().
		Where(where, args...).
		Order(buildSortOrder(sort, order)).
		Find(&results)
	for i, r := range results {
		if r.TradingSymbol == symbol {
			if i > 0 {
				prev = results[i-1].TradingSymbol
			}
			if i < len(results)-1 {
				next = results[i+1].TradingSymbol
			}
			return
		}
	}
	return
}

func (repo *InstrumentRepo) adjacentSymbolsForDate(symbol, sort, order, date string, where string, args ...interface{}) (prev, next string) {
	type result struct {
		TradingSymbol string
	}
	var results []result
	repo.withHoldingQueryForDate(date).
		Where(where, args...).
		Order(buildSortOrder(sort, order)).
		Find(&results)
	for i, r := range results {
		if r.TradingSymbol == symbol {
			if i > 0 {
				prev = results[i-1].TradingSymbol
			}
			if i < len(results)-1 {
				next = results[i+1].TradingSymbol
			}
			return
		}
	}
	return
}

func (repo *InstrumentRepo) GetAdjacentSymbols(symbol, sort, order string) (prev, next string) {
	return repo.adjacentSymbols(symbol, sort, order, "instruments.active = ?", true)
}

func (repo *InstrumentRepo) GetAdjacentSymbolsForDate(symbol, sort, order, date string) (prev, next string) {
	return repo.adjacentSymbolsForDate(symbol, sort, order, date, "instruments.active = ?", true)
}

func (repo *InstrumentRepo) GetAdjacentWatchlistSymbols(symbol, sort, order string) (prev, next string) {
	return repo.adjacentSymbols(symbol, sort, order, "instruments.active = ? AND instruments.watchlist = ? AND instruments.is_nav = false", true, true)
}

func (repo *InstrumentRepo) UpdateInstrument(instrument *models.Instrument) error {
	return repo.Db.Save(instrument).Error
}

func (repo *InstrumentRepo) ToggleWatchlist(symbol string) (bool, error) {
	instrument, err := repo.GetInstrumentBySymbol(symbol)
	if err != nil {
		return false, err
	}
	newState := !instrument.Watchlist
	updates := map[string]interface{}{"watchlist": newState}
	if newState {
		updates["watch_list_date"] = time.Now()
	}
	err = repo.Db.Model(instrument).Updates(updates).Error
	return newState, err
}

func (repo *InstrumentRepo) GetWatchlistInstruments() ([]models.Instrument, error) {
	var instruments []models.Instrument
	err := repo.Db.
		Order(clause.OrderByColumn{Column: clause.Column{Name: "market_cap"}, Desc: true}).
		Where(&models.Instrument{IsNav: false, Watchlist: true}).
		Find(&instruments).Error
	return instruments, err
}

func (repo *InstrumentRepo) SearchWatchlistInstruments(query string) ([]models.Instrument, error) {
	var instruments []models.Instrument
	err := repo.Db.
		Where("active = ? AND watchlist = ? AND trading_symbol ILIKE ?", true, true, "%"+query+"%").
		Order("market_cap DESC").
		Find(&instruments).Error
	return instruments, err
}

func (repo *InstrumentRepo) UpdateHistoricalTrade(trade *models.Historicaldata) error {
	return repo.Db.Save(trade).Error
}

func (repo *InstrumentRepo) AdjustPriceByRatio(symbol string, beforeDate time.Time, ratio float64) error {
	return repo.Db.Model(&models.Historicaldata{}).
		Where("symbol = ? AND date < ?", symbol, beforeDate).
		Update("adjusted_close_price", gorm.Expr("adjusted_close_price / ?", ratio)).Error
}

func (repo *InstrumentRepo) GetFailedNseApi() (*[]models.Instrument, error) {
	var instruments []models.Instrument

	err := repo.Db.Where(&models.Instrument{NseApiSuccess: false}).Find(&instruments).Error
	if err != nil {
		return nil, err
	}
	return &instruments, nil
}

func (repo *InstrumentRepo) UpdateTag(symbol string, tag string) error {
	instrument, err := repo.GetInstrumentBySymbol(symbol)
	if err != nil {
		return err
	}
	if instrument.Tag == tag {
		return nil
	}
	return repo.Db.Transaction(func(tx *gorm.DB) error {
		history := models.TagHistory{
			InstrumentID:  instrument.ID,
			TradingSymbol: instrument.TradingSymbol,
			PreviousTag:   instrument.Tag,
			NextTag:       tag,
			UpdatedOn:     time.Now(),
		}
		if err := tx.Create(&history).Error; err != nil {
			return err
		}
		return tx.Model(instrument).Update("tag", tag).Error
	})
}

func (repo *InstrumentRepo) GetTagHistory(symbol string) ([]models.TagHistory, error) {
	var history []models.TagHistory
	err := repo.Db.Where("trading_symbol = ?", symbol).Order("updated_on ASC").Find(&history).Error
	return history, err
}

func (repo *InstrumentRepo) GetClosestAdjustedPrice(symbol string, date time.Time) (float64, error) {
	var data models.Historicaldata
	err := repo.Db.Where("symbol = ? AND date <= ? AND adjusted_close_price IS NOT NULL AND adjusted_close_price != 0", symbol, date).
		Order("date DESC").First(&data).Error
	if err != nil {
		return 0, err
	}
	return data.AdjustedClosePrice, nil
}

func (repo *InstrumentRepo) GetLatestAdjustedClose(symbol string) (float64, error) {
	var data models.Historicaldata
	err := repo.Db.Where("symbol = ? AND adjusted_close_price > 0", symbol).
		Order("date DESC").First(&data).Error
	if err != nil {
		return 0, err
	}
	return data.AdjustedClosePrice, nil
}

func (repo *InstrumentRepo) RenameSymbol(oldSymbol, newSymbol string) error {
	return repo.Db.Transaction(func(tx *gorm.DB) error {
    tables := []struct{ table, column string }{
      {"instruments", "trading_symbol"},
      {"historicaldata", "symbol"},
      {"shorts", "trading_symbol"},
      {"paper_trades", "trading_symbol"},
      {"paper_trade_logs", "trading_symbol"},
      {"gtt_orders", "trading_symbol"},
      {"tag_histories", "trading_symbol"},
      {"events", "trading_symbol"},
      {"notes", "trading_symbol"},
      {"bulk_block_deals", "trading_symbol"},
      {"insider_trade_transactions", "symbol"},
      {"kite_alerts", "trading_symbol"},
      {"alert_logs", "trading_symbol"},
    }
		for _, t := range tables {
			if err := tx.Exec("UPDATE "+t.table+" SET "+t.column+" = ? WHERE "+t.column+" = ?", newSymbol, oldSymbol).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (repo *InstrumentRepo) GetLatestSignals() (map[string][3]float64, error) {
	type row struct {
		Symbol     string
		VptScore   float64
		Divergence float64
		VolumeMa20 float64
	}
	var rows []row
	err := repo.Db.Raw(`SELECT h.symbol, h.vpt_score, h.divergence, h.volume_ma20 FROM historicaldata h
		INNER JOIN (SELECT symbol, MAX(date) AS max_date FROM historicaldata GROUP BY symbol) m
		ON h.symbol = m.symbol AND h.date = m.max_date
		WHERE h.vpt_score != 0 OR h.divergence != 0`).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[string][3]float64, len(rows))
	for _, r := range rows {
		result[r.Symbol] = [3]float64{r.VptScore, r.Divergence, r.VolumeMa20}
	}
	return result, nil
}

func (repo *InstrumentRepo) GetNotesBySymbol(symbol string) ([]models.Note, error) {
	var notes []models.Note
	err := repo.Db.Where("trading_symbol = ?", symbol).Order("date ASC").Find(&notes).Error
	return notes, err
}

func (repo *InstrumentRepo) GetAllNotes() ([]models.Note, error) {
	var notes []models.Note
	err := repo.Db.Order("trading_symbol ASC, date DESC").Find(&notes).Error
	return notes, err
}

func (repo *InstrumentRepo) CreateNote(note *models.Note) error {
	return repo.Db.Create(note).Error
}

func (repo *InstrumentRepo) DeleteNote(id uint) error {
	return repo.Db.Delete(&models.Note{}, id).Error
}
