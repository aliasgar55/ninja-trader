package service

import (
	"errors"
	"fmt"
	"log"
	"ninja-trader/internal/kite"
	models "ninja-trader/internal/model"
	"ninja-trader/internal/nse"
	repo "ninja-trader/internal/repository"
	tradingClient "ninja-trader/internal/trading_client"
	yahoo "ninja-trader/internal/yahoof"
	"os"
	"sync"
	"time"
)

var (
	oneCrore           float64 = 10000000
	marketCapThreshold float64 = 2000 * oneCrore
)

type InstrumentService struct {
	InstruRepo repo.InstrumentRepo
}

func (s *InstrumentService) SyncInstruments() error {
	apiKey := os.Getenv("KITE_API_KEY")
	client := kite.New(apiKey)
	instruments, err := client.GetAllInstruments()
	if err != nil {
		return fmt.Errorf("getting instruments: %w", err)
	}
	c := make(chan tradingClient.Instrument, 1000)
	var wg sync.WaitGroup
	for range 1 {
		wg.Go(func() {
			for instru := range c {
				s.ProcessInstrument(&instru)
			}
		})
	}
	for _, instru := range *instruments {
		c <- instru
	}
	close(c)
	wg.Wait()
	return nil
}

func (service *InstrumentService) ProcessInstrument(instru *tradingClient.Instrument) {

	if instru.IsTradingAllowed() {

		var active, isNav, nseApiSuccess bool
		var priceBand, nseError string
		var marketCap float64

		nseResp, err := nse.GetNseDetails(instru.Tradingsymbol, instru.InstrumentType)

		if err != nil {
			log.Printf("Error getting nse reponse, %v\n", err)
			nseError = err.Error()
		} else if len(nseResp.EquityResponse) == 0 {
			nseError = fmt.Sprintf("empty EquityResponse for %s", instru.Tradingsymbol)
		} else {
			active = true
			equityRes := nseResp.EquityResponse[0]
			marketCap = nseResp.GetMarketCap()
			priceBand = nseResp.GetPriceBand()
			nseApiSuccess = true
			if equityRes.PriceInfo.IsINav == "True" {
				isNav = true
				active = false
			} else if marketCap <= marketCapThreshold {
				active = false
			}
		}

		dbInstru := models.Instrument{
			InstrumentToken:    instru.InstrumentToken,
			ExchangeToken:      instru.ExchangeToken,
			InstrumentFullName: instru.Name,
			TradingSymbol:      instru.Tradingsymbol,
			Exchange:           instru.Exchange,
			MarketCap:          marketCap / oneCrore,
			Active:             active,
			IsNav:              isNav,
			PriceBand:          priceBand,
			NseApiSuccess:      nseApiSuccess,
			NseApiError:        nseError,
		}
		err = service.InstruRepo.CreateInstrument(&dbInstru)

		if err != nil {
			log.Fatalf("Error saving instrument to database: %s", err)
		}
	}
}

func (s *InstrumentService) SyncShorts(from, to time.Time) error {
	// TODO: implement short trades sync logic for date range

	log.Printf("SyncShorts called for range %s to %s", from.Format("2006-01-02"), to.Format("2006-01-02"))
	processWorkers := 2
	var processWg sync.WaitGroup
	processChan := make(chan []nse.ShortTrade, 100)

	for range processWorkers {
		processWg.Go(func() {
			for trades := range processChan {
				s.ProcessShortTrades(trades)
			}
		})
	}
	defer processWg.Wait()

	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		shortTrades, err := nse.GetShortTrades(d)
		if err != nil {
			log.Printf("Error processing short trades for date %v, %v\n", d, err)
		}
		processChan <- shortTrades
	}
	close(processChan)
	return nil
}

func (s *InstrumentService) ProcessShortTrades(trades []nse.ShortTrade) {
	if len(trades) > 0 {
		tradesDb := make([]models.Shorts, len(trades))

		if err := s.InstruRepo.DeleteShortsByDate(time.Time(trades[0].TradeDate)); err != nil {
			log.Printf("Error clearing short trades error: %v", err)
		}
		for i, trade := range trades {
			tradesDb[i] = *trade.MapToShortModel()
		}
		if err := s.InstruRepo.CreateShorts(tradesDb); err != nil {
			log.Printf("Error saving short trades error: %v", err)
		}
	}
}

func (s *InstrumentService) SyncTradeHistory(symbol string, from time.Time, to time.Time) {

	var processWg sync.WaitGroup
	processChan := make(chan nse.HistoricalTrade)
	workers := uint8(5)
	tradeData, err := nse.GetHistoricalData(symbol, "EQ", from, to)
	if err != nil {
		fmt.Printf("Error fetching nse data from symbol: %s, %v, %v, %v\n", symbol, from, to, err)

	}
	for range workers {
		processWg.Go(
			func() {
				tradesList := make([]models.Historicaldata, 50)
				for trade := range processChan {
					tradesList = append(tradesList, *trade.MapToDb())
				}
				s.ProcessTradeHistory(tradesList)
			})
	}
	for _, tradeData := range tradeData {
		processChan <- tradeData
	}
}

