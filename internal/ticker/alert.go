package ticker

import (
	"fmt"
	"log"
	"math"
	"os/exec"
	"sync"
	"time"
)

// InstrumentInfo holds DB metadata for an instrument, used in alert logging.
type InstrumentInfo struct {
	Symbol     string
	Tag        string
	Industry   string
	Index      string
	VptScore   float64
	Volume20Ma float64
	Divergence float64
}
    const (
        colorGreen = "\033[32m"
        colorBlue  = "\033[34m"
        colorReset = "\033[0m"
    )


// Alert is the interface for tick-based alert checks.
type Alert interface {
	Check(token uint32, tick *TickData) *AlertEntry
}

type PriceAlert struct {
	BuySellRatioThreshold float64 // e.g. 2.0 means buy qty > 2x sell qty
	PriceThreshold        float64 // e.g. 0.03 for 3%
	MoveThreshold         float64 // e.g. 0.01 for 1% move to re-alert
	Instruments           map[uint32]InstrumentInfo

	mu          sync.Mutex
	lastAlerted map[uint32]float64 // token -> price at last alert
}

type VolumeAlert struct {
	VolumeThreshold    float64 // e.g. 2 for 2x volume traded compare to 20ma
	SubsequentIncrease float64 // e.g. 0.2 for increase in volume by 0.2
	Instruments        map[uint32]InstrumentInfo

	mu          sync.Mutex
	lastAlerted map[uint32]float64 // token -> price at last alert
}

// NewPriceAlert creates an alert that fires when price is near the day's low
// AND buy/sell ratio exceeds the threshold. Re-alerts on 1% moves from last alert price.
func NewPriceAlert(priceThreshold, ratioThreshold float64, instruments map[uint32]InstrumentInfo) *PriceAlert {
	return &PriceAlert{
		PriceThreshold:        priceThreshold,
		BuySellRatioThreshold: ratioThreshold,
		MoveThreshold:         0.01,
		Instruments:           instruments,
		lastAlerted:           make(map[uint32]float64),
	}
}

func NewVolumeAlert(volumeThresold, subsequentIncrease float64, instruments map[uint32]InstrumentInfo) *VolumeAlert {
	return &VolumeAlert{
		VolumeThreshold:    volumeThresold,
		SubsequentIncrease: subsequentIncrease,
		Instruments:        instruments,
		lastAlerted:        make(map[uint32]float64),
	}
}

func (a *PriceAlert) Check(token uint32, tick *TickData) *AlertEntry {
	if tick.TotalSellQuantity == 0 || tick.Low <= 0 || tick.LastPrice <= 0 {
		return nil
	}

	ratio := float64(tick.TotalBuyQuantity) / float64(tick.TotalSellQuantity)
	pctFromLow := (tick.LastPrice - tick.Low) / tick.Low

	_, seen := a.lastAlerted[token]
	if pctFromLow > a.PriceThreshold || seen {
		info := a.Instruments[token]
		volumeMultiplier := float64(tick.VolumeTraded) / info.Volume20Ma

		a.mu.Lock()
		lastPrice, seen := a.lastAlerted[token]
		shouldAlert := !seen || math.Abs(tick.LastPrice-lastPrice)/lastPrice >= a.MoveThreshold
		if shouldAlert {
			a.lastAlerted[token] = tick.LastPrice
		}
		a.mu.Unlock()

    if shouldAlert {
      sym := info.Symbol
      if sym == "" {
        sym = "unknown"
      }
      label := "NEW"
      moveStr := "new"
      if seen {
        pctMove := (tick.LastPrice - lastPrice) / lastPrice * 100
        if pctMove > 0 {
          label = fmt.Sprintf("+%.1f%%", pctMove)
        } else {
          label = fmt.Sprintf("%.1f%%", pctMove)
        }
        moveStr = fmt.Sprintf("%+.1f%%", pctMove)
      }
      msg := fmt.Sprintf("%s | %s | up: %.1f%% | vol: %.1fx | vpt: %.0f | div: %.2f | %.2f | b/s: %.2f | %s",
        sym, moveStr, pctFromLow*100, volumeMultiplier, info.VptScore, info.Divergence, tick.LastPrice, ratio, info.Industry)
			log.Printf("%s[ALERT %s] %s%s", colorBlue, label, msg, colorReset)
			exec.Command("osascript", "-e", fmt.Sprintf(`display notification "%s" with title "Ninja Trader" sound name "Glass"`, msg)).Start()
			return &AlertEntry{
				Time:    time.Now(),
				Type:    "PRICE",
				Symbol:  sym,
				Price:   tick.LastPrice,
				Message: msg,
			}
		}
	}
	return nil
}

func (a *VolumeAlert) Check(token uint32, tick *TickData) *AlertEntry {
	info := a.Instruments[token]

	if info.Volume20Ma <= 0 {
		return nil
	}
	volumeMultiplier := float64(tick.VolumeTraded) / info.Volume20Ma

	if volumeMultiplier > a.VolumeThreshold {
		a.mu.Lock()
		lastMultiplier, seen := a.lastAlerted[token]
		subsequentIncrease := volumeMultiplier - lastMultiplier
		shouldAlert := !seen || subsequentIncrease > a.SubsequentIncrease
		if shouldAlert {
			a.lastAlerted[token] = volumeMultiplier
		}
		a.mu.Unlock()

    if shouldAlert {
      sym := info.Symbol
      if sym == "" {
        sym = "unknown"
      }
      label := "NEW"
      moveStr := "new"
      if seen {
        label = fmt.Sprintf("+%.1fx", subsequentIncrease)
        moveStr = fmt.Sprintf("+%.1fx", subsequentIncrease)
      }
      dayChange := 0.0
      if tick.Close > 0 {
        dayChange = (tick.LastPrice - tick.Close) / tick.Close * 100
      }
      msg := fmt.Sprintf("%s | %+.1f%% | vol: %.1fx (%s) | price: %.2f | vpt: %.0f | div: %.2f | %s",
        sym, dayChange, volumeMultiplier, moveStr, tick.LastPrice, info.VptScore, info.Divergence, info.Industry)
			log.Printf("%s[VOLUME %s] %s%s", colorGreen, label, msg, colorReset)
			exec.Command("osascript", "-e", fmt.Sprintf(`display notification "%s" with title "Ninja Trader" sound name "Glass"`, msg)).Start()
			return &AlertEntry{
				Time:    time.Now(),
				Type:    "VOLUME",
				Symbol:  sym,
				Price:   tick.LastPrice,
				Message: msg,
			}
		}
	}
	return nil
}

// ResetAlerts clears the alerted state.
func (a *PriceAlert) ResetAlerts() {
	a.mu.Lock()
	a.lastAlerted = make(map[uint32]float64)
	a.mu.Unlock()
}
