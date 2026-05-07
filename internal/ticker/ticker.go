package ticker

import (
	"context"
	"log"
	"ninja-trader/internal/kite"
	models "ninja-trader/internal/model"
	repo "ninja-trader/internal/repository"
	"sync"
	"sync/atomic"
	"time"

	kiteticker "github.com/zerodha/gokiteconnect/v4/ticker"
	kitemodels "github.com/zerodha/gokiteconnect/v4/models"
)

// TickData holds the latest tick for an instrument.
type TickData struct {
	LastPrice         float64
	Open              float64
	High              float64
	Low               float64
	Close             float64
	VolumeTraded      uint32
	TotalBuyQuantity  uint32
	TotalSellQuantity uint32
	AverageTradePrice float64
	NetChange         float64
	Timestamp         time.Time
}

// BreadthSnapshot records the breadth count at a point in time.
type BreadthSnapshot struct {
	Time  time.Time
	Count int64
}

// Service manages the Kite WebSocket ticker connection.
type Service struct {
	kiteClient *kite.Client
	repo       repo.InstrumentRepo

	tokens      []uint32
	instruments map[uint32]InstrumentInfo

	ticker *kiteticker.Ticker
	prices sync.Map // map[uint32]*atomic.Pointer[TickData]
	alerts []Alert

	mu     sync.Mutex
	cancel context.CancelFunc

	// Breadth tracking: count of instruments up 2%+ from day's low
	breadthFlags   sync.Map     // map[uint32]bool
	breadthCount   atomic.Int64
	breadthHistory []BreadthSnapshot
	breadthMu      sync.Mutex
}

// New creates a new ticker service.
func New(kiteClient *kite.Client, instrumentRepo repo.InstrumentRepo) *Service {
	return &Service{
		kiteClient: kiteClient,
		repo:       instrumentRepo,
	}
}

// Start loads active instruments, sets up alerts, and connects to the Kite WebSocket.
// It runs the ticker in a background goroutine and returns immediately.
func (s *Service) Start(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ticker != nil {
		log.Println("[ticker] already running")
		return
	}

	accessToken := s.kiteClient.GetAccessToken()
	if accessToken == "" {
		log.Println("[ticker] no access token available — not starting. Login via /auth/login first.")
		return
	}

	instruments, err := s.repo.GetAllInstruments()
	if err != nil {
		log.Printf("[ticker] failed to load instruments: %v", err)
		return
	}
	if len(instruments) == 0 {
		log.Println("[ticker] no active instruments found — not starting")
		return
	}

	s.tokens = make([]uint32, len(instruments))
	s.instruments = make(map[uint32]InstrumentInfo, len(instruments))

	signals, _ := s.repo.GetLatestSignals()

	for i, inst := range instruments {
		token := uint32(inst.InstrumentToken)
		s.tokens[i] = token
		info := InstrumentInfo{
			Symbol:   inst.TradingSymbol,
			Tag:      inst.Tag,
			Industry: inst.BasicIndustry,
			Index:    inst.Index,
		}
		if sig, ok := signals[inst.TradingSymbol]; ok {
			info.VptScore = sig[0]
			info.Divergence = sig[1]
		}
		s.instruments[token] = info
	}

	s.alerts = []Alert{
    NewPriceAndVolumeAlert(0.02, 1.0, s.instruments),
	}

	s.startWebSocket(ctx, accessToken)
	s.startBreadthRecorder(ctx)
	// log.Printf("[ticker] started with %d instruments", len(s.tokens))
}