func (s *InstrumentService) SyncAdjClosePrice(symbol string, minDate, maxDate *time.Time) error {

	if minDate == nil || maxDate == nil {
		dateRange, err := s.InstruRepo.GetHistoricalDataDateRange(symbol)
		if err != nil {
			log.Printf("Error getting range for symbol %s\n", symbol)
			return err
		}
		fmt.Println(dateRange)
		minDate = &dateRange.MinDate
		maxDate = &dateRange.MaxDate
	}
	tradeData, err := yahoo.GetHistoricalData(symbol, *minDate, *maxDate)
	if err != nil {
		log.Printf("Error getting yahoo finance data, symbol: %s, startDate: %v, endDate: %v, error1: %v\n", symbol, minDate, maxDate, err)
		return err
	}
	if tradeData.Chart.Error != nil {
		log.Printf("Error getting yahoo finance data, symbol: %s, startDate: %v, endDate: %v, error2: %v\n", symbol, minDate, maxDate, tradeData.Chart.Error)
		return errors.New(tradeData.Chart.Error.Description)
	}
	timeStamps := tradeData.Chart.Result[0].Timestamp
	adjustedClosePrice := tradeData.Chart.Result[0].Indicators.AdjClose[0].AdjClose
	dbUpdateArray := make([]models.AdjustedCloseUpdate, len(timeStamps))
	for i, ts := range timeStamps {
		if adjustedClosePrice[i] == 0 {
		}
		update := models.AdjustedCloseUpdate{
			Date:               time.Unix(ts, 0),
			Symbol:             symbol,
			AdjustedClosePrice: adjustedClosePrice[i],
		}
		dbUpdateArray = append(dbUpdateArray, update)
	}
	err = s.InstruRepo.BulkUpdateAdjustedClosePrice(dbUpdateArray)
	if err != nil {
		log.Printf("Error Updating adjusted close price %s, error: %v", symbol, err)
	}
	missingAdjustedTrades, err := s.InstruRepo.GetTradesWithNullAdjustedClose(symbol)
	if err != nil {
		log.Printf("Error getting missing adjusted prices trades, %s, error: %v", symbol, err)
		return err
	}
	var fillMissingWg sync.WaitGroup
	for _, trade := range missingAdjustedTrades {
		fillMissingWg.Go(func() { s.AddMissingAdjustedPrice(&trade) })

	}
	fillMissingWg.Wait()
	return nil

}

func (s *InstrumentService) SyncSplitAndDividend(symbol string, minDate, maxDate *time.Time) error {

	instrument, err := s.InstruRepo.GetInstrumentBySymbol(symbol)
	if err != nil {
		log.Printf("Error getting instrument by symbol %s\n", symbol)
	}
	if minDate == nil || maxDate == nil {
		dateRange, err := s.InstruRepo.GetHistoricalDataDateRange(symbol)
		if err != nil {
			log.Printf("Error getting range for symbol %s\n", symbol)
			return err
		}
		fmt.Println(dateRange)
		minDate = &dateRange.MinDate
		maxDate = &dateRange.MaxDate
	}
	events, err := yahoo.GetHistoricalEvents(symbol, *minDate, *maxDate)
	if err != nil {
		log.Printf("Error getting yahoo finance data, symbol: %s, startDate: %v, endDate: %v, error1: %v\n", symbol, minDate, maxDate, err)
		return err
	}
	var dbList []models.Event
	for _, dividend := range events.Dividends {
		dbList = append(dbList, *dividend.MapToDb(instrument))
	}
	for _, split := range events.Splits {
		dbList = append(dbList, *split.MapToDb(instrument))
	}

	err = s.InstruRepo.BulkInsertEvents(dbList)
	if err != nil {
		fmt.Printf("Error inserting events to the database error: %v\n", err)
		return err
	}
	return nil

}

func (s *InstrumentService) ProcessTradeHistory(trades []models.Historicaldata) error {
	err := s.InstruRepo.BulkInsertTradeHistory(trades)
	if err != nil {
		fmt.Println("Error saving trade history to the database")
		return err
	}
	return nil

}

func (s *InstrumentService) AddMissingAdjustedPrice(trade *models.Historicaldata) {
	previousTrade, err := s.InstruRepo.GetPreviousTrade(trade.Symbol, trade.Date)
	if err != nil {
		log.Printf("Error getting previous trade, %s, %v, %v\n", trade.Symbol, trade.Date, err)
		return
	}
	priceDifference := previousTrade.C - previousTrade.AdjustedClosePrice
	trade.AdjustedClosePrice = previousTrade.C - priceDifference
	err = s.InstruRepo.UpdateHistoricalTrade(trade)
	if err != nil {
		log.Printf("Error updating historical trade %s\n", trade.Symbol)
		return
	}
	log.Printf("Added missing price %s, %v, price_diffrence: %f\n", trade.Symbol, trade.Date, priceDifference)
}

