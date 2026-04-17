package handler

import (
	"html/template"
	"log"
	"net/http"
	"ninja-trader/internal/service"
	"sync"
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

func (h *AdminHandler) RenameSymbol(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	oldSymbol := r.FormValue("old_symbol")
	newSymbol := r.FormValue("new_symbol")
	if oldSymbol == "" || newSymbol == "" {
		http.Redirect(w, r, "/admin?msg=rename_missing_fields", http.StatusSeeOther)
		return
	}
	if err := h.Service.RenameSymbol(oldSymbol, newSymbol); err != nil {
		log.Printf("RenameSymbol error: %v", err)
		http.Redirect(w, r, "/admin?msg=rename_failed", http.StatusSeeOther)
		return
	}
	log.Printf("Renamed symbol %s -> %s", oldSymbol, newSymbol)
	http.Redirect(w, r, "/admin?msg=symbol_renamed", http.StatusSeeOther)
}

func (h *AdminHandler) ComputeSignals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	go func() {
		log.Println("ComputeSignals backfill started")
		instruments, err := h.Service.InstruRepo.GetAllInstruments()
		if err != nil {
			log.Printf("ComputeSignals error fetching instruments: %v", err)
			return
		}
		workers := 5
		ch := make(chan string, len(instruments))
		var wg sync.WaitGroup
		for range workers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for symbol := range ch {
					if err := h.Service.ComputeSignals(symbol); err != nil {
						log.Printf("ComputeSignals error for %s: %v", symbol, err)
					}
				}
			}()
		}
		for _, inst := range instruments {
			ch <- inst.TradingSymbol
		}
		close(ch)
		wg.Wait()
		log.Println("ComputeSignals backfill completed")
	}()
	http.Redirect(w, r, "/admin?msg=compute_signals_started", http.StatusSeeOther)
}
