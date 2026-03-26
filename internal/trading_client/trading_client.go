package tradingClient

import (
	"strings"

	kiteconnect "github.com/zerodha/gokiteconnect/v4"
)




type Client interface {
	GetAllInstruments() ([]any, error)
}

type Instrument kiteconnect.Instrument

func (instru *Instrument) IsTradingAllowed() bool {
	if instru.Segment == "NSE" &&  instru.Exchange == "NSE" && instru.InstrumentType == "EQ" && instru.LotSize == 1 {
		if strings.Contains(instru.Tradingsymbol, "-") {
			return false
		}
		return true
	}
	return false
}

