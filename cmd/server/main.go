package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"exchange-rate-service/internal/adapter/cache"
	httpRouter "exchange-rate-service/internal/adapter/http"
	"exchange-rate-service/internal/adapter/repository"
	"exchange-rate-service/internal/config"
	"exchange-rate-service/internal/metrics"
	"exchange-rate-service/internal/service"
	"exchange-rate-service/pkg/logger"
	
	_ "github.com/prometheus/client_golang/prometheus"
)

func main() {
	log := logger.NewLogger(os.Getenv("LOG_LEVEL"))
	log.Info("Starting exchange rate service")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	appMetrics := metrics.NewMetrics()
	rateCache := cache.NewMemoryCache(cfg.Cache.TTL, log)

	rateRepo := repository.NewExchangeAPI(
		cfg.ExchangeAPI.BaseURL,
		cfg.ExchangeAPI.APIKey,
		cfg.ExchangeAPI.Timeout,
		log,
	)

	exchangeService := service.NewExchangeService(rateRepo, rateCache, log)
	handler := httpRouter.NewHandler(exchangeService, log, appMetrics)

	router := httpRouter.NewRouter(handler, log, appMetrics)
	routes := router.SetupRoutes()

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      routes,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	ctx, cancelRefresh := context.WithCancel(context.Background())
	go refreshRates(ctx, exchangeService, cfg.ExchangeAPI.RefreshRate, log)

	go func() {
		log.Info("Starting HTTP server", "port", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down server...")

	cancelRefresh()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	log.Info("Server exited")
}

// refreshRates periodically refreshes exchange rates
func refreshRates(ctx context.Context, service *service.ExchangeService, interval time.Duration, log *logger.Logger) {
	// Refresh rates immediately at startup
	if err := service.RefreshRates(ctx); err != nil {
		log.Error("Failed to refresh rates at startup", "error", err)
	}

	// Create ticker for periodic refresh
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := service.RefreshRates(ctx); err != nil {
				log.Error("Failed to refresh rates", "error", err)
			}
		case <-ctx.Done():
			log.Info("Stopping rate refresh goroutine")
			return
		}
	}
}
