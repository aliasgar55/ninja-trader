package service

import (
  "log"
  models "ninja-trader/internal/model"
  "ninja-trader/internal/nse"
  "sync"
  "time"
)

type BulkBlockDealRepository interface {
  Create(deal *models.BulkBlockDeal) error
  GetBySymbolAndDate(symbol string, date time.Time) ([]models.BulkBlockDeal, error)
  GetLastDealDate(dealType string) (time.Time, error)
}

type BBService struct {
  repo BulkBlockDealRepository
}

func NewBBService(repo BulkBlockDealRepository) *BBService {
  return &BBService{repo: repo}
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


type DealKey struct {
	ClientName string
	Quantity   int64
	Price      float64
	BuySell    models.TradeType

}

type VolKey struct {
	ClientName string
	BuySell    models.TradeType

}
func (s *BBService) GetIntraDayVolumeBySymbol(symbol string, date time.Time) (uint64, error) {
	// 1. There can be duplication of deals between bulk an block
	// sanity checks
	deals, err := s.repo.GetBySymbolAndDate(symbol, date)
	if err != nil {
		log.Printf("Error fetching deals for symbol: %s, date: %v, error: %v", symbol, date, err)
		return 0, err
	}
	// if there are not deals return vol as 0
	if len(deals) == 0 {
		return 0, nil
	}
	blockDeals := make([]models.BulkBlockDeal,0, len(deals))
	blockMap := make(map[DealKey]models.BulkBlockDeal) // used for deduplication
	bulkDeals := make([]models.BulkBlockDeal, 0, len(deals))

	// segregate into bulk and block deals
	for _, deal := range deals {
		if deal.DealType ==  models.DealTypeBlock {
			blockDeals = append(blockDeals, deal)
		} else {
			bulkDeals = append(bulkDeals, deal)
		}
	}
	// deduplicate the deals
	for _, block := range blockDeals { 
		key := DealKey{
			ClientName: block.ClientName,
			Quantity: block.Quantity,
			Price: block.Price,
			BuySell: block.BuySell,
		}
		blockMap[key] = block
	}
	for i, bulk := range bulkDeals { 
		key := DealKey{
			ClientName: bulk.ClientName,
			Quantity: bulk.Quantity,
			Price: bulk.Price,
			BuySell: bulk.BuySell,
		}
		if _, ok := blockMap[key]; ok {
			// making the quantity 0 will have no effect on the intraday volume
			// effectively removing the deal from the list
			bulkDeals[i].Quantity = 0
		}
	}
	allDeals := append(bulkDeals, blockDeals...)
	// calculated the volume
	buySellVolPerClient := make(map[VolKey]uint64)

	for _, deal := range allDeals {
		key := VolKey{
			ClientName: deal.ClientName,
			BuySell: deal.BuySell,
		}
		buySellVolPerClient[key] += uint64(deal.Quantity)
	}

	intraDayVol := uint64(0)
	for k, v := range buySellVolPerClient {
		tradeType := k.BuySell
		var oppTradeType models.TradeType
		if tradeType == models.Buy {
			oppTradeType = models.Sell
		} else {
			oppTradeType = models.Buy
		}
		oppKey := VolKey{
			ClientName: k.ClientName,
			BuySell: oppTradeType,
			
		}
		intraDayVol += min(v, buySellVolPerClient[oppKey])
	}
	return intraDayVol, nil
}

