package nse

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	models "ninja-trader/internal/model"

	"github.com/gocarina/gocsv"
)

type EquityRoot struct {
	EquityResponse []EquityResponse `json:"equityResponse"`
}

type EquityResponse struct {
	// OrderBook      OrderBook `json:"orderBook"`
	MetaData       MetaData          `json:"metaData"`
	TradeInfo      TradeInfo         `json:"tradeInfo"`
	PriceInfo      PriceInfo         `json:"priceInfo"`
	SecInfo        SecInfo           `json:"secInfo"`
	LastUpdateTime NseLastUpdateTime `json:"lastUpdateTime"`
}

/* -------------------- ORDER BOOK -------------------- */

// type OrderBook struct {
// 	BuyPrice1         float64 `json:"buyPrice1"`
// 	BuyQuantity1      int64   `json:"buyQuantity1"`
// 	BuyPrice2         float64 `json:"buyPrice2"`
// 	BuyQuantity2      int64   `json:"buyQuantity2"`
// 	BuyPrice3         float64 `json:"buyPrice3"`
// 	BuyQuantity3      int64   `json:"buyQuantity3"`
// 	BuyPrice4         float64 `json:"buyPrice4"`
// 	BuyQuantity4      int64   `json:"buyQuantity4"`
// 	BuyPrice5         float64 `json:"buyPrice5"`
// 	BuyQuantity5      int64   `json:"buyQuantity5"`
// 	SellPrice1        float64 `json:"sellPrice1"`
// 	SellQuantity1     int64   `json:"sellQuantity1"`
// 	SellPrice2        float64 `json:"sellPrice2"`
// 	SellQuantity2     int64   `json:"sellQuantity2"`
// 	SellPrice3        float64 `json:"sellPrice3"`
// 	SellQuantity3     int64   `json:"sellQuantity3"`
// 	SellPrice4        float64 `json:"sellPrice4"`
// 	SellQuantity4     int64   `json:"sellQuantity4"`
// 	SellPrice5        float64 `json:"sellPrice5"`
// 	SellQuantity5     int64   `json:"sellQuantity5"`
// 	LastPrice         float64 `json:"lastPrice"`
// 	TotalBuyQuantity  int64   `json:"totalBuyQuantity"`
// 	TotalSellQuantity int64   `json:"totalSellQuantity"`
// 	PerBuyQty         float64 `json:"perBuyQty"`
// 	PerSellQty        float64 `json:"perSellQty"`
// }

/* -------------------- META DATA -------------------- */

type MetaData struct {
	Identifier  string `json:"identifier"`
	CompanyName string `json:"companyName"`
	IsinCode    string `json:"isinCode"`
	Symbol      string `json:"symbol"`
	// Series          string  `json:"series"`
	// MarketType      string  `json:"marketType"`
	// Open            float64 `json:"open"`
	// DayHigh         float64 `json:"dayHigh"`
	// DayLow          float64 `json:"dayLow"`
	// PreviousClose   float64 `json:"previousClose"`
	// AveragePrice    float64 `json:"averagePrice"`
	// Change          float64 `json:"change"`
	// PChange         float64 `json:"pChange"`
	// BasePrice       float64 `json:"basePrice"`
	// ClosePrice      float64 `json:"closePrice"`
	// IndicativeClose float64 `json:"indicativeClose"`
	// IcChange        float64 `json:"ic_change"`
	// IcPchange       float64 `json:"ic_pchange"`
	// SpoChange       float64 `json:"spoChange"`
	// SpoPchange      float64 `json:"spoPchange"`
	// SymbolStatus    string  `json:"symbolStatus"`
	// AdjPrice        float64 `json:"adjPrice"`
	// Iep             float64 `json:"iep"`
	// Ieq             float64 `json:"ieq"`
}

/* -------------------- TRADE INFO -------------------- */

