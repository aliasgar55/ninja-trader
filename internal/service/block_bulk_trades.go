package service

import (
	"log"
	models "ninja-trader/internal/model"
	"ninja-trader/internal/nse"
	repo "ninja-trader/internal/repository"
	"sync"
	"time"
)

type BBService struct {
	repo repo.BulkBlockDealRepo
}

func NewBBService(db repo.BulkBlockDealRepo) *BBService {
	return &BBService{repo: db}
}

func (s *BBService) StartBulkBlockDealSyncWithoutDate() {
	defaultDate := time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)
	for _, dealType := range []models.DealType{models.DealTypeBlock, models.DealTypeBulk} {
		lastDate, err := s.repo.GetLastDealDate(string(dealType))
		if err != nil {
			log.Printf("Error fetching last %s deal sync date: %v", dealType, err)
			return
		}
		if lastDate.IsZero() {
			lastDate = defaultDate
		} else {
			lastDate = lastDate.AddDate(0, 0, 1) // Start from the next day
		}
		s.startBulkBlockDealSync(lastDate, time.Now(), dealType)
	}

}

func (s *BBService) startBulkBlockDealSync(from, to time.Time, dealType models.DealType) {
	log.Printf("Starting %s trade sync\n", string(dealType))
	defer log.Printf("Completed processing %s deals\n", string(dealType))

	var detailsWg sync.WaitGroup
	defer detailsWg.Wait()
	var dealsChan = make(chan nse.BulkBlockDeals)
	detailsWorkers := 10

	for range detailsWorkers {
		detailsWg.Go(func() {
			for deal := range dealsChan {
				dbModel, err := deal.MapToDb()
				if err != nil {
					log.Fatalf("Error mapping nse bulkbulk deal to db model: err: %v\n", err)
				}
				err = s.repo.Create(&dbModel)
				if err != nil {
					log.Printf("Error saving bbdeal model to db, err:%v", err)
				}

			}
		})
	}

	log.Printf("Getting %s trades from: %s, to: %s\n", string(dealType), from, to)
	blockDeals, err := nse.GetBulkBlockDeals(from, to, dealType)
	if err != nil {
		log.Printf("Error processing %s trades for date %s, %s, error: %v\n", string(dealType), from, to, err)
		return
	}
	log.Printf("Fetched %s deal: %d\n", string(dealType), len(blockDeals))
	for _, deal := range blockDeals {
		dealsChan <- deal
	}
	close(dealsChan)
	log.Printf("Completed fetching all the %s deals from nse\n", string(dealType))

}
