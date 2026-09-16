package bse

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	models "ninja-trader/internal/model"

)

var bseHTTPClient = &http.Client{
	Transport: &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	},
	Timeout: 15 * time.Second,
}


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
		fmt.Printf("Calling: %s: \n", url)
		method := "GET"
		req, err := http.NewRequest(method, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Add("Accept", "*/*")
		req.Header.Add("User-Agent", "PostmanRuntime/7.51.1")
		res, err := bseHTTPClient.Do(req)
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