type TradeInfo struct {
	// TotalTradedVolume        int64       `json:"totalTradedVolume"`
	// TotalTradedValue         float64     `json:"totalTradedValue"`
	// Series                   string      `json:"series"`
	// LastPrice                float64     `json:"lastPrice"`
	// IssuedSize               int64       `json:"issuedSize"`
	// BasePrice                float64     `json:"basePrice"`
	// Ffmc                     float64     `json:"ffmc"`
	// FaceValue                float64     `json:"faceValue"`
	// ImpactCost               float64     `json:"impactCost"`
	DeliveryToTradedQuantity float32 `json:"deliveryToTradedQuantity"`
	// ApplicableMargin         float64     `json:"applicableMargin"`
	// MarketLot                interface{} `json:"marketLot"`
	// QuantityTraded           int64       `json:"quantitytraded"`
	// DeliveryQuantity         int64       `json:"deliveryquantity"`
	TotalMarketCap float64 `json:"totalMarketCap"`
	// SecWiseDelPosDate        string      `json:"secwisedelposdate"`
}

/* -------------------- PRICE INFO -------------------- */

type PriceInfo struct {
	// YearHighDt         string  `json:"yearHightDt"`
	// YearLowDt          string  `json:"yearLowDt"`
	// YearHigh           float64 `json:"yearHigh"`
	// YearLow            float64 `json:"yearLow"`
	PriceBand string `json:"priceBand"`
	// CmDailyVolatility  string  `json:"cmDailyVolatility"`
	// CmAnnualVolatility string  `json:"cmAnnualVolatility"`
	// TickSize           float64 `json:"tickSize"`
	Inav   float64 `json:"inav"`
	IsINav string  `json:"isINav"`
	// PpriceBand         string  `json:"ppriceBand"`
}

/* -------------------- SECURITY INFO -------------------- */

type SecInfo struct {
	// SecStatus                string      `json:"secStatus"`
	ListingDate string `json:"listingDate"`
	// PdSectorInd              string      `json:"pdSectorInd"`
	// PdSectorPe               string      `json:"pdSectorPe"`
	// PdSymbolPe               string      `json:"pdSymbolPe"`
	// IsSuspended              string      `json:"isSuspended"`
	BasicIndustry string `json:"basicIndustry"`
	Index                    string      `json:"index"`
	// DeliveryQuantity         string      `json:"deliveryQuantity"`
	// DeliveryTotradedQuantity string      `json:"deliveryTotradedQuantity"`
	// SecurityVar              string      `json:"securityvar"`
	// IndexVar                 string      `json:"indexvar"`
	// ExtremeLossMargin        string      `json:"extremelossMargin"`
	// VarMargin                string      `json:"varMargin"`
	// AdhocMargin              string      `json:"adhocMargin"`
	// ApplicableMargin         string      `json:"applicableMargin"`
	// BondType                 interface{} `json:"bondType"`
	// IssueDesc                interface{} `json:"issueDesc"`
	// IssueDate                interface{} `json:"issueDate"`
	// MaturityDate             interface{} `json:"maturityDate"`
	// CouponRate               interface{} `json:"couponRate"`
	// NxtIpDate                interface{} `json:"nxtIpDate"`
	// CreditRating             interface{} `json:"creditRating"`
	Macro        string `json:"macro"`
	Sector       string `json:"sector"`
	IndustryInfo string `json:"industryInfo"`
	// IndexList                []string    `json:"indexList"`
	// BoardStatus              string      `json:"boardStatus"`
	// TradingSegment           string      `json:"tradingSegment"`
	// SessionNo                interface{} `json:"sessionNo"`
	// ClassShare               string      `json:"classShare"`
	// NameOfComplianceOfficer  interface{} `json:"nameOfComplianceOfficer"`
	// SddCompliance            interface{} `json:"sddcompliance"`
}

func (equity *EquityRoot) GetMarketCap() float64 {
	return equity.EquityResponse[0].TradeInfo.TotalMarketCap
}

