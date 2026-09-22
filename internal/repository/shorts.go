package repo

import (
	models "ninja-trader/internal/model"
	"time"
)

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

