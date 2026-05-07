package handler

import (
	"encoding/json"
	"html/template"
	"net/http"
	"ninja-trader/internal/ticker"
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
		"labels": labels,
		"values": values,
		"current": h.Ticker.GetBreadthCount(),
	})
}