func (equity *EquityRoot) GetPriceBand() string {
	return equity.EquityResponse[0].PriceInfo.PriceBand

}
func (equity *EquityRoot) GetLastUpdateTime() time.Time {
	return time.Time(equity.EquityResponse[0].LastUpdateTime)
}
func (equity *EquityRoot) GetDeliveryToTradePer() float32 {
	return equity.EquityResponse[0].TradeInfo.DeliveryToTradedQuantity
}

func (equity *EquityRoot) GetSecInfo() *SecInfo {
	return &equity.EquityResponse[0].SecInfo
}


type NseLastUpdateTime time.Time

func (d *NseLastUpdateTime) UnmarshalJSON(data []byte) error {
	t, err := time.Parse("\"02-Jan-2006 15:04:05\"", string(data))
	if err != nil {
		return err
	}
	*d = NseLastUpdateTime(t)
	return nil
}

type TradeDate time.Time

func (d *TradeDate) UnmarshalCSV(csv string) error {
	t, err := time.Parse("02-Jan-2006", csv)
	if err != nil {
		return err
	}
	*d = TradeDate(t)
	return nil
}

func (d *TradeDate) UnmarshalJSON(data []byte) error {
	t, err := time.Parse("\"02-Jan-2006\"", string(data))
	if err != nil {
		return err
	}
	*d = TradeDate(t)
	return nil
}

type ShortTrade struct {
	SecurityName string    `csv:"Security Name"`
	SymbolName   string    `csv:"Symbol Name"`
	TradeDate    TradeDate `csv:"Trade Date"`
	Quantity     int       `csv:"Quantity"`
}

type ShortTrade2 struct {
	SecurityName string    `csv:"Security Name"`
	SymbolName   string    `csv:"Security Symbol"`
	TradeDate    TradeDate `csv:"Trade Date"`
	Quantity     int       `csv:"Quantity"`
}

type HistoricalTrade struct {
	O               float64   `json:"chOpeningPrice"`
	H               float64   `json:"chTradeHighPrice"`
	L               float64   `json:"chTradeLowPrice"`
	C               float64   `json:"chClosingPrice"`
	LastTradedPrice float64   `json:"chLastTradedPrice"`
	Vwap            float64   `json:"vwap"`
	Volume          int64     `json:"chTotTradedQty"`
	TradedValue     float64   `json:"chTotTradedVal"`
	NoOfTrades      int64     `json:"chTotalTrades"`
	Symbol          string    `json:"chSymbol"`
	Date            TradeDate `json:"mtimestamp"`
	YearHigh        float64   `json:"ch52WeekHighPrice"`
	YearLow         float64   `json:"ch52WeekLowPrice"`
}

func (nseModel HistoricalTrade) MapToDb() *models.Historicaldata {
	trade := &models.Historicaldata{
		O:               nseModel.O,
		H:               nseModel.H,
		L:               nseModel.L,
		C:               nseModel.C,
		LastTradedPrice: nseModel.LastTradedPrice,
		Vwap:            nseModel.Vwap,
		Volume:          nseModel.Volume,
		NoOfTrades:      nseModel.NoOfTrades,
		Symbol:          nseModel.Symbol,
		Date:            time.Time(nseModel.Date),
		YearHigh:        nseModel.YearHigh,
		YearLow:         nseModel.YearLow,
	}

	if nseModel.NoOfTrades > 0 {
		trade.VolumePerTrade = nseModel.Volume / nseModel.NoOfTrades
	}
	return trade
}

func (nse ShortTrade) MapToShortModel() *models.Shorts {
	return &models.Shorts{
		TradingSymbol: nse.SymbolName,
		SecurityName:  nse.SecurityName,
		Quantity:      int64(nse.Quantity),
		Date:          time.Time(nse.TradeDate),
	}
}

func (nse ShortTrade2) MapToShortModel() *models.Shorts {
	return &models.Shorts{
		TradingSymbol: nse.SymbolName,
		SecurityName:  nse.SecurityName,
		Quantity:      int64(nse.Quantity),
		Date:          time.Time(nse.TradeDate),
	}
}

