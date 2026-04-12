package repo

import (
	models "ninja-trader/internal/model"
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
			DoUpdates: clause.AssignmentColumns([]string{"market_cap", "nse_api_success", "is_nav", "instrument_full_name"}),
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
}

func (repo *InstrumentRepo) withHoldingQuery() *gorm.DB {
	return repo.Db.Model(&models.Instrument{}).
		Select("instruments.*, COALESCE(pt.quantity, 0) AS holding, COALESCE(hd.delivery_percentage, 0) AS delivery_percentage").
		Joins("LEFT JOIN paper_trades pt ON pt.instrument_id = instruments.id").
		Joins("LEFT JOIN historicaldata hd ON hd.symbol = instruments.trading_symbol AND hd.date = (SELECT MAX(h2.date) FROM historicaldata h2 WHERE h2.symbol = instruments.trading_symbol)")
}

func (repo *InstrumentRepo) GetAllInstrumentsWithHolding(query string) ([]InstrumentWithHolding, error) {
	var results []InstrumentWithHolding
	q := repo.withHoldingQuery().Where("instruments.active = true")
	if query != "" {
		q = q.Where("instruments.trading_symbol ILIKE ?", "%"+query+"%")
	}
	err := q.Order("instruments.market_cap DESC").Scan(&results).Error
	return results, err
}

func (repo *InstrumentRepo) GetWatchlistInstrumentsWithHolding(query string) ([]InstrumentWithHolding, error) {
	var results []InstrumentWithHolding
	q := repo.withHoldingQuery().Where("instruments.active = true AND instruments.watchlist = true AND instruments.is_nav = false")
	if query != "" {
		q = q.Where("instruments.trading_symbol ILIKE ?", "%"+query+"%")
	}
	err := q.Order("instruments.market_cap DESC").Scan(&results).Error
	return results, err
}

func (repo *InstrumentRepo) CreateShorts(shorts []models.Shorts) error {
	return repo.Db.Create(&shorts).Error
}

func (repo *InstrumentRepo) DeleteShortsByDate(date time.Time) error {
	return repo.Db.Where("date = ?", date).Delete(&models.Shorts{}).Error
}

type ShortsByDate struct {
	Date     time.Time
	Quantity int64
}

func (repo *InstrumentRepo) GetShortsBySymbol(symbol string, since *time.Time) ([]ShortsByDate, error) {
	var results []ShortsByDate
	q := repo.Db.Model(&models.Shorts{}).
		Select("date, SUM(quantity) as quantity").
		Where("trading_symbol = ?", symbol)
	if since != nil {
		q = q.Where("date >= ?", *since)
	}
	err := q.Group("date").Order("date ASC").Scan(&results).Error
	return results, err
}

func (repo *InstrumentRepo) GetShortsBySymbolAndYear(symbol string, year string) ([]ShortsByDate, error) {
	var results []ShortsByDate
	err := repo.Db.Model(&models.Shorts{}).
		Select("date, SUM(quantity) as quantity").
		Where("trading_symbol = ? AND EXTRACT(YEAR FROM date) = ?", symbol, year).
		Group("date").Order("date ASC").Scan(&results).Error
	return results, err
}

type ShortSymbolByDate struct {
	Date          time.Time
	TradingSymbol string
	Quantity      int64
}

func (repo *InstrumentRepo) GetShortSymbolsGroupedByDate(since *time.Time) ([]ShortSymbolByDate, error) {
	var results []ShortSymbolByDate
	q := repo.Db.Model(&models.Shorts{}).
		Select("date, trading_symbol, SUM(quantity) as quantity")
	if since != nil {
		q = q.Where("date >= ?", *since)
	}
	err := q.Group("date, trading_symbol").Order("date DESC, quantity DESC").Scan(&results).Error
	return results, err
}

func (repo *InstrumentRepo) GetShortSymbolsGroupedByDateAndYear(year string) ([]ShortSymbolByDate, error) {
	var results []ShortSymbolByDate
	err := repo.Db.Model(&models.Shorts{}).
		Select("date, trading_symbol, SUM(quantity) as quantity").
		Where("EXTRACT(YEAR FROM date) = ?", year).
		Group("date, trading_symbol").Order("date DESC, quantity DESC").Scan(&results).Error
	return results, err
}

type TotalShortsByDate struct {
	Date        time.Time
	Quantity    int64
	SymbolCount int64
}

func (repo *InstrumentRepo) GetTotalShortsByDate(since *time.Time) ([]TotalShortsByDate, error) {
	var results []TotalShortsByDate
	q := repo.Db.Model(&models.Shorts{}).
		Select("date, SUM(quantity) as quantity, COUNT(DISTINCT trading_symbol) as symbol_count")
	if since != nil {
		q = q.Where("date >= ?", *since)
	}
	err := q.Group("date").Order("date ASC").Scan(&results).Error
	return results, err
}

func (repo *InstrumentRepo) GetTotalShortsByDateAndYear(year string) ([]TotalShortsByDate, error) {
	var results []TotalShortsByDate
	err := repo.Db.Model(&models.Shorts{}).
		Select("date, SUM(quantity) as quantity, COUNT(DISTINCT trading_symbol) as symbol_count").
		Where("EXTRACT(YEAR FROM date) = ?", year).
		Group("date").Order("date ASC").Scan(&results).Error
	return results, err
}

func (repo *InstrumentRepo) GetShortsYears() ([]int, error) {
	var years []int
	err := repo.Db.Model(&models.Shorts{}).
		Select("DISTINCT EXTRACT(YEAR FROM date)::int AS year").
		Order("year DESC").
		Pluck("year", &years).Error
	return years, err
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

func (repo *InstrumentRepo) GetAdjacentSymbols(symbol string) (prev, next string) {
	var current models.Instrument
	if err := repo.Db.Where("trading_symbol = ? AND active = ?", symbol, true).First(&current).Error; err != nil {
		return
	}
	var prevInst, nextInst models.Instrument
	if repo.Db.Where("active = ? AND market_cap > ?", true, current.MarketCap).Order("market_cap ASC").First(&prevInst).Error == nil {
		prev = prevInst.TradingSymbol
	}
	if repo.Db.Where("active = ? AND market_cap < ?", true, current.MarketCap).Order("market_cap DESC").First(&nextInst).Error == nil {
		next = nextInst.TradingSymbol
	}
	return
}

func (repo *InstrumentRepo) GetAdjacentWatchlistSymbols(symbol string) (prev, next string) {
	var current models.Instrument
	if err := repo.Db.Where("trading_symbol = ? AND active = ?", symbol, true).First(&current).Error; err != nil {
		return
	}
	var prevInst, nextInst models.Instrument
	if repo.Db.Where("active = ? AND watchlist = ? AND market_cap > ?", true, true, current.MarketCap).Order("market_cap ASC").First(&prevInst).Error == nil {
		prev = prevInst.TradingSymbol
	}
	if repo.Db.Where("active = ? AND watchlist = ? AND market_cap < ?", true, true, current.MarketCap).Order("market_cap DESC").First(&nextInst).Error == nil {
		next = nextInst.TradingSymbol
	}
	return
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
	err = repo.Db.Model(instrument).Update("watchlist", newState).Error
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
