package handler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	models "ninja-trader/internal/model"
	repo "ninja-trader/internal/repository"
	"ninja-trader/internal/service"
	"strconv"
	"time"
)

func formatIndianNumber(n float64) string {
	intPart := int64(n)
	s := strconv.FormatInt(intPart, 10)
	if len(s) <= 3 {
		return s
	}
	result := s[len(s)-3:]
	s = s[:len(s)-3]
	for len(s) > 2 {
		result = s[len(s)-2:] + "," + result
		s = s[:len(s)-2]
	}
	return fmt.Sprintf("%s,%s", s, result)
}

type InstrumentHandler struct {
	Repo      repo.InstrumentRepo
	TradeRepo repo.TradeRepo
	Service   *service.InstrumentService
	Tmpl      *template.Template
}

func (h *InstrumentHandler) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	sort := r.URL.Query().Get("sort")
	order := r.URL.Query().Get("order")
	instruments, err := h.Repo.GetAllInstrumentsWithHolding(query, sort, order)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.Tmpl.ExecuteTemplate(w, "instruments.html", struct {
		Instruments []repo.InstrumentWithHolding
		Query       string
		Sort        string
		Order       string
	}{instruments, query, sort, order})
}

func (h *InstrumentHandler) Detail(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	period := r.URL.Query().Get("period")
	if period == "" {
		redirectURL := "/instrument?symbol=" + url.QueryEscape(symbol) + "&period=max"
		if r.URL.Query().Get("watchlist") == "1" {
			redirectURL += "&watchlist=1"
		}
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}
	if period == "max" {
		period = ""
	}
	watchlistOnly := r.URL.Query().Get("watchlist") == "1"
	var prev, next string
	if watchlistOnly {
		prev, next = h.Repo.GetAdjacentWatchlistSymbols(symbol)
	} else {
		prev, next = h.Repo.GetAdjacentSymbols(symbol)
	}
	fullName := ""
	watchlist := false
	marketCap := ""
	basicIndustry := ""
	index := ""
	holding := 0
	needsAdjustment := false
	var deliveryPct float32
	var updatedAt time.Time
	tag := ""
	if inst, err := h.Repo.GetInstrumentBySymbol(symbol); err == nil {
		fullName = inst.InstrumentFullName
		watchlist = inst.Watchlist
		marketCap = formatIndianNumber(inst.MarketCap)
		basicIndustry = inst.BasicIndustry
		index = inst.Index
		updatedAt = inst.UpdatedAt
		needsAdjustment = inst.NeedsAdjsutment
		tag = inst.Tag
		if trade, err := h.TradeRepo.GetHoldingByInstrumentID(inst.ID); err == nil {
			holding = trade
		}
		if dp, err := h.Repo.GetLatestDeliveryPercentage(symbol); err == nil {
			deliveryPct = dp
		}
	}
	tmplPeriod := period
	if period == "" {
		tmplPeriod = "max"
	}
	lastUpdated := ""
	if !updatedAt.IsZero() {
		lastUpdated = updatedAt.Format("02/01/06 15:04:05")
	}
	h.Tmpl.ExecuteTemplate(w, "instrument_detail.html", struct {
		Symbol             string
		FullName           string
		MarketCap          string
		BasicIndustry      string
		Index              string
		Msg                string
		Period             string
		PrevSymbol         string
		NextSymbol         string
		Watchlist          bool
		WatchlistOnly      bool
		Holding            int
		LastUpdated        string
		NeedsAdjustment    bool
		DeliveryPercentage float32
		Tag                string
		TagOptions         []string
	}{symbol, fullName, marketCap, basicIndustry, index, r.URL.Query().Get("msg"), tmplPeriod, prev, next, watchlist, watchlistOnly, holding, lastUpdated, needsAdjustment, deliveryPct, tag, []string{"oversold", "touch", "scoop", "overbought", "repel", "horizontal"}})
}

