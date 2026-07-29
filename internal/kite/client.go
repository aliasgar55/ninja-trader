package kite

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	tradingClient "ninja-trader/internal/trading_client"
	"strconv"
	"strings"

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

// Alert API types
type KiteAlertResponse struct {
	UUID string `json:"uuid"`
}

type KiteAlertItem struct {
	UUID             string  `json:"uuid"`
	Name             string  `json:"name"`
	Type             string  `json:"type"`
	Status           string  `json:"status"`
	LHSExchange      string  `json:"lhs_exchange"`
	LHSTradingSymbol string  `json:"lhs_tradingsymbol"`
	LHSAttribute     string  `json:"lhs_attribute"`
	Operator         string  `json:"operator"`
	RHSType          string  `json:"rhs_type"`
	RHSConstant      float64 `json:"rhs_constant"`
	AlertCount       int     `json:"alert_count"`
}

func (c *Client) authHeader() string {
	return fmt.Sprintf("token %s:%s", c.apiKey, c.accessToken)
}

// CreateAlert creates a price alert on Kite.
func (c *Client) CreateAlert(exchange, tradingSymbol, operator string, value float64) (string, error) {
	data := url.Values{}
	data.Set("name", tradingSymbol)
	data.Set("lhs_exchange", exchange)
	data.Set("lhs_tradingsymbol", tradingSymbol)
	data.Set("lhs_attribute", "LastTradedPrice")
	data.Set("operator", operator)
	data.Set("rhs_type", "constant")
	data.Set("type", "simple")
	data.Set("rhs_constant", strconv.FormatFloat(value, 'f', 2, 64))

	req, err := http.NewRequest("POST", "https://api.kite.trade/alerts", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("X-Kite-Version", "3")
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("kite alerts API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data KiteAlertResponse `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	return result.Data.UUID, nil
}

// GetAlerts fetches all alerts from Kite.
func (c *Client) GetAlerts() ([]KiteAlertItem, error) {
	req, err := http.NewRequest("GET", "https://api.kite.trade/alerts", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Kite-Version", "3")
	req.Header.Set("Authorization", c.authHeader())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("kite alerts API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []KiteAlertItem `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// DeleteAlert deletes an alert on Kite.
func (c *Client) DeleteAlert(alertID string) error {
	req, err := http.NewRequest("DELETE", "https://api.kite.trade/alerts?uuid="+alertID, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Kite-Version", "3")
	req.Header.Set("Authorization", c.authHeader())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("kite alerts delete error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
