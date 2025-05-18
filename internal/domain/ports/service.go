package ports

import (
	"context"
	"time"

	"exchange-rate-service/internal/domain/model"
)

type ExchangeService interface {
	GetLatestRate(ctx context.Context, from, to model.Currency) (*model.ExchangeRate, error)
	GetHistoricalRate(ctx context.Context, from, to model.Currency, date time.Time) (*model.ExchangeRate, error)
	GetHistoricalRates(ctx context.Context, request model.HistoricalRateRequest) (*model.HistoricalRates, error)
	ConvertCurrency(ctx context.Context, request model.ConversionRequest) (*model.ConversionResult, error)
	RefreshRates(ctx context.Context) error
}
