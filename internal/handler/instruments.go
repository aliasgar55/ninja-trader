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
	date := r.URL.Query().Get("date")
	var instruments []repo.InstrumentWithHolding
	var err error
	if date != "" {
		instruments, err = h.Repo.GetAllInstrumentsWithHoldingForDate(query, sort, order, date)
	} else {
		instruments, err = h.Repo.GetAllInstrumentsWithHolding(query, sort, order)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = h.Tmpl.ExecuteTemplate(w, "instruments.html", struct {
		Instruments []repo.InstrumentWithHolding
		Query       string
		Sort        string
		Order       string
		Date        string
		Today       string
	}{instruments, query, sort, order, date, time.Now().Format("2006-01-02")})
	if err != nil {
		log.Printf("Template error instruments.html: %v", err)
	}
}

func (h *InstrumentHandler) Detail(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	sort := r.URL.Query().Get("sort")
	order := r.URL.Query().Get("order")
	date := r.URL.Query().Get("date")
	period := r.URL.Query().Get("period")
	if period == "" {
		redirectURL := "/instrument?symbol=" + url.QueryEscape(symbol) + "&period=max"
		if r.URL.Query().Get("watchlist") == "1" {
			redirectURL += "&watchlist=1"
		}
		if r.URL.Query().Get("from") == "range" {
			redirectURL += "&from=range"
		}
		if sort != "" {
			redirectURL += "&sort=" + url.QueryEscape(sort) + "&order=" + url.QueryEscape(order)
		}
		if date != "" {
			redirectURL += "&date=" + url.QueryEscape(date)
		}
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}
	if period == "max" {
		period = ""
	}
	watchlistOnly := r.URL.Query().Get("watchlist") == "1"
	fromRange := r.URL.Query().Get("from") == "range"

	// Run independent DB queries concurrently
	type adjacentResult struct{ prev, next string }
	adjCh := make(chan adjacentResult, 1)
	go func() {
		var p, n string
		if fromRange {
			p, n = h.Repo.GetAdjacentRangeSymbols(symbol, sort, order)
		} else if watchlistOnly {
			p, n = h.Repo.GetAdjacentWatchlistSymbols(symbol, sort, order)
		} else if date != "" {
			p, n = h.Repo.GetAdjacentSymbolsForDate(symbol, sort, order, date)
		} else {
			p, n = h.Repo.GetAdjacentSymbols(symbol, sort, order)
		}
		adjCh <- adjacentResult{p, n}
	}()

	fullName := ""
	watchlist := false
	marketCap := ""
	basicIndustry := ""
	index := ""
	holding := 0
	needsAdjustment := false
	isFnoSec := false
	var deliveryPct float32
	var updatedAt time.Time
	var pctFrom52WLow float64
	tag := ""
	if inst, err := h.Repo.GetInstrumentBySymbol(symbol); err == nil {
		fullName = inst.InstrumentFullName
		watchlist = inst.Watchlist
		marketCap = formatIndianNumber(inst.MarketCap)
		basicIndustry = inst.BasicIndustry
		index = inst.Index
		updatedAt = inst.UpdatedAt
		needsAdjustment = inst.NeedsAdjsutment
		isFnoSec = inst.IsFnoSec
		tag = inst.Tag
		if trade, err := h.TradeRepo.GetHoldingByInstrumentID(inst.ID); err == nil {
			holding = trade
		}
		if dp, err := h.Repo.GetLatestDeliveryPercentage(symbol); err == nil {
			deliveryPct = dp
		}
		if latestRow, err := h.Repo.GetPreviousTrade(symbol, time.Now().AddDate(0, 0, 1)); err == nil && latestRow.YearLow > 0 {
			pctFrom52WLow = (latestRow.AdjustedClosePrice - latestRow.YearLow) / latestRow.YearLow * 100
		}
	}

	adj := <-adjCh
	prev, next := adj.prev, adj.next
	tmplPeriod := period
	if period == "" {
		tmplPeriod = "max"
	}
	lastUpdated := ""
	if !updatedAt.IsZero() {
		lastUpdated = updatedAt.Format("02/01/06 15:04:05")
	}
	if err := h.Tmpl.ExecuteTemplate(w, "instrument_detail.html", struct {
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
		FromRange          bool
		Holding            int
		LastUpdated        string
		NeedsAdjustment    bool
		IsFnoSec           bool
		DeliveryPercentage float32
		PctFrom52WLow      float64
		Tag                string
		TagOptions         []string
		Sort               string
		Order              string
		Date               string
	}{symbol, fullName, marketCap, basicIndustry, index, r.URL.Query().Get("msg"), tmplPeriod, prev, next, watchlist, watchlistOnly, fromRange, holding, lastUpdated, needsAdjustment, isFnoSec, deliveryPct, pctFrom52WLow, tag, []string{"oversold", "touch", "scoop", "overbought", "repel", "horizontal", "breakout"}, sort, order, date}); err != nil {
		log.Printf("Template error instrument_detail.html: %v", err)
	}
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
	var marketCap float64
	if inst, err := h.Repo.GetInstrumentBySymbol(symbol); err == nil {
		marketCap = inst.MarketCap
	}
	labels := make([]string, len(rows))
	closeValues := make([]float64, len(rows))
	lowValues := make([]float64, len(rows))
	highValues := make([]float64, len(rows))
	volumePerTrade := make([]int64, len(rows))
	volume := make([]uint64, len(rows))
	volumeMa20 := make([]float64, len(rows))
	noOfTrades := make([]int64, len(rows))
	adjClose := make([]float64, len(rows))
	deliveryPercentage := make([]float32, len(rows))
	deliveryValue := make([]float64, len(rows))
	vptMa20 := make([]float64, len(rows))
	vptScore := make([]float64, len(rows))
  divergence := make([]float64, len(rows))
  divergenceMax3y := make([]float64, len(rows))
  intraDayVol := make([]uint64, len(rows))
  for i, r := range rows {
		labels[i] = r.Date.Format("02 Jan 06")
		closeValues[i] = r.C
		lowValues[i] = r.L
		highValues[i] = r.H
		volumePerTrade[i] = r.VolumePerTrade
		volume[i] = r.Volume
		volumeMa20[i] = r.VolumeMa20
		noOfTrades[i] = r.NoOfTrades
		adjClose[i] = r.AdjustedClosePrice
		deliveryPercentage[i] = r.DeliveryPercentage
		if marketCap > 0 {
			if r.DeliveryValue > 0 {
				deliveryValue[i] = r.DeliveryValue
			} else {
				deliveryValue[i] = float64(r.Volume) * r.C * float64(r.DeliveryPercentage) / 100.0 / (marketCap * 10000000) * 100
			}
		}
		vptMa20[i] = r.VptMa20
		vptScore[i] = r.VptScore
    divergence[i] = r.Divergence
    divergenceMax3y[i] = r.DivergenceMax3y
    intraDayVol[i] = r.EstimatedIntraDayVol
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"labels":             labels,
		"values":             closeValues,
		"low":                lowValues,
		"high":               highValues,
		"volumePerTrade":     volumePerTrade,
		"volume":             volume,
		"volumeMa20":         volumeMa20,
		"noOfTrades":         noOfTrades,
		"adjClose":           adjClose,
		"deliveryPercentage": deliveryPercentage,
		"deliveryValue":      deliveryValue,
		"vptMa20":            vptMa20,
		"vptScore":           vptScore,
    "divergence":         divergence,
    "divergenceMax3y":    divergenceMax3y,
    "intraDayVol":        intraDayVol,
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
		if err := h.Service.ProcessDailyData(symbol); err != nil {
			log.Printf("ProcessDailyData FAILED for %s: %v", symbol, err)
			return
		}
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

func (h *InstrumentHandler) AdjustPrice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	symbol := r.FormValue("symbol")
	ratioStr := r.FormValue("ratio")
	dateStr := r.FormValue("date")
	if symbol == "" || ratioStr == "" || dateStr == "" {
		http.Error(w, "symbol, ratio, and date required", http.StatusBadRequest)
		return
	}
	ratio, err := strconv.ParseFloat(ratioStr, 64)
	if err != nil || ratio <= 0 {
		http.Error(w, "invalid ratio", http.StatusBadRequest)
		return
	}
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		http.Error(w, "invalid date", http.StatusBadRequest)
		return
	}
	if err := h.Repo.AdjustPriceByRatio(symbol, date, ratio); err != nil {
		log.Printf("AdjustPrice error for %s: %v", symbol, err)
		http.Error(w, "failed to adjust prices", http.StatusInternalServerError)
		return
	}
	log.Printf("AdjustPrice: divided adjusted_close_price by %.4f for %s before %s", ratio, symbol, dateStr)
	http.Redirect(w, r, "/instrument?symbol="+url.QueryEscape(symbol)+"&msg=price_adjusted", http.StatusSeeOther)
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

func (h *InstrumentHandler) NotesAPI(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		http.Error(w, "symbol required", http.StatusBadRequest)
		return
	}
	notes, err := h.Repo.GetNotesBySymbol(symbol)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type noteEntry struct {
		ID    uint    `json:"id"`
		Date  string  `json:"date"`
		Price float64 `json:"price"`
		Text  string  `json:"text"`
	}
	entries := make([]noteEntry, len(notes))
	for i, n := range notes {
		var price float64
		history, err := h.Repo.GetPreviousTrade(symbol, n.Date)
		if err == nil {
			price = history.C
		}
		entries[i] = noteEntry{
			ID:    n.ID,
			Date:  n.Date.Format("02 Jan 2006"),
			Price: price,
			Text:  n.Text,
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

func (h *InstrumentHandler) CreateNote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	symbol := r.FormValue("symbol")
	text := r.FormValue("text")
	if symbol == "" || text == "" {
		http.Error(w, "symbol and text are required", http.StatusBadRequest)
		return
	}

	instrument, err := h.Repo.GetInstrumentBySymbol(symbol)
	if err != nil {
		http.Error(w, "instrument not found", http.StatusNotFound)
		return
	}

	note := &models.Note{
		InstrumentID:  instrument.ID,
		TradingSymbol: symbol,
		Date:          time.Now(),
		Text:          text,
	}
	if err := h.Repo.CreateNote(note); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *InstrumentHandler) DeleteNote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	idStr := r.FormValue("id")
	if idStr == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.Repo.DeleteNote(uint(id)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *InstrumentHandler) RangePage(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	sort := r.URL.Query().Get("sort")
	order := r.URL.Query().Get("order")
	instruments, err := h.Repo.GetInstrumentsWithRange(query, sort, order)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.Tmpl.ExecuteTemplate(w, "range.html", struct {
		Instruments []repo.InstrumentWithRange
		Query       string
		Sort        string
		Order       string
	}{instruments, query, sort, order})
}

func (h *InstrumentHandler) NotesPage(w http.ResponseWriter, r *http.Request) {
	notes, err := h.Repo.GetAllNotes()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type noteView struct {
		ID            uint
		TradingSymbol string
		Date          string
		Text          string
		Price         float64
		CurrentPrice  float64
		ChangePct     float64
		PctFrom52WLow float64
	}
	views := make([]noteView, len(notes))
	for i, n := range notes {
		var price, currentPrice, pctFrom52WLow float64
		if history, err := h.Repo.GetPreviousTrade(n.TradingSymbol, n.Date); err == nil {
			price = history.C
		}
		if latestRow, err := h.Repo.GetPreviousTrade(n.TradingSymbol, time.Now().AddDate(0, 0, 1)); err == nil {
			currentPrice = latestRow.AdjustedClosePrice
			if latestRow.YearLow > 0 {
				pctFrom52WLow = (latestRow.AdjustedClosePrice - latestRow.YearLow) / latestRow.YearLow * 100
			}
		}
		var changePct float64
		if price > 0 && currentPrice > 0 {
			changePct = (currentPrice - price) / price * 100
		}
		views[i] = noteView{
			ID:            n.ID,
			TradingSymbol: n.TradingSymbol,
			Date:          n.Date.Format("02 Jan 2006"),
			Text:          n.Text,
			Price:         price,
			CurrentPrice:  currentPrice,
			ChangePct:     changePct,
			PctFrom52WLow: pctFrom52WLow,
		}
	}
	// Group by symbol
	type symbolGroup struct {
		Symbol string
		Notes  []noteView
	}
	groupMap := make(map[string][]noteView)
	var groupOrder []string
	for _, v := range views {
		if _, exists := groupMap[v.TradingSymbol]; !exists {
			groupOrder = append(groupOrder, v.TradingSymbol)
		}
		groupMap[v.TradingSymbol] = append(groupMap[v.TradingSymbol], v)
	}
	groups := make([]symbolGroup, len(groupOrder))
	for i, sym := range groupOrder {
		groups[i] = symbolGroup{Symbol: sym, Notes: groupMap[sym]}
	}
	h.Tmpl.ExecuteTemplate(w, "notes.html", struct {
		Groups []symbolGroup
		Total  int
	}{groups, len(views)})
}

func (h *InstrumentHandler) BackFillIntraDayVol(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	h.Service.BackFillIntraDayVol()
	w.WriteHeader(http.StatusOK)
}
