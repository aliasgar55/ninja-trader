package service

import (
	"errors"
	"fmt"
	"log"
	"math"
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

	log.Printf("SyncShorts called for range %s to %s\n", from.Format("2006-01-02"), to.Format("2006-01-02"))
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
    log.Printf("Error fetching nse data from symbol: %s, %v, %v, %v\n", symbol, from, to, err)

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
		log.Println(dateRange)
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
    log.Printf("Error inserting events to the database error: %v\n", err)
		return err
	}
	return nil

}

func (s *InstrumentService) ProcessTradeHistory(trades []models.Historicaldata) error {
	err := s.InstruRepo.BulkInsertTradeHistory(trades)
	if err != nil {
    log.Println("Error saving trade history to the database")
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
		// add 1 so the max date to get the next day data
		syncStartDate = dateRange.MaxDate.AddDate(0, 0, 1)
		if syncStartDate.After(time.Now()) {
			syncStartDate = time.Now()
		}

	}

	syncEndDate := time.Now()
	log.Printf("ProcessDailyData [%s] syncing trade history from %s to %s\n", symbol, syncStartDate.Format("2006-01-02"), syncEndDate.Format("2006-01-02"))
	metaData,err := nse.GetMetaData(symbol)
	series, err := metaData.GetActiveSeries()
	if err != nil {
		return fmt.Errorf("Error syncing symbol: %s, due to series not found, err: %s\n", symbol, err)
	}

	historicalTrades, err := nse.GetHistoricalData(symbol, series, syncStartDate, syncEndDate)
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

	log.Printf("ProcessDailyData [%s] updating instrument metadata\n", symbol)
	nseResp, err := nse.GetNseDetails(symbol, series)
	if err != nil {
		log.Printf("Error syncing daily data for symbol nse api failed: %s\n", symbol)
		return err
	}
	instrument, err := s.InstruRepo.GetInstrumentBySymbol(symbol)
	if err != nil {
		log.Printf("Error fetching marketCap, and priceBand during daily sync, getInstrumentFailed: %s\n", symbol)
		return err
	}
	instrument.MarketCap = nseResp.GetMarketCap() / oneCrore
	instrument.PriceBand = nseResp.GetPriceBand()
	instrument.BasicIndustry = nseResp.GetSecInfo().BasicIndustry
	instrument.Index = nseResp.GetSecInfo().Index
	instrument.NeedsAdjsutment = nseResp.GetNeedsAdjustment()
	if metaData.IsFNOSec == "true" {
		instrument.IsFnoSec = true
	}
	if instrument.NeedsAdjsutment {
		// todo: implement auto adjustment
	}
	s.InstruRepo.UpdateInstrument(instrument)
	log.Printf("ProcessDailyData [%s] instrument metadata updated, marketCap: %.2f, active: %v\n", symbol, instrument.MarketCap, instrument.Active)
	historicalTrade, err := s.InstruRepo.GetHistoricalDataBySymbolAndDate(symbol, nseResp.GetLastUpdateTime().Truncate(24*time.Hour))
	if err == nil {
		historicalTrade.DeliveryPercentage = nseResp.GetDeliveryToTradePer()
		if instrument.MarketCap > 0 {
			historicalTrade.DeliveryValue = float64(historicalTrade.Volume) * historicalTrade.C * float64(historicalTrade.DeliveryPercentage) / 100.0 / (instrument.MarketCap * oneCrore) * 100
		}
		log.Printf("ProcessDailyData [%s] delivery percentage updated: %.2f, delivery value: %.4f%%\n", symbol, historicalTrade.DeliveryPercentage, historicalTrade.DeliveryValue)
	}
	s.InstruRepo.UpdateHistoricalTrade(historicalTrade)

	log.Printf("ProcessDailyData completed for %s\n", symbol)

	if err := s.ComputeSignals(symbol); err != nil {
    log.Println("Calling compute signal")
		log.Printf("ProcessDailyData [%s] error computing signals: %v\n", symbol, err)
	}

	return nil
}

func (s *InstrumentService) AdjustPriceByEvents(symbol string) error {
	// events, err := s.InstruRepo.GetEventsBySymbol(symbol); err
	return nil
}

func (s *InstrumentService) RenameSymbol(oldSymbol, newSymbol string) error {
	return s.InstruRepo.RenameSymbol(oldSymbol, newSymbol)
}

func (s *InstrumentService) ApplySymbolChanges() (int, error) {
	changes, err := nse.GetSymbolChanges()
	if err != nil {
		return 0, fmt.Errorf("fetching symbol changes: %w", err)
	}
	count := 0
	for _, c := range changes {
		if err := s.InstruRepo.RenameSymbol(c.OldSymbol, c.NewSymbol); err != nil {
			log.Printf("Failed to rename %s -> %s: %v", c.OldSymbol, c.NewSymbol, err)
			continue
		}
		log.Printf("Renamed symbol %s -> %s", c.OldSymbol, c.NewSymbol)
		count++
	}
	return count, nil
}

const rollingWindowDays = 1095 // 3 years in calendar days

