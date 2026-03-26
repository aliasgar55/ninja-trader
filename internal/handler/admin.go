package handler

import (
	"html/template"
	"log"
	"net/http"
	"ninja-trader/internal/service"
	"time"
)

type AdminHandler struct {
	Service *service.InstrumentService
	Tmpl    *template.Template
}

func (h *AdminHandler) Page(w http.ResponseWriter, r *http.Request) {
	data := struct{ Msg string }{Msg: r.URL.Query().Get("msg")}
	h.Tmpl.ExecuteTemplate(w, "admin.html", data)
}

func (h *AdminHandler) SyncInstruments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	go func() {
		if err := h.Service.SyncInstruments(); err != nil {
			log.Printf("SyncInstruments error: %v", err)
		} else {
			log.Println("SyncInstruments completed")
		}
	}()
	http.Redirect(w, r, "/admin?msg=sync_instruments_started", http.StatusSeeOther)
}

func (h *AdminHandler) SyncDailyData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	go func() {
		log.Println("SyncDailyData started")
		h.Service.SyncDailyData()
		log.Println("SyncDailyData completed")
	}()
	http.Redirect(w, r, "/admin?msg=sync_daily_started", http.StatusSeeOther)
}

func (h *AdminHandler) SyncShorts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
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
		if err := h.Service.SyncShorts(from, to); err != nil {
			log.Printf("SyncShorts error: %v", err)
		} else {
			log.Println("SyncShorts completed")
		}
	}()
	http.Redirect(w, r, "/admin?msg=sync_shorts_started", http.StatusSeeOther)
}
