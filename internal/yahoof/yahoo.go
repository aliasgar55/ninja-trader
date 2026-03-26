package yahoo

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

type Root struct {
	Chart Chart `json:"chart"`
}

type Chart struct {
	Result []Result  `json:"result"`
	Error  *YahooError `json:"error"`
}

type YahooError struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

type Result struct {
	Timestamp  []int64    `json:"timestamp"`
	Events     Events     `json:"events"`
	Indicators Indicators `json:"indicators"`
}


type Events struct {
	Dividends map[string]Dividend `json:"dividends"`
	Splits map[string]Split `json:"splits"`
}

type Split struct {
	Date        int64   `json:"date"`
	Numerator   float64 `json:"numerator"`
	Denominator float64 `json:"denominator"`
	SplitRatio  string  `json:"splitRatio"`
}

type Dividend struct {
	Amount float64 `json:"amount"`
	Date   int64   `json:"date"`
}

type Indicators struct {
	Quote    []Quote    `json:"quote"`
	AdjClose []AdjClose `json:"adjclose"`
}

type Quote struct {
	Close  []float64 `json:"close"`
	Volume []int64   `json:"volume"`
	Low    []float64 `json:"low"`
	Open   []float64 `json:"open"`
	High   []float64 `json:"high"`
}

type AdjClose struct {
	AdjClose []float64 `json:"adjclose"`
}


func GetHistoricalData(symbol string, startDate, endDate time.Time) (*Root, error) {

	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s.NS?events=%s&interval=1d&period1=%d&period2=%d&symbol=%s.NS",  url.QueryEscape(symbol), url.QueryEscape("div|split"), startDate.Unix(), endDate.Unix(), url.QueryEscape(symbol))
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		log.Fatalf("Error creating  request from yahoo %s\n", err)
	}
	req.Header.Add("Accept", "*/*")
	req.Header.Add("User-Agent", "PostmanRuntime/7.51.1")

	reqDump, err := httputil.DumpRequestOut(req, true)
	fmt.Println(string(reqDump))

	res, err := client.Do(req)
	if err != nil {
		log.Printf("Error fetching data from yahoo, %s\n", err)
		return nil, err
	}
	defer res.Body.Close()

	var result Root
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		log.Printf("Error decoding json response %s, %s \n", err, url)
		return nil, err
	}
	return &result, nil
}
