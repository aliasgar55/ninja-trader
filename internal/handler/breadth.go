package handler

import (
	"encoding/json"
	"html/template"
	"net/http"
	"ninja-trader/internal/ticker"
	"time"
)

type BreadthHandler struct {
	Ticker *ticker.Service
	Tmpl   *template.Template
}

func (h *BreadthHandler) Page(w http.ResponseWriter, r *http.Request) {
	h.Tmpl.ExecuteTemplate(w, "breadth.html", nil)
}

func (h *BreadthHandler) DataAPI(w http.ResponseWriter, r *http.Request) {
	history := h.Ticker.GetBreadthHistory()
	labels := make([]string, len(history))
	values := make([]int64, len(history))
	for i, s := range history {
		labels[i] = s.Time.Format("15:04:05")
		values[i] = s.Count
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"labels":  labels,
		"values":  values,
		"current": h.Ticker.GetBreadthCount(),
	})
}

func (h *BreadthHandler) AlertsPage(w http.ResponseWriter, r *http.Request) {
	h.Tmpl.ExecuteTemplate(w, "alerts.html", nil)
}

func (h *BreadthHandler) AlertsAPI(w http.ResponseWriter, r *http.Request) {
	dateStr := r.URL.Query().Get("date")
	var day time.Time
	if dateStr != "" {
		parsed, err := time.Parse("2006-01-02", dateStr)
		if err == nil {
			day = parsed
		} else {
			day = time.Now().UTC().Truncate(24 * time.Hour)
		}
	} else {
		day = time.Now().UTC().Truncate(24 * time.Hour)
	}
	history := h.Ticker.GetAlertHistoryForDate(day)
	type alertJSON struct {
		Time    string  `json:"time"`
		Type    string  `json:"type"`
		Symbol  string  `json:"symbol"`
		Price   float64 `json:"price"`
		Message string  `json:"message"`
	}
	entries := make([]alertJSON, len(history))
	for i, a := range history {
		entries[i] = alertJSON{
			Time:    a.Time.Format("15:04:05"),
			Type:    a.Type,
			Symbol:  a.Symbol,
			Price:   a.Price,
			Message: a.Message,
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}
