package ticker

import (
	"fmt"
	"log"
	"math"
	"sync"
)

// InstrumentInfo holds DB metadata for an instrument, used in alert logging.
type InstrumentInfo struct {
	Symbol   string
	Tag      string
	Industry string
	Index    string
}

// Alert is the interface for tick-based alert checks.
type Alert interface {
	Check(token uint32, tick *TickData)
}

type PriceAndVolumeAlert struct {
	BuySellRatioThreshold float64 // e.g. 2.0 means buy qty > 2x sell qty
	PriceThreshold        float64 // e.g. 0.03 for 3%
	MoveThreshold         float64 // e.g. 0.01 for 1% move to re-alert
	Instruments           map[uint32]InstrumentInfo

	mu          sync.Mutex
	lastAlerted map[uint32]float64 // token -> price at last alert
}

// NewPriceAndVolumeAlert creates an alert that fires when price is near the day's low
// AND buy/sell ratio exceeds the threshold. Re-alerts on 1% moves from last alert price.
func NewPriceAndVolumeAlert(priceThreshold, ratioThreshold float64, instruments map[uint32]InstrumentInfo) *PriceAndVolumeAlert {
	return &PriceAndVolumeAlert{
		PriceThreshold:        priceThreshold,
		BuySellRatioThreshold: ratioThreshold,
		MoveThreshold:         0.01,
		Instruments:           instruments,
		lastAlerted:           make(map[uint32]float64),
	}
}

func (a *PriceAndVolumeAlert) Check(token uint32, tick *TickData) {
	if tick.TotalSellQuantity == 0 || tick.Low <= 0 || tick.LastPrice <= 0 {
		return
	}

	ratio := float64(tick.TotalBuyQuantity) / float64(tick.TotalSellQuantity)
	pctFromLow := (tick.LastPrice - tick.Low) / tick.Low

	if pctFromLow > a.PriceThreshold {
		a.mu.Lock()
		lastPrice, seen := a.lastAlerted[token]
		shouldAlert := !seen || math.Abs(tick.LastPrice-lastPrice)/lastPrice >= a.MoveThreshold
		if shouldAlert {
			a.lastAlerted[token] = tick.LastPrice
		}
		a.mu.Unlock()

		if shouldAlert {
			info := a.Instruments[token]
			sym := info.Symbol
			if sym == "" {
				sym = "unknown"
			}
			tag := info.Tag
			if tag == "" {
				tag = "-"
			}
			label := "NEW"
			if seen {
				pctMove := (tick.LastPrice - lastPrice) / lastPrice * 100
				if pctMove > 0 {
					label = fmt.Sprintf("+%.1f%%", pctMove)
				} else {
					label = fmt.Sprintf("%.1f%%", pctMove)
				}
			}
			log.Printf("[ALERT %s] %.1f%% from low | %s [%s] @ %.2f (avg: %.2f, low: %.2f) | buy/sell %.2f (buy: %d, sell: %d) | %s | %s",
				label, pctFromLow*100, sym, tag, tick.LastPrice, tick.AverageTradePrice, tick.Low, ratio, tick.TotalBuyQuantity, tick.TotalSellQuantity, info.Industry, info.Index)
		}
	}
}

// ResetAlerts clears the alerted state.
func (a *PriceAndVolumeAlert) ResetAlerts() {
	a.mu.Lock()
	a.lastAlerted = make(map[uint32]float64)
	a.mu.Unlock()
}
