package ports

import (
	"context"
	"time"

	"exchange-rate-service/internal/domain/model"
)

type RateCache interface {
	Get(ctx context.Context, pair model.CurrencyPair, date time.Time) (*model.ExchangeRate, bool)
	Set(ctx context.Context, rate *model.ExchangeRate) error
	ClearExpired(ctx context.Context) error
}
