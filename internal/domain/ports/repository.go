package ports

import (
	"context"
	"time"

	"exchange-rate-service/internal/domain/model"
)

type RateRepository interface {
	FetchLatestRate(ctx context.Context, pair model.CurrencyPair) (*model.ExchangeRate, error)
	FetchHistoricalRate(ctx context.Context, pair model.CurrencyPair, date time.Time) (*model.ExchangeRate, error)
	FetchHistoricalRates(ctx context.Context, request model.HistoricalRateRequest) (*model.HistoricalRates, error)
	RefreshRates(ctx context.Context) error
}
