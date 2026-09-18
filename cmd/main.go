package main

import (
  "context"
  "html/template"
  "log"
	"net/http"
	"ninja-trader/internal/database"
	"ninja-trader/internal/handler"
	"ninja-trader/internal/kite"
	models "ninja-trader/internal/model"
	repo "ninja-trader/internal/repository"
	"ninja-trader/internal/service"
	"ninja-trader/internal/ticker"
	"runtime"

	"os"
	"runtime/pprof"

	"github.com/joho/godotenv"
)

func main() {
  log.Println("NumCPU:", runtime.NumCPU())
  log.Println("GOMAXPROCS:", runtime.GOMAXPROCS(0))
	err := godotenv.Load(".env")

	if os.Getenv("PROF") == "1" {
		f, err := os.Create("cpu.prof")
		if err != nil {
			log.Fatal(err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal(err)
		}

		defer func() {
			pprof.StopCPUProfile()
			f.Close()
			fMem, err := os.Create("mem.prof")
			if err == nil {
				pprof.WriteHeapProfile(fMem)
				fMem.Close()
			}
		}()
	}

	if err != nil {
		log.Fatalf("Error loading .env file, %s", err)
	}

	db, err := database.NewDatabase(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Error setting up database, %s", err)
	}
	db.AutoMigrate(
		&models.Instrument{},
		&models.Shorts{},
		&models.Historicaldata{},
		&models.Event{},
		&models.PaperTrade{},
		&models.PaperTradeLog{},
		&models.GTTOrder{},
		&models.TagHistory{},
		&models.AppSetting{},
		&models.Note{},
		&models.AlertLog{},
		&models.KiteAlert{},
		&models.InsiderTradeTransaction{},
		&models.InsiderTradeEntity{},
		&models.BulkBlockDeal{},
	)

	instrumentRepo := repo.InstrumentRepo{Db: db}
	tradeRepo := repo.TradeRepo{Db: db}
	insideTradesRepo := repo.InsiderTradesRepo{Db: db}

	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}
	tmpl := template.Must(template.New("").Funcs(funcMap).ParseGlob("web/templates/*.html"))

	kiteClient := kite.New(os.Getenv("KITE_API_KEY"))
	if token := os.Getenv("KITE_ACCESS_TOKEN"); token != "" {
		kiteClient.SetAccessToken(token)
		log.Println("Using KITE_ACCESS_TOKEN from env")
	} else {
		log.Println("No KITE_ACCESS_TOKEN set — use /auth/login for Kite OAuth")
	}
	tickerService := ticker.New(kiteClient, instrumentRepo)
	// Check if ticker was enabled in settings
	var tickerSetting models.AppSetting
	if err := db.Where("key = ?", "ticker_enabled").First(&tickerSetting).Error; err == nil && tickerSetting.Value == "false" {
		log.Println("[ticker] disabled by admin setting, not starting")
	} else {
		tickerService.Start(context.Background())
	}
	defer tickerService.Stop()

	instrumentService := &service.InstrumentService{InstruRepo: instrumentRepo}
	tradeService := &service.TradeService{TradeRepo: tradeRepo, InstruRepo: instrumentRepo}
	insideTradesService := &service.InsiderTradesService{InsiderTradeRepo: insideTradesRepo}
	bbService := service.NewBBService(repo.BulkBlockDealRepo{Db: db})

	instrumentHandler := &handler.InstrumentHandler{Repo: instrumentRepo, TradeRepo: tradeRepo, Service: instrumentService, Tmpl: tmpl}
	insiderTradesHandler := &handler.InsiderTradesHandler{Service: insideTradesService, Tmpl: tmpl}
	tradeHandler := &handler.TradeHandler{TradeService: tradeService, Tmpl: tmpl}
	shortsHandler := &handler.ShortsHandler{Repo: instrumentRepo, Tmpl: tmpl}
	authHandler := &handler.AuthHandler{
		KiteClient: kiteClient,
		APISecret:  os.Getenv("KITE_API_SECRET"),
		Db:         db,
	}
	authHandler.LoadSession()
	adminHandler := &handler.AdminHandler{Service: instrumentService, Ticker: tickerService, InsiderTradeService: insideTradesService, BBService: bbService, Db: db, Tmpl: tmpl}
	gttRepo := &repo.GTTRepo{Db: db}
	gttHandler := &handler.GTTHandler{KiteClient: kiteClient, GTTRepo: gttRepo, Tmpl: tmpl}
	kiteAlertRepo := &repo.KiteAlertRepo{Db: db}
	kiteAlertHandler := &handler.KiteAlertHandler{KiteClient: kiteClient, Repo: kiteAlertRepo, Tmpl: tmpl}
	breadthHandler := &handler.BreadthHandler{Ticker: tickerService, Tmpl: tmpl}

	http.HandleFunc("/trade", tradeHandler.Trade)
	http.HandleFunc("/trades", tradeHandler.TradesPage)
	http.HandleFunc("/api/trade/logs", tradeHandler.TradeLogs)
	http.HandleFunc("/", instrumentHandler.List)
	http.HandleFunc("/watchlist", instrumentHandler.WatchlistPage)
	http.HandleFunc("/instrument", instrumentHandler.Detail)
	http.HandleFunc("/api/instrument/chart", instrumentHandler.ChartData)
	http.HandleFunc("/api/instrument/shorts", instrumentHandler.ShortsChartData)
	http.HandleFunc("/instrument/sync-history", instrumentHandler.SyncTradeHistory)
	http.HandleFunc("/instrument/sync-adj-close", instrumentHandler.SyncAdjClosePrice)
	http.HandleFunc("/instrument/sync-split-dividend", instrumentHandler.SyncSplitAndDividend)
	http.HandleFunc("/instrument/process-daily", instrumentHandler.ProcessDailyData)
	http.HandleFunc("/instrument/adjust-price", instrumentHandler.AdjustPrice)
	http.HandleFunc("/instrument/watchlist", instrumentHandler.ToggleWatchlist)
	http.HandleFunc("/instrument/tag", instrumentHandler.SetTag)
	http.HandleFunc("/api/instrument/tag-history", instrumentHandler.TagHistoryAPI)
	http.HandleFunc("/api/instrument/notes", instrumentHandler.NotesAPI)
	http.HandleFunc("/instrument/notes/create", instrumentHandler.CreateNote)
	http.HandleFunc("/instrument/notes/delete", instrumentHandler.DeleteNote)
	http.HandleFunc("/api/instrument/insider-trades", insiderTradesHandler.API)
	http.HandleFunc("/insider-trades", insiderTradesHandler.Page)
	http.HandleFunc("/shorts", shortsHandler.Page)
	http.HandleFunc("/api/shorts/chart", shortsHandler.ChartData)
	http.HandleFunc("/api/shorts/total", shortsHandler.TotalChartData)
	http.HandleFunc("/admin", adminHandler.Page)
	http.HandleFunc("/admin/sync-instruments", adminHandler.SyncInstruments)
	http.HandleFunc("/admin/sync-shorts", adminHandler.SyncShorts)
	http.HandleFunc("/admin/sync-daily", adminHandler.SyncDailyData)
	http.HandleFunc("/admin/rename-symbol", adminHandler.RenameSymbol)
	http.HandleFunc("/admin/compute-signals", adminHandler.ComputeSignals)
	http.HandleFunc("/admin/sync-insider-trades", adminHandler.SyncInsiderTrades)
	http.HandleFunc("/admin/sync-bulk-block-deals", adminHandler.SyncBulkBlockDeals)
	http.HandleFunc("/api/ticker/status", adminHandler.TickerStatus)
	http.HandleFunc("/api/ticker/toggle", adminHandler.TickerToggle)
	http.HandleFunc("/api/last-url", adminHandler.GetLastURL)
	http.HandleFunc("/api/last-url/save", adminHandler.SaveLastURL)
	http.HandleFunc("/gtt", gttHandler.ListPage)
	http.HandleFunc("/gtt/place", gttHandler.Place)
	http.HandleFunc("/gtt/delete", gttHandler.Delete)
	http.HandleFunc("/api/gtt/list", gttHandler.ListAPI)
	http.HandleFunc("/gtt/sync", gttHandler.Sync)
	http.HandleFunc("/kite-alerts", kiteAlertHandler.ListPage)
	http.HandleFunc("/kite-alerts/create", kiteAlertHandler.Create)
	http.HandleFunc("/kite-alerts/delete", kiteAlertHandler.Delete)
	http.HandleFunc("/kite-alerts/sync", kiteAlertHandler.Sync)
	http.HandleFunc("/api/kite-alerts", kiteAlertHandler.ListAPI)
	http.HandleFunc("/auth/login", authHandler.Login)
	http.HandleFunc("/auth/callback", authHandler.Callback)
	http.HandleFunc("/auth/logout", authHandler.Logout)
	http.HandleFunc("/api/auth/status", authHandler.Status)
	http.HandleFunc("/breadth", breadthHandler.Page)
	http.HandleFunc("/api/breadth", breadthHandler.DataAPI)
	http.HandleFunc("/alerts", breadthHandler.AlertsPage)
	http.HandleFunc("/api/alerts", breadthHandler.AlertsAPI)
	http.HandleFunc("/range", instrumentHandler.RangePage)
	http.HandleFunc("/notes", instrumentHandler.NotesPage)

	addr := ":6969"
	log.Printf("UI available at http://localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
