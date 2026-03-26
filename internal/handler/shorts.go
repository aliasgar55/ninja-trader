package handler

import (
	"encoding/json"
	"html/template"
	"net/http"
	repo "ninja-trader/internal/repository"
	"strconv"
	"time"
)

type ShortsHandler struct {
	Repo repo.InstrumentRepo
	Tmpl *template.Template
}

type symbolEntry struct {
	Symbol   string
	Quantity int64
}

type dateGroup struct {
	Date    string
	Symbols []symbolEntry
}

func parseSince(r *http.Request) *time.Time {
	d := r.URL.Query().Get("days")
	if d == "" || d == "all" {
		return nil
	}
	days, err := strconv.Atoi(d)
	if err != nil {
		return nil
	}
	t := time.Now().AddDate(0, 0, -days)
	return &t
}

func (h *ShortsHandler) Page(w http.ResponseWriter, r *http.Request) {
	year := r.URL.Query().Get("year")
	var rows []repo.ShortSymbolByDate
	if year != "" {
		rows, _ = h.Repo.GetShortSymbolsGroupedByDateAndYear(year)
	} else {
		since := parseSince(r)
		rows, _ = h.Repo.GetShortSymbolsGroupedByDate(since)
	}
	selected := r.URL.Query().Get("symbol")
	selectedDate := r.URL.Query().Get("date")
	days := r.URL.Query().Get("days")
	years, _ := h.Repo.GetShortsYears()

	var groups []dateGroup
	var current *dateGroup
	for _, row := range rows {
		d := row.Date.Format("Mon 02 Jan 06")
		if current == nil || current.Date != d {
			groups = append(groups, dateGroup{Date: d})
			current = &groups[len(groups)-1]
		}
		current.Symbols = append(current.Symbols, symbolEntry{Symbol: row.TradingSymbol, Quantity: row.Quantity})
	}

	data := struct {
		Groups       []dateGroup
		Selected     string
		SelectedDate string
		Days         string
		Years        []int
		Year         string
	}{Groups: groups, Selected: selected, SelectedDate: selectedDate, Days: days, Years: years, Year: year}
	h.Tmpl.ExecuteTemplate(w, "shorts.html", data)
}

func (h *ShortsHandler) TotalChartData(w http.ResponseWriter, r *http.Request) {
	year := r.URL.Query().Get("year")
	var rows []repo.TotalShortsByDate
	var err error
	if year != "" {
		rows, err = h.Repo.GetTotalShortsByDateAndYear(year)
	} else {
		since := parseSince(r)
		rows, err = h.Repo.GetTotalShortsByDate(since)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	labels := make([]string, len(rows))
	quantities := make([]int64, len(rows))
	symbolCounts := make([]int64, len(rows))
	for i, r := range rows {
		labels[i] = r.Date.Format("Mon 02 Jan 06")
		quantities[i] = r.Quantity
		symbolCounts[i] = r.SymbolCount
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"labels":       labels,
		"values":       quantities,
		"symbolCounts": symbolCounts,
	})
}

func (h *ShortsHandler) ChartData(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		http.Error(w, "symbol required", http.StatusBadRequest)
		return
	}
	year := r.URL.Query().Get("year")
	var rows []repo.ShortsByDate
	var err error
	if year != "" {
		rows, err = h.Repo.GetShortsBySymbolAndYear(symbol, year)
	} else {
		since := parseSince(r)
		rows, err = h.Repo.GetShortsBySymbol(symbol, since)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	labels := make([]string, len(rows))
	values := make([]int64, len(rows))
	for i, r := range rows {
		labels[i] = r.Date.Format("Mon 02 Jan 06")
		values[i] = r.Quantity
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"labels": labels,
		"values": values,
	})
}