func (h *InstrumentHandler) ToggleWatchlist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	symbol := r.FormValue("symbol")
	if symbol == "" {
		http.Error(w, "symbol required", http.StatusBadRequest)
		return
	}
	newState, err := h.Repo.ToggleWatchlist(symbol)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if r.Header.Get("X-Requested-With") == "fetch" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"watchlist": newState})
		return
	}
	redirectURL := "/instrument?symbol=" + url.QueryEscape(symbol)
	if period := r.FormValue("period"); period != "" {
		redirectURL += "&period=" + url.QueryEscape(period)
	}
	if r.FormValue("watchlist_only") == "1" {
		redirectURL += "&watchlist=1"
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (h *InstrumentHandler) WatchlistPage(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	sort := r.URL.Query().Get("sort")
	order := r.URL.Query().Get("order")
	instruments, err := h.Repo.GetWatchlistInstrumentsWithHolding(query, sort, order)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.Tmpl.ExecuteTemplate(w, "watchlist.html", struct {
		Instruments []repo.InstrumentWithHolding
		Query       string
		Sort        string
		Order       string
	}{instruments, query, sort, order})
}

func parsePeriodSince(period string) *time.Time {
	now := time.Now()
	var t time.Time
	switch period {
	case "1m":
		t = now.AddDate(0, -1, 0)
	case "6m":
		t = now.AddDate(0, -6, 0)
	case "1y":
		t = now.AddDate(-1, 0, 0)
	case "3y":
		t = now.AddDate(-3, 0, 0)
	case "5y":
		t = now.AddDate(-5, 0, 0)
	case "10y":
		t = now.AddDate(-10, 0, 0)
	default:
		return nil
	}
	return &t
}

func (h *InstrumentHandler) ChartData(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		http.Error(w, "symbol required", http.StatusBadRequest)
		return
	}
	since := parsePeriodSince(r.URL.Query().Get("period"))
	var rows []models.Historicaldata
	var err error
	if since != nil {
		rows, err = h.Repo.GetHistoricalDataBySymbolSince(symbol, *since)
	} else {
		rows, err = h.Repo.GetHistoricalDataBySymbol(symbol)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	labels := make([]string, len(rows))
	closeValues := make([]float64, len(rows))
	volumePerTrade := make([]int64, len(rows))
	volume := make([]int64, len(rows))
	noOfTrades := make([]int64, len(rows))
	adjClose := make([]float64, len(rows))
	deliveryPercentage := make([]float32, len(rows))
	for i, r := range rows {
		labels[i] = r.Date.Format("02 Jan 06")
		closeValues[i] = r.C
		volumePerTrade[i] = r.VolumePerTrade
		volume[i] = r.Volume
		noOfTrades[i] = r.NoOfTrades
		adjClose[i] = r.AdjustedClosePrice
		deliveryPercentage[i] = r.DeliveryPercentage
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"labels":             labels,
		"values":             closeValues,
		"volumePerTrade":     volumePerTrade,
		"volume":             volume,
		"noOfTrades":         noOfTrades,
		"adjClose":           adjClose,
		"deliveryPercentage": deliveryPercentage,
	})
}

func (h *InstrumentHandler) SyncTradeHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	symbol := r.FormValue("symbol")
	from, err := time.Parse("2006-01-02", r.FormValue("from"))
	if err != nil {
		http.Error(w, "invalid 'from' date", http.StatusBadRequest)
		return
	}
	to, err := time.Parse("2006-01-02", r.FormValue("to"))
	if err != nil {
		http.Error(w, "invalid 'to' date", http.StatusBadRequest)
		return
	}
	go func() {
		log.Printf("SyncTradeHistory started for %s", symbol)
		h.Service.SyncTradeHistory(symbol, from, to)
		log.Printf("SyncTradeHistory completed for %s", symbol)
	}()
	http.Redirect(w, r, "/instrument?symbol="+url.QueryEscape(symbol)+"&msg=sync_started", http.StatusSeeOther)
}

