package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"ninja-trader/internal/database"
	"ninja-trader/internal/handler"
	"ninja-trader/internal/model"
	repo "ninja-trader/internal/repository"
	"ninja-trader/internal/service"
	"runtime"

	"os"
	"runtime/pprof"

	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("NumCPU:", runtime.NumCPU())
	fmt.Println("GOMAXPROCS:", runtime.GOMAXPROCS(0))
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
	db.AutoMigrate(&models.Instrument{}, &models.Shorts{}, &models.Historicaldata{})

	instrumentRepo := repo.InstrumentRepo{Db: db}
	instrumentService := &service.InstrumentService{InstruRepo: instrumentRepo}

	tmpl := template.Must(template.ParseGlob("web/templates/*.html"))

	instrumentHandler := &handler.InstrumentHandler{Repo: instrumentRepo, Service: instrumentService, Tmpl: tmpl}
	adminHandler := &handler.AdminHandler{Service: instrumentService, Tmpl: tmpl}
	shortsHandler := &handler.ShortsHandler{Repo: instrumentRepo, Tmpl: tmpl}

	http.HandleFunc("/", instrumentHandler.List)
	http.HandleFunc("/watchlist", instrumentHandler.WatchlistPage)
	http.HandleFunc("/instrument", instrumentHandler.Detail)
	http.HandleFunc("/api/instrument/chart", instrumentHandler.ChartData)
	http.HandleFunc("/api/instrument/shorts", instrumentHandler.ShortsChartData)
	http.HandleFunc("/instrument/sync-history", instrumentHandler.SyncTradeHistory)
	http.HandleFunc("/instrument/sync-adj-close", instrumentHandler.SyncAdjClosePriceAndEvents)
	http.HandleFunc("/instrument/process-daily", instrumentHandler.ProcessDailyData)
	http.HandleFunc("/instrument/watchlist", instrumentHandler.ToggleWatchlist)
	http.HandleFunc("/shorts", shortsHandler.Page)
	http.HandleFunc("/api/shorts/chart", shortsHandler.ChartData)
	http.HandleFunc("/api/shorts/total", shortsHandler.TotalChartData)
	http.HandleFunc("/admin", adminHandler.Page)
	http.HandleFunc("/admin/sync-instruments", adminHandler.SyncInstruments)
	http.HandleFunc("/admin/sync-shorts", adminHandler.SyncShorts)
	http.HandleFunc("/admin/sync-daily", adminHandler.SyncDailyData)

	addr := ":6969"
	log.Printf("UI available at http://localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
