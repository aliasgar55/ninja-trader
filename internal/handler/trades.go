package handler

import (
	"encoding/json"
	"html/template"
	"net/http"
	models "ninja-trader/internal/model"
	repo "ninja-trader/internal/repository"
	"ninja-trader/internal/service"
	"strconv"
)

type TradeHandler struct {
	TradeService *service.TradeService
	Tmpl         *template.Template
}

func (h *TradeHandler) TradesPage(w http.ResponseWriter, r *http.Request) {
	trades, err := h.TradeService.GetAllTradesWithPnL()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var totalInvested, totalCurrentValue, totalPnL float64
	for _, t := range trades {
		totalInvested += t.InvestedAmount
		totalCurrentValue += t.CurrentValue
		totalPnL += t.PnL
	}
	totalPnLPct := 0.0
	if totalInvested > 0 {
		totalPnLPct = (totalPnL / totalInvested) * 100
	}
	h.Tmpl.ExecuteTemplate(w, "trades.html", struct {
		Trades            []repo.TradeWithPnL
		TotalInvested     float64
		TotalCurrentValue float64
		TotalPnL          float64
		TotalPnLPct       float64
	}{trades, totalInvested, totalCurrentValue, totalPnL, totalPnLPct})
}

func (h *TradeHandler) TradeLogs(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		http.Error(w, "symbol is required", http.StatusBadRequest)
		return
	}
	logs, err := h.TradeService.GetTradeLogsBySymbol(symbol)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

func (h *TradeHandler) Trade(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	symbol := r.FormValue("symbol")
	qtyStr := r.FormValue("quantity")
	action := r.FormValue("action")
	if symbol == "" || qtyStr == "" || action == "" {
		http.Error(w, "symbol, quantity and action are required", http.StatusBadRequest)
		return
	}
	qty, err := strconv.ParseUint(qtyStr, 10, 64)
	if err != nil || qty < 1 {
		http.Error(w, "invalid quantity", http.StatusBadRequest)
		return
	}
	tradeType := models.Buy
	if action == "SELL" {
		tradeType = models.Sell
	}
	if err := h.TradeService.Trade(symbol, uint(qty), tradeType); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