var nseHTTPClient = &http.Client{
	Transport: &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	},
	Timeout: 15 * time.Second,
}

func GetNseDetails(symbol, segment string) (*EquityRoot, error) {
	url := fmt.Sprintf("https://www.nseindia.com/api/NextApi/apiClient/GetQuoteApi?functionName=getSymbolData&marketType=N&series=%s&symbol=%s", segment, url.QueryEscape(symbol))
	method := "GET"

	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		log.Printf("Error creating req %s\n", err)
		return nil, err
	}

	req.Header.Add("Accept", "*/*")
	req.Header.Add("User-Agent", "PostmanRuntime/7.51.1")

	reqDump, err := httputil.DumpRequestOut(req, true)
	fmt.Println(string(reqDump))

	res, err := nseHTTPClient.Do(req)
	if err != nil {
		log.Printf("Error fetching nse data %s\n", err)
		return nil, err
	}
	defer res.Body.Close()

	var result EquityRoot

	if err = json.NewDecoder(res.Body).Decode(&result); err != nil {
		log.Printf("Error decoding json response %s, %s \n", err, url)
		return nil, err
	}
	return &result, nil
}

func GetShortTrades(date time.Time) ([]ShortTrade, error) {
	formattedDate := date.Format("02012006")
	url := fmt.Sprintf("https://nsearchives.nseindia.com/archives/equities/shortSelling/shortselling_%s.csv", formattedDate)
	method := "GET"

	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		log.Printf("Error creating req %s\n", err)
		return nil, err
	}

	req.Header.Add("Accept", "*/*")
	req.Header.Add("User-Agent", "PostmanRuntime/7.51.1")

	reqDump, err := httputil.DumpRequestOut(req, true)
	fmt.Println(string(reqDump))

	res, err := nseHTTPClient.Do(req)
	if err != nil {
		log.Printf("Error fetching req %s\n", err)
		return nil, err
	}
	resDump, err := httputil.DumpResponse(res, true)
	fmt.Println(string(resDump))
	if res.StatusCode == 404 {
		return nil, fmt.Errorf("Report for %s is not yet available please try after some time", formattedDate)
	}
	defer res.Body.Close()

	var trades []ShortTrade
	if err := gocsv.Unmarshal(res.Body, &trades); err != nil {
		return nil, err
	}
	return trades, nil
}

func GetHistoricalData(symbol, series string, from, to time.Time) ([]HistoricalTrade, error) {
	var comibnedData []HistoricalTrade
	for d := from; !d.After(to); d = d.AddDate(1, 0, 0) {
		fromStr := d.Format("02-01-2006")
		var currTo time.Time
		if to.Before(d.AddDate(1, 0, 0)) {
			currTo = to
		} else {
			currTo = d.AddDate(1, 0, 0)
		}
		currToStr := currTo.Format("02-01-2006")
		url := fmt.Sprintf("https://www.nseindia.com/api/NextApi/apiClient/GetQuoteApi?functionName=getHistoricalTradeData&symbol=%s&series=%s&fromDate=%s&toDate=%s&csv=true", url.QueryEscape(symbol), series, fromStr, currToStr)
		method := "GET"

		req, err := http.NewRequest(method, url, nil)

		if err != nil {
			log.Printf("Error creating req %s\n", err)
			return nil, err
		}

		req.Header.Add("Accept", "*/*")
		req.Header.Add("User-Agent", "PostmanRuntime/7.51.1")

		reqDump, err := httputil.DumpRequestOut(req, true)
		fmt.Println(string(reqDump))

		res, err := nseHTTPClient.Do(req)
		if err != nil {
			log.Printf("Error fetching req %s\n", err)
			return nil, err
		}

		var result []HistoricalTrade
		if err = json.NewDecoder(res.Body).Decode(&result); err != nil {
			log.Printf("Error decoding json response %s, %s \n", err, url)
			return nil, err
		}
		comibnedData = append(comibnedData, result...)
		defer res.Body.Close()
	}
	return comibnedData, nil

}