// ComputeSignals computes VPT MA20, z-scores, sigmoid VPT score, divergence,
// and 3-year rolling max divergence for a symbol's historical data, then saves to DB.
func (s *InstrumentService) ComputeSignals(symbol string) error {
	data, err := s.InstruRepo.GetHistoricalDataBySymbol(symbol)
	if err != nil {
		return fmt.Errorf("getting historical data for %s: %w", symbol, err)
	}
	if len(data) < 21 {
		return nil // not enough data for MA20
	}

	// Compute VPT MA20 (simple moving average of VolumePerTrade over 20 days)
	vptMa20 := make([]float64, len(data))
	volumeMa20 := make([]float64, len(data))
	for i := range data {
		if i < 19 {
			continue
		}
		var sum float64
		var volumeSum uint64
		for j := i - 19; j <= i; j++ {
			sum += float64(data[j].VolumePerTrade)
			volumeSum += data[j].Volume
		}
		vptMa20[i] = sum / 20.0
		volumeMa20[i] = float64(volumeSum) / 20.0
	}

	// Compute rolling z-scores using 3-year calendar window
	priceZ := make([]float64, len(data))
	vptZ := make([]float64, len(data))
	for i := range data {
		if i < 19 { // need at least MA20 to be valid
			continue
		}
		windowStart := data[i].Date.AddDate(0, 0, -rollingWindowDays)

		// Find start index for the rolling window
		startIdx := i
		for startIdx > 0 && data[startIdx-1].Date.After(windowStart) {
			startIdx--
		}

		// Compute price z-score over window
		priceZ[i] = zScore(data, startIdx, i, func(d models.Historicaldata) float64 {
			return d.AdjustedClosePrice
		})

    // Compute VPT MA20 z-score over window (only valid entries where i >= 19)
    validStart := startIdx
    if validStart < 19 {
      validStart = 19
    }
    vptZ[i] = zScoreSlice(vptMa20, validStart, i)
	}

  // Compute VPT score as percentage of rolling 3-year max VPT MA20, and divergence
  vptScore := make([]float64, len(data))
  divergence := make([]float64, len(data))
  for i := range data {
    if i < 19 {
      continue
    }
    // Find rolling window start
    windowStart := data[i].Date.AddDate(0, 0, -rollingWindowDays)
    startIdx := i
    for startIdx > 0 && data[startIdx-1].Date.After(windowStart) {
      startIdx--
    }
    validStart := startIdx
    if validStart < 19 {
      validStart = 19
    }
    // Find min and max VPT MA20 in the window
    minVpt := math.Inf(1)
    maxVpt := math.Inf(-1)
    for j := validStart; j <= i; j++ {
      if vptMa20[j] < minVpt {
        minVpt = vptMa20[j]
      }
      if vptMa20[j] > maxVpt {
        maxVpt = vptMa20[j]
      }
    }
    if maxVpt > minVpt {
      vptScore[i] = (vptMa20[i] - minVpt) / (maxVpt - minVpt) * 100.0
    }
    divergence[i] = vptZ[i] - priceZ[i]
  }

	// Compute 3-year rolling max divergence
	divMax3y := make([]float64, len(data))
	for i := range data {
		if i < 19 {
			continue
		}
		windowStart := data[i].Date.AddDate(0, 0, -rollingWindowDays)
		maxDiv := math.Inf(-1)
		for j := i; j >= 19; j-- {
			if data[j].Date.Before(windowStart) {
				break
			}
			if divergence[j] > maxDiv {
				maxDiv = divergence[j]
			}
		}
		if math.IsInf(maxDiv, -1) {
			maxDiv = 0
		}
		divMax3y[i] = maxDiv
	}

	// Build update batch (only rows where MA20 is valid)
	updates := make([]models.SignalUpdate, 0, len(data)-19)
	for i := 19; i < len(data); i++ {
		updates = append(updates, models.SignalUpdate{
			ID:              data[i].ID,
			VptMa20:         vptMa20[i],
			VptScore:        vptScore[i],
			Divergence:      divergence[i],
			DivergenceMax3y: divMax3y[i],
			VolumeMa20: volumeMa20[i],
		})
	}

	if err := s.InstruRepo.BulkUpdateSignals(updates); err != nil {
		return fmt.Errorf("updating signals for %s: %w", symbol, err)
	}
	log.Printf("ComputeSignals [%s] updated %d rows\n", symbol, len(updates))
	return nil
}

// zScore computes the z-score of the value at index end, using data[start..end] extracted by fn.
// Uses sample standard deviation (ddof=1) to match pandas behavior.
func zScore(data []models.Historicaldata, start, end int, fn func(models.Historicaldata) float64) float64 {
	n := end - start + 1
	if n < 2 {
		return 0
	}
	var sum float64
	for i := start; i <= end; i++ {
		sum += fn(data[i])
	}
	mean := sum / float64(n)
	var sumSqDiff float64
	for i := start; i <= end; i++ {
		d := fn(data[i]) - mean
		sumSqDiff += d * d
	}
	variance := sumSqDiff / float64(n-1) // sample variance (ddof=1)
	if variance <= 0 {
		return 0
	}
	return (fn(data[end]) - mean) / math.Sqrt(variance)
}

// zScoreSlice computes the z-score of slice[end] using slice[start..end].
// Uses sample standard deviation (ddof=1) to match pandas behavior.
func zScoreSlice(slice []float64, start, end int) float64 {
	n := end - start + 1
	if n < 2 {
		return 0
	}
	var sum float64
	for i := start; i <= end; i++ {
		sum += slice[i]
	}
	mean := sum / float64(n)
	var sumSqDiff float64
	for i := start; i <= end; i++ {
		d := slice[i] - mean
		sumSqDiff += d * d
	}
	variance := sumSqDiff / float64(n-1) // sample variance (ddof=1)
	if variance <= 0 {
		return 0
	}
  return (slice[end] - mean) / math.Sqrt(variance)
}
