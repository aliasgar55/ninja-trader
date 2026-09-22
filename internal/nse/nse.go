package nse

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	models "ninja-trader/internal/model"

	"github.com/gocarina/gocsv"
)

var nseHTTPClient = &http.Client{
	Transport: &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	},
	Timeout: 15 * time.Second,
}


type NseLastUpdateTime time.Time
type TradeDate time.Time

func (d *NseLastUpdateTime) UnmarshalJSON(data []byte) error {
	t, err := time.Parse("\"02-Jan-2006 15:04:05\"", string(data))
	if err != nil {
		return err
	}
	*d = NseLastUpdateTime(t)
	return nil
}

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
// 	BuyQuantity4      int64   `json:"buyQuantity3"`
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
	AdjPrice float64 `json:"adjPrice"`
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
	// Iep             float64 `json:"iep"`
	// Ieq             float64 `json:"ieq"`
}

/* -------------------- TRADE INFO -------------------- */

type TradeInfo struct {
	DeliveryToTradedQuantity float32 `json:"deliveryToTradedQuantity"`
	TotalMarketCap float64 `json:"totalMarketCap"`
	// TotalTradedVolume        int64       `json:"totalTradedVolume"`
	// TotalTradedValue         float64     `json:"totalTradedValue"`
	// Series                   string      `json:"series"`
	// LastPrice                float64     `json:"lastPrice"`
	// IssuedSize               int64       `json:"issuedSize"`
	// BasePrice                float64     `json:"basePrice"`
	// Ffmc                     float64     `json:"ffmc"`
	// FaceValue                float64     `json:"faceValue"`
	// ImpactCost               float64     `json:"impactCost"`
	// ApplicableMargin         float64     `json:"applicableMargin"`
	// MarketLot                interface{} `json:"marketLot"`
	// QuantityTraded           int64       `json:"quantitytraded"`
	// DeliveryQuantity         int64       `json:"deliveryquantity"`
	// SecWiseDelPosDate        string      `json:"secwisedelposdate"`
}

/* -------------------- PRICE INFO -------------------- */

type PriceInfo struct {
	PriceBand string `json:"priceBand"`
	Inav   float64 `json:"inav"`
	IsINav string  `json:"isINav"`
	// PpriceBand         string  `json:"ppriceBand"`
	// YearHighDt         string  `json:"yearHightDt"`
	// YearLowDt          string  `json:"yearLowDt"`
	// YearHigh           float64 `json:"yearHigh"`
	// YearLow            float64 `json:"yearLow"`
	// CmDailyVolatility  string  `json:"cmDailyVolatility"`
	// CmAnnualVolatility string  `json:"cmAnnualVolatility"`
	// TickSize           float64 `json:"tickSize"`
}

/* -------------------- SECURITY INFO -------------------- */