func (s *InstrumentService) SyncDailyData() {
	instruments, err := s.InstruRepo.GetAllInstruments()
	if err != nil {
		log.Printf("Error running daily sync error: %v\n", err)
	}
	for _, instrument := range instruments {
		fmt.Println(instrument)
		err := s.ProcessDailyData(instrument.TradingSymbol)
		if err != nil {
			instrument.DailySyncError = fmt.Sprint(err)
			instrument.IsDailySyncFailed = true
			s.InstruRepo.UpdateInstrument(&instrument)
		} else if instrument.IsDailySyncFailed { // REMOVE PREVIOUS RUN ERROR IF PRESENT
			instrument.IsDailySyncFailed = false
			s.InstruRepo.UpdateInstrument(&instrument)
		}
	}

}

func (s *InstrumentService) ProcessDailyData(symbol string) error {
	log.Printf("ProcessDailyData started for %s\n", symbol)

	dateRange, err := s.InstruRepo.GetHistoricalDataDateRange(symbol)
	if err != nil {
		log.Printf("Error syncing daily data for symbol, Get historical date range failed: %s\n", symbol)
		return err
	}
	var syncStartDate time.Time
	if dateRange.MinDate.IsZero() {
		syncStartDate = time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)
	} else {
		syncStartDate = dateRange.MaxDate.AddDate(0, 0, 1)
	}

	if syncStartDate.After(time.Now()) {
		syncStartDate = time.Now()
	}
	syncEndDate := time.Now()
	log.Printf("ProcessDailyData [%s] syncing trade history from %s to %s\n", symbol, syncStartDate.Format("2006-01-02"), syncEndDate.Format("2006-01-02"))

	historicalTrades, err := nse.GetHistoricalData(symbol, "EQ", syncStartDate, syncEndDate)
	if err != nil {
		log.Printf("ProcessDailyData [%s] error fetching historical trades: %v\n", symbol, err)
		return err
	} else if len(historicalTrades) == 0 {
		return nil // return if no new data if found on nse website
	}
	historicalTradesDb := make([]models.Historicaldata, len(historicalTrades))
	for i, trade := range historicalTrades {
		historicalTrade := *trade.MapToDb()
		historicalTrade.AdjustedClosePrice = trade.C
		historicalTradesDb[i] = historicalTrade
	}
	err = s.ProcessTradeHistory(historicalTradesDb)
	if err != nil {
		log.Printf("ProcessDailyData [%s] error insert to db failed: %v\n", symbol, err)
		return err
	}
	log.Printf("ProcessDailyData [%s] trade history sync complete\n", symbol)

	var processWg sync.WaitGroup
	errorChan := make(chan error, 5)
	processWg.Go(func() {
		log.Printf("ProcessDailyData [%s] updating instrument metadata\n", symbol)
		nseResp, err := nse.GetNseDetails(symbol, "EQ")
		if err != nil {
			log.Printf("Error syncing daily data for symbol nse api failed: %s\n", symbol)
			errorChan <- err
			return
		}
		instrument, err := s.InstruRepo.GetInstrumentBySymbol(symbol)
		if err != nil {
			log.Printf("Error fetching marketCap, and priceBand during daily sync, getInstrumentFailed: %s\n", symbol)
			errorChan <- err
			return
		}
		instrument.MarketCap = nseResp.GetMarketCap() / oneCrore
		instrument.PriceBand = nseResp.GetPriceBand()
		instrument.BasicIndustry = nseResp.GetSecInfo().BasicIndustry
		instrument.Index = nseResp.GetSecInfo().Index
		instrument.NeedsAdjsutment = nseResp.GetNeedsAdjustment()
		if instrument.NeedsAdjsutment {
			fmt.Println("%s needs adjusment")
		}
		s.InstruRepo.UpdateInstrument(instrument)
		log.Printf("ProcessDailyData [%s] instrument metadata updated, marketCap: %.2f, active: %v\n", symbol, instrument.MarketCap, instrument.Active)
		historicalTrade, err := s.InstruRepo.GetHistoricalDataBySymbolAndDate(symbol, nseResp.GetLastUpdateTime().Truncate(24*time.Hour))
		if err == nil {
			historicalTrade.DeliveryPercentage = nseResp.GetDeliveryToTradePer()
			s.InstruRepo.UpdateHistoricalTrade(historicalTrade)
			log.Printf("ProcessDailyData [%s] delivery percentage updated: %.2f\n", symbol, historicalTrade.DeliveryPercentage)
		} else {
			log.Printf("ProcessDailyData [%s] save insert to db failed: %v\n", symbol, err)
			errorChan <- err
			return
		}
	})

	processWg.Wait()
	close(errorChan)
	for err := range errorChan {
		if err != nil {
			fmt.Println(err)
			return err
		}
	}
	log.Printf("ProcessDailyData completed for %s\n", symbol)
	return nil
}

func (s *InstrumentService) AdjustPriceByEvents(symbol string) error {
	// events, err := s.InstruRepo.GetEventsBySymbol(symbol); err
	return nil

}
