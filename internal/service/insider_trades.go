package service

import (
	"fmt"
	"log"
	models "ninja-trader/internal/model"
	"ninja-trader/internal/nse"
	repo "ninja-trader/internal/repository"
	"sync"
	"time"
)

type InsiderTradesService struct {
	InsiderTradeRepo repo.InsiderTradesRepo
}

func (s *InsiderTradesService) StartInsiderTradeSync(from, to time.Time) {
	log.Printf("Starting insider trade sync\n")
	defer fmt.Println("Completed processing insider trade sync")

	var detailsWg sync.WaitGroup
	defer detailsWg.Wait()
	var detailsChan = make(chan nse.InsideTrade)
	detailsWorkers := 10

	for range detailsWorkers {
		detailsWg.Go(func() {
			for insideTrade := range detailsChan {
				insideTradeTxn, insideTradeEntity, err := insideTrade.MapToDb()
				if err != nil {
					log.Fatalf("Error mapping nse inside trade to db model: err: %v\n", err)
				}
				err = s.InsiderTradeRepo.CreateInsiderTrade(insideTradeTxn, &insideTradeEntity)
				if err != nil  {
					log.Printf("Error saving insider trade to db, FilingId: %d\n, transactionDate: %v", insideTradeTxn.FilingID, insideTradeTxn.TransactionDate)
				}

			}
		})
	}


	for d := from; !d.After(to); d = d.AddDate(1, 0, 0) {
		toCurr := d.AddDate(1, 0, 0)
		if toCurr.After(to) {
			toCurr = to
		}
		log.Printf("Getting insider trades from: %v, to: %v\n", from, toCurr)
		insiderTrades, err := nse.GetInsideTrades(d, toCurr)
		log.Printf("Fetched trades: %d\n", len(insiderTrades))
		if err != nil {
			log.Printf("Error processing insider trades for date %s, %v\n", d, err)
			return
		}
		for _, trade := range insiderTrades {
			detailsChan <- trade
		}
	}
	log.Printf("Completed fetching all the insiderTrades from nse\n")
	close(detailsChan)

}

func (s *InsiderTradesService) GetTradesBySymbol(symbol string) ([]models.InsiderTradeWithEntity, error) {
	return s.InsiderTradeRepo.GetBySymbol(symbol)
}

func (s *InsiderTradesService) GetAllTrades(limit, offset int) ([]models.InsiderTradeWithEntity, int64, error) {
	return s.InsiderTradeRepo.GetAll(limit, offset)
}

