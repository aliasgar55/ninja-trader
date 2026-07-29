package handler

import (
	"context"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	models "ninja-trader/internal/model"
	"ninja-trader/internal/service"
	"ninja-trader/internal/ticker"
	"sync"
	"time"

	"gorm.io/gorm"
)

type AdminHandler struct {
	Service *service.InstrumentService
	Ticker  *ticker.Service
	Db      *gorm.DB
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

func (h *AdminHandler) TickerStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"running": h.Ticker.IsRunning()})
}

func (h *AdminHandler) TickerToggle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.Ticker.IsRunning() {
		h.Ticker.Stop()
		h.setSetting("ticker_enabled", "false")
	} else {
		h.Ticker.Start(context.Background())
		h.setSetting("ticker_enabled", "true")
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"running": h.Ticker.IsRunning()})
}

func (h *AdminHandler) setSetting(key, value string) {
	var setting models.AppSetting
	result := h.Db.Where("key = ?", key).First(&setting)
	if result.Error != nil {
		h.Db.Create(&models.AppSetting{Key: key, Value: value})
		return
	}
	h.Db.Model(&setting).Update("value", value)
}

func (h *AdminHandler) GetSetting(key string) string {
	var setting models.AppSetting
	if err := h.Db.Where("key = ?", key).First(&setting).Error; err != nil {
		return ""
	}
	return setting.Value
}

func (h *AdminHandler) SaveLastURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	url := r.FormValue("url")
	if url != "" {
		h.setSetting("last_url", url)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) GetLastURL(w http.ResponseWriter, r *http.Request) {
	url := h.GetSetting("last_url")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": url})
}