func (s *Service) startWebSocket(ctx context.Context, accessToken string) {
	t := kiteticker.New(s.kiteClient.GetAPIKey(), accessToken)
	t.SetAutoReconnect(true)
	t.SetReconnectMaxRetries(50)
	t.SetReconnectMaxDelay(10 * time.Second)

	t.OnConnect(func() {
		log.Printf("[ticker] connected, subscribing to %d instruments", len(s.tokens))
		if err := t.Subscribe(s.tokens); err != nil {
			log.Printf("[ticker] subscribe error: %v", err)
			return
		}
		if err := t.SetMode(kiteticker.ModeQuote, s.tokens); err != nil {
			log.Printf("[ticker] set mode error: %v", err)
		}
	})

	t.OnTick(func(tick kitemodels.Tick) {
		s.handleTick(tick)
	})

	t.OnError(func(err error) {
		log.Printf("[ticker] error: %v", err)
	})

	t.OnClose(func(code int, reason string) {
		log.Printf("[ticker] closed: %d - %s", code, reason)
	})

	t.OnReconnect(func(attempt int, delay time.Duration) {
		log.Printf("[ticker] reconnecting attempt %d, delay %v", attempt, delay)
	})

	t.OnNoReconnect(func(attempt int) {
		log.Printf("[ticker] max reconnect attempts reached (%d), giving up", attempt)
	})

	s.ticker = t

	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	go func() {
		log.Println("[ticker] starting websocket connection...")
		t.ServeWithContext(ctx)
		log.Println("[ticker] websocket connection ended")
	}()
}

// Stop gracefully stops the ticker connection.
func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	if s.ticker != nil {
		s.ticker.Stop()
		s.ticker = nil
	}
	log.Println("[ticker] stopped")
}

// GetPrice returns the latest tick data for an instrument token.
func (s *Service) GetPrice(token uint32) *TickData {
	val, ok := s.prices.Load(token)
	if !ok {
		return nil
	}
	ptr := val.(*atomic.Pointer[TickData])
	return ptr.Load()
}

func (s *Service) handleTick(tick kitemodels.Tick) {
	td := &TickData{
		LastPrice:         tick.LastPrice,
		Open:              tick.OHLC.Open,
		High:              tick.OHLC.High,
		Low:               tick.OHLC.Low,
		Close:             tick.OHLC.Close,
		VolumeTraded:      tick.VolumeTraded,
		TotalBuyQuantity:  tick.TotalBuyQuantity,
		TotalSellQuantity: tick.TotalSellQuantity,
		AverageTradePrice: tick.AverageTradePrice,
		NetChange:         tick.NetChange,
		Timestamp:         tick.Timestamp.Time,
	}

	// Store atomically
	val, _ := s.prices.LoadOrStore(tick.InstrumentToken, &atomic.Pointer[TickData]{})
	ptr := val.(*atomic.Pointer[TickData])
	ptr.Store(td)

	// Breadth tracking
	isUp := td.Low > 0 && (td.LastPrice-td.Low)/td.Low >= 0.02
	prev, _ := s.breadthFlags.Load(tick.InstrumentToken)
	wasUp, _ := prev.(bool)
	if isUp && !wasUp {
		s.breadthFlags.Store(tick.InstrumentToken, true)
		s.breadthCount.Add(1)
	} else if !isUp && wasUp {
		s.breadthFlags.Store(tick.InstrumentToken, false)
		s.breadthCount.Add(-1)
	}

	// Run alerts
	for _, alert := range s.alerts {
		alert.Check(tick.InstrumentToken, td)
	}
}

// SymbolForToken returns the trading symbol for a given instrument token.
func (s *Service) SymbolForToken(token uint32) string {
	return s.instruments[token].Symbol
}

// compile-time check that Service uses the correct model
var _ = models.Instrument{}

func (s *Service) startBreadthRecorder(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				snapshot := BreadthSnapshot{
					Time:  time.Now(),
					Count: s.breadthCount.Load(),
				}
				s.breadthMu.Lock()
				s.breadthHistory = append(s.breadthHistory, snapshot)
				s.breadthMu.Unlock()
			}
		}
	}()
}

// GetBreadthCount returns the current number of instruments up 2%+ from day's low.
func (s *Service) GetBreadthCount() int64 {
	return s.breadthCount.Load()
}

// GetBreadthHistory returns the time-series of breadth snapshots.
func (s *Service) GetBreadthHistory() []BreadthSnapshot {
	s.breadthMu.Lock()
	defer s.breadthMu.Unlock()
	result := make([]BreadthSnapshot, len(s.breadthHistory))
	copy(result, s.breadthHistory)
	return result
}
