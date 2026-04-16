package kite

import (
	tradingClient "ninja-trader/internal/trading_client"

	kiteconnect "github.com/zerodha/gokiteconnect/v4"
)

type Client struct {
	client      *kiteconnect.Client
	apiKey      string
	accessToken string
}

func New(apiKey string) *Client {
	kc := kiteconnect.New(apiKey)
	return &Client{
		client: kc,
		apiKey: apiKey,
	}
}

func (c *Client) SetAccessToken(token string) {
	c.accessToken = token
	c.client.SetAccessToken(token)
}

func (c *Client) GetAPIKey() string {
	return c.apiKey
}

func (c *Client) GetAccessToken() string {
	return c.accessToken
}

func (c *Client) GetAllInstruments() (*[]tradingClient.Instrument, error) {
	instruments, err := c.client.GetInstruments()
	dst := make([]tradingClient.Instrument, len(instruments))
	for i, v := range instruments {
		dst[i] = tradingClient.Instrument(v)
	}
	return &dst, err
}

func (c *Client) PlaceGTT(params kiteconnect.GTTParams) (kiteconnect.GTTResponse, error) {
	return c.client.PlaceGTT(params)
}

func (c *Client) GetGTTs() (kiteconnect.GTTs, error) {
	return c.client.GetGTTs()
}

func (c *Client) GetGTT(triggerID int) (kiteconnect.GTT, error) {
	return c.client.GetGTT(triggerID)
}

func (c *Client) ModifyGTT(triggerID int, params kiteconnect.GTTParams) (kiteconnect.GTTResponse, error) {
	return c.client.ModifyGTT(triggerID, params)
}

func (c *Client) DeleteGTT(triggerID int) (kiteconnect.GTTResponse, error) {
	return c.client.DeleteGTT(triggerID)
}

func (c *Client) GetLoginURL() string {
	return c.client.GetLoginURL()
}

func (c *Client) GenerateSession(requestToken string, apiSecret string) (kiteconnect.UserSession, error) {
	return c.client.GenerateSession(requestToken, apiSecret)
}