func (h *InstrumentHandler) ProcessDailyData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	symbol := r.FormValue("symbol")
	if symbol == "" {
		http.Error(w, "symbol required", http.StatusBadRequest)
		return
	}
	go func() {
		log.Printf("ProcessDailyData started for %s", symbol)
		h.Service.ProcessDailyData(symbol)
		log.Printf("ProcessDailyData completed for %s", symbol)
	}()
	http.Redirect(w, r, "/instrument?symbol="+url.QueryEscape(symbol)+"&msg=daily_sync_started", http.StatusSeeOther)
}

func (h *InstrumentHandler) SyncAdjClosePrice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	symbol := r.FormValue("symbol")
	if symbol == "" {
		http.Error(w, "symbol required", http.StatusBadRequest)
		return
	}
	go func() {
		log.Printf("SyncAdjClosePriceAndEvents started for %s", symbol)
		h.Service.SyncAdjClosePrice(symbol, nil, nil)
		log.Printf("SyncAdjClosePriceAndEvents completed for %s", symbol)
	}()
	http.Redirect(w, r, "/instrument?symbol="+url.QueryEscape(symbol)+"&msg=adj_sync_started", http.StatusSeeOther)
}

func (h *InstrumentHandler) SyncSplitAndDividend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	symbol := r.FormValue("symbol")
	if symbol == "" {
		http.Error(w, "symbol required", http.StatusBadRequest)
		return
	}
	go func() {
		log.Printf("SyncSplitAndDividend started for %s", symbol)
		h.Service.SyncSplitAndDividend(symbol, nil, nil)
		log.Printf("SyncSplitAndDividend completed for %s", symbol)
	}()
	http.Redirect(w, r, "/instrument?symbol="+url.QueryEscape(symbol)+"&msg=split_dividend_sync_started", http.StatusSeeOther)
}

func (h *InstrumentHandler) ShortsChartData(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		http.Error(w, "symbol required", http.StatusBadRequest)
		return
	}
	since := parsePeriodSince(r.URL.Query().Get("period"))
	var rows []repo.ShortsByDate
	var err error
	rows, err = h.Repo.GetShortsBySymbol(symbol, since)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	labels := make([]string, len(rows))
	values := make([]int64, len(rows))
	for i, r := range rows {
		labels[i] = r.Date.Format("02 Jan 06")
		values[i] = r.Quantity
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"labels": labels,
		"values": values,
	})
}

var validTags = map[string]bool{
	"":           true,
	"oversold":   true,
	"touch":      true,
	"scoop":      true,
	"overbought": true,
	"repel":      true,
	"horizontal": true,
}

func (h *InstrumentHandler) SetTag(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	symbol := r.FormValue("symbol")
	if symbol == "" {
		http.Error(w, "symbol required", http.StatusBadRequest)
		return
	}
	tag := r.FormValue("tag")
	if !validTags[tag] {
		http.Error(w, "invalid tag value", http.StatusBadRequest)
		return
	}
	if err := h.Repo.UpdateTag(symbol, tag); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if r.Header.Get("X-Requested-With") == "fetch" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"tag": tag})
		return
	}
	http.Redirect(w, r, "/instrument?symbol="+url.QueryEscape(symbol), http.StatusSeeOther)
}

func (h *InstrumentHandler) TagHistoryAPI(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		http.Error(w, "symbol required", http.StatusBadRequest)
		return
	}
	history, err := h.Repo.GetTagHistory(symbol)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type entry struct {
		Date           string   `json:"date"`
		PreviousTag    string   `json:"previous_tag"`
		NextTag        string   `json:"next_tag"`
		PriceChangePct *float64 `json:"price_change_pct"`
	}
	entries := make([]entry, len(history))
	var prevPrice float64
	for i, th := range history {
		e := entry{
			Date:        th.UpdatedOn.Format("02 Jan 2006"),
			PreviousTag: th.PreviousTag,
			NextTag:     th.NextTag,
		}
		curPrice, err := h.Repo.GetClosestAdjustedPrice(symbol, th.UpdatedOn)
		if err == nil && curPrice > 0 {
			if i > 0 && prevPrice > 0 {
				pct := (curPrice - prevPrice) / prevPrice * 100
				e.PriceChangePct = &pct
			}
			prevPrice = curPrice
		}
		entries[i] = e
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}
