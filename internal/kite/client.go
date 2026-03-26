package kite

import (
	tradingClient "ninja-trader/internal/trading_client"

	kiteconnect "github.com/zerodha/gokiteconnect/v4"
)

type Client struct {
	client *kiteconnect.Client
}

func New(apiKey string) *Client {
	kc := kiteconnect.New(apiKey)
	return &Client{
		client: kc,
	}
}

func (c *Client) GetAllInstruments() (*[]tradingClient.Instrument, error) {
	instruments, err := c.client.GetInstruments()
	dst := make([]tradingClient.Instrument, len(instruments))
	for i, v := range(instruments) {
		dst[i] = tradingClient.Instrument(v)
	}
	return &dst, err
}
