package handler

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"ninja-trader/internal/kite"
	models "ninja-trader/internal/model"
	repo "ninja-trader/internal/repository"
	"strconv"
)

type KiteAlertHandler struct {
	KiteClient *kite.Client
	Repo       *repo.KiteAlertRepo
	Tmpl       *template.Template
}

func (h *KiteAlertHandler) ListPage(w http.ResponseWriter, r *http.Request) {
	alerts, err := h.Repo.GetAll()
	if err != nil {
		log.Printf("Error fetching kite alerts: %v", err)
	}
	h.Tmpl.ExecuteTemplate(w, "kite_alerts.html", struct {
		Alerts []models.KiteAlert
	}{alerts})
}

func (h *KiteAlertHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	symbol := r.FormValue("symbol")
	exchange := r.FormValue("exchange")
	if exchange == "" {
		exchange = "NSE"
	}
	operator := r.FormValue("operator")
	valueStr := r.FormValue("value")

	if symbol == "" || operator == "" || valueStr == "" {
		http.Error(w, "symbol, operator, and value are required", http.StatusBadRequest)
		return
	}

	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		http.Error(w, "invalid value", http.StatusBadRequest)
		return
	}

	alertID, err := h.KiteClient.CreateAlert(exchange, symbol, operator, value)
	if err != nil {
		log.Printf("Error creating kite alert: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.Repo.Create(&models.KiteAlert{
		AlertID:       alertID,
		TradingSymbol: symbol,
		Exchange:      exchange,
		Operator:      operator,
		TriggerValue:  value,
		Status:        "active",
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "alert_id": alertID})
}

func (h *KiteAlertHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	alertID := r.FormValue("alert_id")
	if alertID == "" {
		http.Error(w, "alert_id required", http.StatusBadRequest)
		return
	}

	if err := h.KiteClient.DeleteAlert(alertID); err != nil {
		log.Printf("Error deleting kite alert %s: %v — marking as deleted locally", alertID, err)
	}

	h.Repo.UpdateStatus(alertID, "deleted")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (h *KiteAlertHandler) Sync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	alerts, err := h.KiteClient.GetAlerts()
	if err != nil {
		log.Printf("Error fetching kite alerts: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

  for _, a := range alerts {
    h.Repo.Upsert(&models.KiteAlert{
      AlertID:       a.UUID,
      TradingSymbol: a.LHSTradingSymbol,
      Exchange:      a.LHSExchange,
      Operator:      a.Operator,
      TriggerValue:  a.RHSConstant,
      Status:        a.Status,
    })
  }

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "count": len(alerts)})
}

func (h *KiteAlertHandler) ListAPI(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	var alerts []models.KiteAlert
	var err error
	if symbol != "" {
		alerts, err = h.Repo.GetBySymbol(symbol)
	} else {
		alerts, err = h.Repo.GetAll()
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}
