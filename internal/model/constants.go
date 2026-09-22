package models

type DealType string
const (
	DealTypeBulk  DealType = "BULK"
	DealTypeBlock DealType = "BLOCK"
)

type TradeType int

const (
	Buy TradeType = iota
	Sell
)

type EventType int

const (
	Dividend EventType = iota
	Split
)

