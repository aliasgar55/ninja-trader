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

	kiteconnect "github.com/zerodha/gokiteconnect/v4"
)

type GTTHandler struct {
	KiteClient *kite.Client
	GTTRepo    *repo.GTTRepo
	Tmpl       *template.Template
}

func (h *GTTHandler) ListPage(w http.ResponseWriter, r *http.Request) {
	orders, err := h.GTTRepo.GetAllWithPrice()
	if err != nil {
		log.Printf("Error fetching GTTs from DB: %v", err)
		h.Tmpl.ExecuteTemplate(w, "gtt.html", struct {
			GTTs  []repo.GTTOrderWithPrice
			Error string
		}{nil, err.Error()})
		return
	}
	h.Tmpl.ExecuteTemplate(w, "gtt.html", struct {
		GTTs  []repo.GTTOrderWithPrice
		Error string
	}{orders, ""})
}

func (h *GTTHandler) ListAPI(w http.ResponseWriter, r *http.Request) {
	symbolFilter := r.URL.Query().Get("symbol")
	var orders []models.GTTOrder
	var err error
	if symbolFilter != "" {
		orders, err = h.GTTRepo.GetBySymbol(symbolFilter)
	} else {
		orders, err = h.GTTRepo.GetAll()
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

func (h *GTTHandler) Place(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	symbol := r.FormValue("symbol")
	exchange := r.FormValue("exchange")
	if exchange == "" {
		exchange = "NSE"
	}
	transactionType := r.FormValue("transaction_type")
	triggerType := r.FormValue("trigger_type")
	lastPriceStr := r.FormValue("last_price")
	triggerPriceStr := r.FormValue("trigger_price")
	limitPriceStr := r.FormValue("limit_price")
	quantityStr := r.FormValue("quantity")

	if symbol == "" || transactionType == "" || triggerType == "" || lastPriceStr == "" || triggerPriceStr == "" || limitPriceStr == "" || quantityStr == "" {
		http.Error(w, "all fields are required", http.StatusBadRequest)
		return
	}

	lastPrice, err := strconv.ParseFloat(lastPriceStr, 64)
	if err != nil {
		http.Error(w, "invalid last_price", http.StatusBadRequest)
		return
	}
	triggerPrice, err := strconv.ParseFloat(triggerPriceStr, 64)
	if err != nil {
		http.Error(w, "invalid trigger_price", http.StatusBadRequest)
		return
	}
	limitPrice, err := strconv.ParseFloat(limitPriceStr, 64)
	if err != nil {
		http.Error(w, "invalid limit_price", http.StatusBadRequest)
		return
	}
	quantity, err := strconv.ParseFloat(quantityStr, 64)
	if err != nil || quantity < 1 {
		http.Error(w, "invalid quantity", http.StatusBadRequest)
		return
	}

	var upperTrigger, upperLimit, upperQty float64

	params := kiteconnect.GTTParams{
		Tradingsymbol:   symbol,
		Exchange:        exchange,
		LastPrice:       lastPrice,
		TransactionType: transactionType,
	}

	if triggerType == "oco" {
		upperTriggerStr := r.FormValue("upper_trigger_price")
		upperLimitStr := r.FormValue("upper_limit_price")
		upperQtyStr := r.FormValue("upper_quantity")

		if upperTriggerStr == "" || upperLimitStr == "" || upperQtyStr == "" {
			http.Error(w, "OCO requires upper trigger, limit price and quantity", http.StatusBadRequest)
			return
		}

		upperTrigger, err = strconv.ParseFloat(upperTriggerStr, 64)
		if err != nil {
			http.Error(w, "invalid upper_trigger_price", http.StatusBadRequest)
			return
		}
		upperLimit, err = strconv.ParseFloat(upperLimitStr, 64)
		if err != nil {
			http.Error(w, "invalid upper_limit_price", http.StatusBadRequest)
			return
		}
		upperQty, err = strconv.ParseFloat(upperQtyStr, 64)
		if err != nil || upperQty < 1 {
			http.Error(w, "invalid upper_quantity", http.StatusBadRequest)
			return
		}

		params.Trigger = &kiteconnect.GTTOneCancelsOtherTrigger{
			Lower: kiteconnect.TriggerParams{
				TriggerValue: triggerPrice,
				LimitPrice:   limitPrice,
				Quantity:     quantity,
			},
			Upper: kiteconnect.TriggerParams{
				TriggerValue: upperTrigger,
				LimitPrice:   upperLimit,
				Quantity:     upperQty,
			},
		}
	} else {
		params.Trigger = &kiteconnect.GTTSingleLegTrigger{
			TriggerParams: kiteconnect.TriggerParams{
				TriggerValue: triggerPrice,
				LimitPrice:   limitPrice,
				Quantity:     quantity,
			},
		}
	}

	resp, err := h.KiteClient.PlaceGTT(params)
	if err != nil {
		log.Printf("Error placing GTT: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Save to local DB
	order := &models.GTTOrder{
		TriggerID:         resp.TriggerID,
		TradingSymbol:     symbol,
		Exchange:          exchange,
		TransactionType:   transactionType,
		TriggerType:       triggerType,
		Status:            "active",
		LastPrice:         lastPrice,
		TriggerPrice:      triggerPrice,
		LimitPrice:        limitPrice,
		Quantity:          quantity,
		UpperTriggerPrice: upperTrigger,
		UpperLimitPrice:   upperLimit,
		UpperQuantity:     upperQty,
	}
	if err := h.GTTRepo.Create(order); err != nil {
		log.Printf("GTT placed on Kite (ID=%d) but failed to save locally: %v", resp.TriggerID, err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *GTTHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	triggerIDStr := r.FormValue("trigger_id")
	if triggerIDStr == "" {
		http.Error(w, "trigger_id is required", http.StatusBadRequest)
		return
	}

	triggerID, err := strconv.Atoi(triggerIDStr)
	if err != nil {
		http.Error(w, "invalid trigger_id", http.StatusBadRequest)
		return
	}

	_, err = h.KiteClient.DeleteGTT(triggerID)
	if err != nil {
		log.Printf("Error deleting GTT %d: %v", triggerID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update local DB
	if err := h.GTTRepo.UpdateStatus(triggerID, "deleted"); err != nil {
		log.Printf("GTT %d deleted on Kite but failed to update locally: %v", triggerID, err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (h *GTTHandler) Sync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	gtts, err := h.KiteClient.GetGTTs()
	if err != nil {
		log.Printf("Error fetching GTTs from Kite for sync: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	synced := 0
	for _, g := range gtts {
		triggerType := "single"
		if g.Type == "two-leg" {
			triggerType = "oco"
		}

		var triggerPrice, limitPrice, qty float64
		var upperTrigger, upperLimit, upperQty float64

		if len(g.Orders) > 0 {
			triggerPrice = g.Orders[0].Price
			qty = g.Orders[0].Quantity
		}
		if len(g.Condition.TriggerValues) > 0 {
			triggerPrice = g.Condition.TriggerValues[0]
		}
		if len(g.Orders) > 0 {
			limitPrice = g.Orders[0].Price
			qty = g.Orders[0].Quantity
		}

		if triggerType == "oco" && len(g.Condition.TriggerValues) > 1 {
			upperTrigger = g.Condition.TriggerValues[1]
			if len(g.Orders) > 1 {
				upperLimit = g.Orders[1].Price
				upperQty = g.Orders[1].Quantity
			}
		}

		transactionType := ""
		if len(g.Orders) > 0 {
			transactionType = g.Orders[0].TransactionType
		}

		order := &models.GTTOrder{
			TriggerID:         g.ID,
			TradingSymbol:     g.Condition.Tradingsymbol,
			Exchange:          g.Condition.Exchange,
			TransactionType:   transactionType,
			TriggerType:       triggerType,
			Status:            g.Status,
			LastPrice:         g.Condition.LastPrice,
			TriggerPrice:      triggerPrice,
			LimitPrice:        limitPrice,
			Quantity:          qty,
			UpperTriggerPrice: upperTrigger,
			UpperLimitPrice:   upperLimit,
			UpperQuantity:     upperQty,
		}

		if err := h.GTTRepo.Upsert(order); err != nil {
			log.Printf("Error upserting GTT %d: %v", g.ID, err)
		} else {
			synced++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"synced": synced})
}
