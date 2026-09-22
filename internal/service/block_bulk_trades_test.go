package service

import (
	models "ninja-trader/internal/model"
	"testing"
	"time"
)

type mockBBRepo struct {
	deals []models.BulkBlockDeal
	err   error
}

func (m *mockBBRepo) Create(deal *models.BulkBlockDeal) error { return nil }
func (m *mockBBRepo) GetLastDealDate(dealType string) (time.Time, error) {
	return time.Time{}, nil
}
func (m *mockBBRepo) GetBySymbolAndDate(symbol string, date time.Time) ([]models.BulkBlockDeal, error) {
	return m.deals, m.err
}

var testDate = time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)

func deal(client string, buySell models.TradeType, qty int64, price float64, dealType models.DealType) models.BulkBlockDeal {
	return models.BulkBlockDeal{
		TradingSymbol: "JSL",
		ClientName:    client,
		BuySell:       buySell,
		Quantity:      qty,
		Price:         price,
		DealType:      dealType,
		Date:          testDate,
	}
}

func TestIntraDayVolume_NoDeals(t *testing.T) {
	svc := NewBBService(&mockBBRepo{deals: nil})
	vol, err := svc.GetIntraDayVolumeBySymbol("JSL", testDate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vol != 0 {
		t.Errorf("expected 0, got %d", vol)
	}
}

func TestIntraDayVolume_OnlyBuys_NoMatching(t *testing.T) {
	// Client A buys 100, no sells — intraday vol should be 0
	svc := NewBBService(&mockBBRepo{deals: []models.BulkBlockDeal{
		deal("ClientA", models.Buy, 100, 50.0, models.DealTypeBulk),
	}})
	vol, err := svc.GetIntraDayVolumeBySymbol("JSL", testDate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vol != 0 {
		t.Errorf("expected 0, got %d", vol)
	}
}

func TestIntraDayVolume_SameClientBuyAndSell(t *testing.T) {
	// Client A buys 100, sells 80 — intraday = min(100,80) + min(80,100) = 80+80 = 160
	svc := NewBBService(&mockBBRepo{deals: []models.BulkBlockDeal{
		deal("ClientA", models.Buy, 100, 50.0, models.DealTypeBulk),
		deal("ClientA", models.Sell, 80, 50.0, models.DealTypeBulk),
	}})
	vol, err := svc.GetIntraDayVolumeBySymbol("JSL", testDate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// min(100,80) from Buy side + min(80,100) from Sell side = 160
	if vol != 160 {
		t.Errorf("expected 160, got %d", vol)
	}
}

func TestIntraDayVolume_TwoClients(t *testing.T) {
	// ClientA buys 200, sells 150
	// ClientB buys 100, sells 300
	svc := NewBBService(&mockBBRepo{deals: []models.BulkBlockDeal{
		deal("ClientA", models.Buy, 200, 50.0, models.DealTypeBulk),
		deal("ClientA", models.Sell, 150, 50.0, models.DealTypeBulk),
		deal("ClientB", models.Buy, 100, 50.0, models.DealTypeBulk),
		deal("ClientB", models.Sell, 300, 50.0, models.DealTypeBulk),
	}})
	vol, err := svc.GetIntraDayVolumeBySymbol("JSL", testDate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// ClientA: min(200,150) + min(150,200) = 150+150 = 300
	// ClientB: min(100,300) + min(300,100) = 100+100 = 200
	// Total = 500
	if vol != 500 {
		t.Errorf("expected 500, got %d", vol)
	}
}

func TestIntraDayVolume_DeduplicateBlockAndBulk(t *testing.T) {
	// Same deal appears in both block and bulk — bulk copy should be zeroed out
	svc := NewBBService(&mockBBRepo{deals: []models.BulkBlockDeal{
		deal("ClientA", models.Buy, 100, 50.0, models.DealTypeBlock),
		deal("ClientA", models.Buy, 100, 50.0, models.DealTypeBulk), // duplicate, should be zeroed
		deal("ClientA", models.Sell, 100, 50.0, models.DealTypeBlock),
	}})
	vol, err := svc.GetIntraDayVolumeBySymbol("JSL", testDate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// After dedup: ClientA buys 100 (block only), sells 100
	// min(100,100) + min(100,100) = 200
	if vol != 200 {
		t.Errorf("expected 200, got %d", vol)
	}
}

func TestIntraDayVolume_BulkOnlyNoDuplicates(t *testing.T) {
	// All bulk deals, no block overlap
	svc := NewBBService(&mockBBRepo{deals: []models.BulkBlockDeal{
		deal("ClientA", models.Buy, 500, 40.0, models.DealTypeBulk),
		deal("ClientA", models.Sell, 200, 40.0, models.DealTypeBulk),
	}})
	vol, err := svc.GetIntraDayVolumeBySymbol("JSL", testDate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// min(500,200) + min(200,500) = 200+200 = 400
	if vol != 400 {
		t.Errorf("expected 400, got %d", vol)
	}
}

func TestIntraDayVolume_MultipleDealsAggregatedPerClient(t *testing.T) {
	// ClientA has two buy deals and one sell
	svc := NewBBService(&mockBBRepo{deals: []models.BulkBlockDeal{
		deal("ClientA", models.Buy, 100, 50.0, models.DealTypeBulk),
		deal("ClientA", models.Buy, 200, 55.0, models.DealTypeBulk),
		deal("ClientA", models.Sell, 250, 52.0, models.DealTypeBulk),
	}})
	vol, err := svc.GetIntraDayVolumeBySymbol("JSL", testDate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Aggregated: Buy=300, Sell=250
	// min(300,250) + min(250,300) = 250+250 = 500
	if vol != 500 {
		t.Errorf("expected 500, got %d", vol)
	}
}