type SecInfo struct {
	BasicIndustry string `json:"basicIndustry"`
	Index         string `json:"index"`
	ListingDate string `json:"listingDate"`
	Macro        string `json:"macro"`
	Sector       string `json:"sector"`
	IndustryInfo string `json:"industryInfo"`
	// SecStatus                string      `json:"secStatus"`
	// PdSectorInd              string      `json:"pdSectorInd"`
	// PdSectorPe               string      `json:"pdSectorPe"`
	// PdSymbolPe               string      `json:"pdSymbolPe"`
	// IsSuspended              string      `json:"isSuspended"`
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

func (equity *EquityRoot) GetNeedsAdjustment() bool {
	return equity.EquityResponse[0].MetaData.AdjPrice > 0
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
	
	log.Printf("Calling NSE Details: %s\n", url)
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

// SHORT TRADES START

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

	log.Printf("Calling NSE Short Trades: %s\n", url)
	res, err := nseHTTPClient.Do(req)
	if err != nil {
		log.Printf("Error fetching req %s\n", err)
		return nil, err
	}
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
// SHORT TRADES ENDS


// METADATA STARTS
// WE NEED THIS TO GET THE ACTIVE SERIES FOR A SYMBOL WHICH WILL BE USED AS INPUT FOR GETTING THE HISTORICAL DATA AND DETAIL API
type SymbolMetaData struct {
	Symbol       string   `json:"symbol"`
	ActiveSeries []string `json:"activeSeries"`
	IsFNOSec     string   `json:"isFNOSec"`
	// CompanyName         string        `json:"companyName"`
	// DebtSeries          []interface{} `json:"debtSeries"`
	// IsCASec             string        `json:"isCASec"`
	// IsSLBSec            string        `json:"isSLBSec"`
	// IsDebtSec           string        `json:"isDebtSec"`
	// TempSuspendedSeries []interface{} `json:"tempSuspendedSeries"`
	// IsSuspended         string        `json:"isSuspended"`
	// IsETFSec            string        `json:"isETFSec"`
	// IsDelisted          string        `json:"isDelisted"`
	// Isin                string        `json:"isin"`
	// IsMunicipalBond     string        `json:"isMunicipalBond"`
	// IsHybridSymbol      string        `json:"isHybridSymbol"`
	// MarketType          string        `json:"marketType"`
	// ParentSymbol        string        `json:"parentSymbol"`
}


func (metaData SymbolMetaData) GetActiveSeries() (string, error) {
	activeSeries := metaData.ActiveSeries
	if len(activeSeries) == 0 {
		return "", fmt.Errorf("No active series found for symbol: %s", metaData.Symbol)

	}
	for _, series := range activeSeries {
		if series == "EQ" {
			return series, nil
		} else if series == "BE" {
			return series, nil
		}
	}
	log.Printf("New Series found %s for symbol: %s", activeSeries[0], metaData.Symbol)
	return activeSeries[0], nil
}


func GetMetaData(symbol string) (*SymbolMetaData, error) {

	url := fmt.Sprintf("https://www.nseindia.com/api/NextApi/apiClient/GetQuoteApi?functionName=getMetaData&symbol=%s", url.QueryEscape(symbol))
	method := "GET"

	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		log.Printf("Error creating req %s\n", err)
		return nil, err
	}

	req.Header.Add("Accept", "*/*")
	req.Header.Add("User-Agent", "PostmanRuntime/7.51.1")

	fmt.Printf("Calling: %s\n", url)
	res, err := nseHTTPClient.Do(req)
	if err != nil {
		log.Printf("Error fetching req %s\n", err)
		return nil, err
	}
	if res.StatusCode == 404 {
		return nil, fmt.Errorf("Error getting metadata for symbol: %s", symbol)
	}
	defer res.Body.Close()

	var result SymbolMetaData

	if err = json.NewDecoder(res.Body).Decode(&result); err != nil {
		log.Printf("Error decoding json response %s, %s \n", err, url)
		return nil, err
	}
	return &result, nil

}
// METADATA ENDS


// HISTORICAL TRDES START
type HistoricalTrade struct {
	O               float64   `json:"chOpeningPrice"`
	H               float64   `json:"chTradeHighPrice"`
	L               float64   `json:"chTradeLowPrice"`
	C               float64   `json:"chClosingPrice"`
	LastTradedPrice float64   `json:"chLastTradedPrice"`
	Vwap            float64   `json:"vwap"`
	Volume          uint64    `json:"chTotTradedQty"`
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
		trade.VolumePerTrade = int64(nseModel.Volume) / nseModel.NoOfTrades
	}
	return trade
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

		log.Printf("Calling: %s\n", url)
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
// HISTORICAL TRADES ENDS

// INSIDER TRADES START
type InsideTradesRoot struct {
	AcqNameList []string      `json:"acqNameList"`
	Data        []InsideTrade `json:"data"`
}

type InsideTrade struct {
	AcqMode                   string `json:"acqMode"`
	AcqName                   string `json:"acqName"`
	AcqfromDt                 string `json:"acqfromDt"`
	AcqtoDt                   string `json:"acqtoDt"`
	AfterAcqSharesNo          string `json:"afterAcqSharesNo"`
	AfterAcqSharesPer         string `json:"afterAcqSharesPer"`
	Anex                      string `json:"anex"`
	BefAcqSharesNo            string `json:"befAcqSharesNo"`
	BefAcqSharesPer           string `json:"befAcqSharesPer"`
	BuyQuantity               string `json:"buyQuantity"`
	BuyValue                  string `json:"buyValue"`
	Company                   string `json:"company"`
	Date                      string `json:"date"`
	DerivativeType            string `json:"derivativeType"`
	Did                       string `json:"did"`
	Exchange                  string `json:"exchange"`
	IntimDt                   string `json:"intimDt"`
	PersonCategory            string `json:"personCategory"`
	Pid                       string `json:"pid"`
	Remarks                   string `json:"remarks"`
	SecAcq                    string `json:"secAcq"`
	SecType                   string `json:"secType"`
	SecVal                    string `json:"secVal"`
	SecuritiesTypePost        string `json:"securitiesTypePost"`
	SellValue                 string `json:"sellValue"`
	Sellquantity              string `json:"sellquantity"`
	Symbol                    string `json:"symbol"`
	TdpDerivativeContractType string `json:"tdpDerivativeContractType"`
	TdpTransactionType        string `json:"tdpTransactionType"`
	TkdAcqm                   any    `json:"tkdAcqm"`
	Xbrl                      string `json:"xbrl"`
	XbrlFileSize              any    `json:"xbrlFileSize"`
}

func (t *InsideTrade) MapToDb() (*models.InsiderTradeTransaction, models.InsiderTradeEntity, error) {
	did, err := strconv.ParseUint(t.Did, 10, 64)
	if err != nil {
		log.Fatalf("Error converting did: %s to uint\n", t.Did)
		return nil, models.InsiderTradeEntity{}, err
	}
	quantity, err := strconv.ParseInt(t.SecAcq, 10, 64)
	if err != nil {
		quantity = 0
	}
	value, err := strconv.ParseFloat(t.SecVal, 64)
	if err != nil {
		value = 0
	}
	beforeAcqShare, err := strconv.ParseInt(t.BefAcqSharesNo, 10, 64)
	if err != nil {
		beforeAcqShare = 0
	}
	beforeAcqPer, err := strconv.ParseFloat(t.BefAcqSharesPer, 64)
	if err != nil {
		beforeAcqPer = 0.0
	}
	afterAcqShare, err := strconv.ParseInt(t.AfterAcqSharesNo, 10, 64)
	if err != nil {
		afterAcqShare = 0
	}
	afterAcqPer, err := strconv.ParseFloat(t.AfterAcqSharesPer, 64)
	if err != nil {
		afterAcqPer = 0.0
	}

	txnDate, err := time.Parse("02-Jan-2006", t.AcqtoDt)
	intimDt, err := time.Parse("02-Jan-2006", t.IntimDt)
	nseDate, err := time.Parse("02-Jan-2006 15:04", t.Date)

	transaction := models.InsiderTradeTransaction{
		FilingID:         uint(did),
		Symbol:           t.Symbol,
		SecurityType:     t.SecType,
		AcqMode:          t.AcqMode,
		TransactionType:  t.TdpTransactionType,
		TransactionMode:  t.AcqMode,
		TransactionDate:  txnDate,
		IntimationDate:   intimDt,
		NseDate:          nseDate,
		Quantity:         quantity,
		Value:            value,
		HoldingBefore:    beforeAcqShare,
		HoldingAfter:     afterAcqShare,
		HoldingBeforePct: beforeAcqPer,
		HoldingAfterPct:  afterAcqPer,
		Exchange:         t.Exchange,
		Notes:            t.Remarks,
		DerevativeType:   t.DerivativeType,
		SecurityTypePost: t.SecuritiesTypePost,
	}

	entity := models.InsiderTradeEntity{
		Name:     t.AcqName,
		Category: t.PersonCategory,
	}

	return &transaction, entity, nil

}

func GetInsideTrades(from, to time.Time) ([]InsideTrade, error) {
	// loop while adding 1 year to "from" until it is greater than "to" nse api only return 1 year data at max
	combinedResult := []InsideTrade{}
	for d := from; !d.After(to); d = d.AddDate(1, 0, 0) {
		dateFrom := d.Format("02-01-2006")
		toCurr := d.AddDate(1, 0, 0)
		if toCurr.After(to) || toCurr.Equal(to) {
			toCurr = to
		}
		toStr := toCurr.Format("02-01-2006")
		url := fmt.Sprintf("https://www.nseindia.com/api/corporates-pit?index=equities&from_date=%s&to_date=%s", dateFrom, toStr)
		log.Printf("Calling: %s\n", url)
		method := "GET"
		req, err := http.NewRequest(method, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Add("Accept", "*/*")
		req.Header.Add("User-Agent", "PostmanRuntime/7.51.1")
		res, err := nseHTTPClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()
		var result InsideTradesRoot
		if err = json.NewDecoder(res.Body).Decode(&result); err != nil {
			return nil, err
		}
		combinedResult = append(combinedResult, result.Data...)

	}
	return combinedResult, nil
}
// INSIDER TRADES ENDS


// BULK AND BLOCK DEALS STARTS
func (deal *BulkBlockDeals) MapToDb() (models.BulkBlockDeal, error) {
	date, err := time.Parse("02-Jan-2006", deal.BDDTDATE)

	if date.After(time.Now()) {
		log.Printf("Date is in the future: %s\n", deal.BDDTDATE)
		return models.BulkBlockDeal{}, fmt.Errorf("Date is in the future: %s", deal.BDDTDATE)
	}
	if err != nil {
		log.Printf("Error parsing date: %s\n", deal.BDDTDATE)
		return models.BulkBlockDeal{}, err
	}
	trimedPrice := strings.ReplaceAll(deal.BDTPWATP, ",", "")
	price, err := strconv.ParseFloat(trimedPrice, 64)
	if err != nil {
		log.Printf("Error parsing price: %s to float, err: %v\n", deal.BDTPWATP, err)
		return models.BulkBlockDeal{}, err
	}

	trimedQty := strings.ReplaceAll(deal.BDQTYTRD, ",", "")
	qty, err := strconv.ParseInt(trimedQty, 10, 64)
	if err != nil {
		log.Printf("Error parsing quantity: %s to int, err: %v\n", deal.BDQTYTRD, err)
		return models.BulkBlockDeal{}, err
	}
	
	var buySell models.TradeType
	if deal.BDBUYSELL == "BUY" {
		buySell = models.Buy
	} else if deal.BDBUYSELL == "SELL" {
		buySell = models.Sell
	} else {
		slog.Error("Unkown Deal type found", "dealType", deal.BDBUYSELL)
		return models.BulkBlockDeal{}, errors.New("Unknow dealType found")
	}


	dbModel := models.BulkBlockDeal{
		TradingSymbol: strings.TrimSpace(deal.BDSYMBOL),
		ScripName:     strings.TrimSpace(deal.BDSCRIPNAME),
		ClientName:    strings.TrimSpace(deal.BDCLIENTNAME),
		BuySell:       buySell,
		Quantity:      qty,
		Price:         price,
		Date:          date,
		Remarks:       strings.TrimSpace(deal.BDREMARKS),
		DealType:      deal.dealType,
	}
	return dbModel, nil

}

type BulkBlockDeals struct {
	BDDTDATE     string `csv:"Date "`
	BDSYMBOL     string `csv:"Symbol "`
	BDSCRIPNAME  string `csv:"Security Name "`
	BDCLIENTNAME string `csv:"Client Name "`
	BDBUYSELL    string `csv:"Buy / Sell "`
	BDQTYTRD     string `csv:"Quantity Traded "`
	BDTPWATP     string `csv:"Trade Price / Wght. Avg. Price "`
	BDREMARKS    string `csv:"Remarks "`
	dealType     models.DealType
}

func GetBulkBlockDeals(from, to time.Time, dealType models.DealType) ([]BulkBlockDeals, error) {
	dateRanges := SplitDateRangeByYear(from, to)
	var result []BulkBlockDeals
	var optionType string

	if dealType == models.DealTypeBulk {
		optionType = "bulk_deals"
	} else if dealType == models.DealTypeBlock {
		optionType = "block_deals"
	}

	for dateRange := range dateRanges {
		dateFrom := dateRanges[dateRange][0].Format("02-01-2006")
		dateTo := dateRanges[dateRange][1].Format("02-01-2006")
		url := fmt.Sprintf("https://www.nseindia.com/api/historicalOR/bulk-block-short-deals?optionType=%s&from=%s&to=%s&csv=true", optionType, dateFrom, dateTo)
		method := "GET"

		req, err := http.NewRequest(method, url, nil)

		if err != nil {
			return nil, err
		}
		req.Header.Add("Accept", "*/*")
		req.Header.Add("User-Agent", "PostmanRuntime/7.51.1")
		log.Printf("Calling: %s\n", url)
		res, err := nseHTTPClient.Do(req)
		if err != nil {
			return nil, err
		}
		if res.StatusCode != http.StatusOK {
			panic(fmt.Sprintf("API returned status: %s", res.Status))
		}

		defer res.Body.Close()
		var intResult []BulkBlockDeals

		resBody, err := skipBOM(res.Body)

		if err = gocsv.Unmarshal(resBody, &intResult); err != nil {
			log.Printf("Error unmarshalling csv response: %v\n", err)
			return nil, err
		}

		log.Printf("Fetched %d %s deals from %s to %s\n", len(intResult), dealType, dateFrom, dateTo)
		for r := range intResult {
			intResult[r].dealType = dealType
		}
		result = append(result, intResult...)
	}
	log.Printf("Fetched total %d %s deals from %s to %s\n", len(result), dealType, from.Format("02-01-2006"), to.Format("02-01-2006"))
	return result, nil

}
// BULK AND BLOCK DEALS END

type SymbolChange struct {
	OldSymbol string
	NewSymbol string
}

func GetSymbolChanges() ([]SymbolChange, error) {
	// TODO: fetch from NSE corporate actions API
	return []SymbolChange{}, nil
}

