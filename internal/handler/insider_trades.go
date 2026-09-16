package handler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"ninja-trader/internal/service"
	"strconv"
)

type InsiderTradesHandler struct {
	Service *service.InsiderTradesService
	Tmpl    *template.Template
}

const insiderPageSize = 100

func (h *InsiderTradesHandler) Page(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * insiderPageSize

	trades, total, err := h.Service.GetAllTrades(insiderPageSize, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	totalPages := int(total) / insiderPageSize
	if int(total)%insiderPageSize > 0 {
		totalPages++
	}

	type tradeEntry struct {
		Date           string
		Symbol         string
		EntityName     string
		EntityCategory string
		Type           string
		AcqMode        string
		Quantity       string
		Value          string
	}

	type yearGroup struct {
		Year   int
		Trades []tradeEntry
	}

	var yearOrder []int
	grouped := map[int][]tradeEntry{}
	for _, t := range trades {
		year := t.TransactionDate.Year()
		if _, exists := grouped[year]; !exists {
			yearOrder = append(yearOrder, year)
		}
		qty := fmt.Sprintf("%d", t.Quantity)
		val := ""
		if t.Value > 0 {
			val = formatIndianNumber(t.Value)
		}
		grouped[year] = append(grouped[year], tradeEntry{
			Date:           t.TransactionDate.Format("02 Jan 2006"),
			Symbol:         t.Symbol,
			EntityName:     t.EntityName,
			EntityCategory: t.EntityCategory,
			Type:           t.TransactionType,
			AcqMode:        t.AcqMode,
			Quantity:       qty,
			Value:          val,
		})
	}

	years := make([]yearGroup, 0, len(yearOrder))
	for _, y := range yearOrder {
		years = append(years, yearGroup{Year: y, Trades: grouped[y]})
	}

	if err := h.Tmpl.ExecuteTemplate(w, "insider_trades.html", struct {
		Years      []yearGroup
		Page       int
		TotalPages int
		Total      int64
		HasPrev    bool
		HasNext    bool
	}{years, page, totalPages, total, page > 1, page < totalPages}); err != nil {
		log.Printf("Template error insider_trades.html: %v", err)
	}
}

func (h *InsiderTradesHandler) API(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		http.Error(w, "symbol required", http.StatusBadRequest)
		return
	}
	trades, err := h.Service.GetTradesBySymbol(symbol)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
  type entry struct {
    Date            string  `json:"date"`
    Entity          string  `json:"entity"`
    Category        string  `json:"category"`
    Type            string  `json:"type"`
    Quantity        int64   `json:"quantity"`
    Value           float64 `json:"value"`
    AcqMode         string  `json:"acqMode"`
    HoldingAfterPct float64 `json:"holdingAfterPct"`
  }
  entries := make([]entry, len(trades))
  for i, t := range trades {
    entries[i] = entry{
      Date:            t.TransactionDate.Format("02 Jan 2006"),
      Entity:          t.EntityName,
      Category:        t.EntityCategory,
      Type:            t.TransactionType,
      Quantity:        t.Quantity,
      Value:           t.Value,
      AcqMode:         t.AcqMode,
      HoldingAfterPct: t.HoldingAfterPct,
    }
  }
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}
