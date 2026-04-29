package service

import (
	"errors"
	"fmt"
	models "ninja-trader/internal/model"
	repo "ninja-trader/internal/repository"
	"time"

	"gorm.io/gorm"
)

type TradeService struct {
	TradeRepo  repo.TradeRepo
	InstruRepo repo.InstrumentRepo
}

func (s *TradeService) GetAllTradesWithPnL() ([]repo.TradeWithPnL, error) {
	return s.TradeRepo.GetAllTradesWithPnL()
}

func (s *TradeService) GetTradeLogsBySymbol(symbol string) ([]models.PaperTradeLog, error) {
	return s.TradeRepo.GetLogsBySymbol(symbol)
}

func (s *TradeService) Trade(symbol string, quantity uint, tradeType models.TradeType, price float64) error {
	instrument, err := s.InstruRepo.GetInstrumentBySymbol(symbol)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("no instrument found for symbol: %s", symbol)
	} else if err != nil {
		return err
	}
	if !instrument.Active {
		return fmt.Errorf("instrument %s is inactive", symbol)
	}

	instrumentHistory, err := s.InstruRepo.GetPreviousTrade(symbol, time.Now())
	tradeDate := instrumentHistory.Date
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("no trade history available for symbol: %s and date: %s", symbol, tradeDate.Format("2006-01-02"))
	} else if err != nil {
		return err
	}

	execPrice := instrumentHistory.C
	if price > 0 {
		execPrice = price
	}

	trade, err := s.TradeRepo.GetTradeBySymbol(symbol)
	if err != nil {
		trade = &models.PaperTrade{Instrument: *instrument, InstrumentID: instrument.ID, TradingSymbol: symbol}
	}

	parsedQty := int(quantity)
	if tradeType == models.Sell {
		parsedQty = -parsedQty
	}

	totalCost := float64(trade.Quantity)*trade.AveragePrice + float64(parsedQty)*execPrice
	trade.Quantity += parsedQty
	if trade.Quantity ==  0 {
		trade.AveragePrice = 0
	} else {
		trade.AveragePrice = totalCost / float64(trade.Quantity)

	}

	fmt.Printf("Trade price: %f\n", trade.AveragePrice)

	log := &models.PaperTradeLog{
		InstrumentID:  instrument.ID,
		Instrument:    *instrument,
		TradingSymbol: symbol,
		Quantity:      quantity,
		TradeType:     tradeType,
		Date:          tradeDate,
		ExecutedAt:    time.Now(),
		AvgPrice:      execPrice,
		Amount:        float64(quantity) * execPrice,
	}

	return s.TradeRepo.UpsertTrade(trade, log)
}
