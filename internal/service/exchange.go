package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"exchange-rate-service/internal/domain/model"
	"exchange-rate-service/internal/domain/ports"
	"exchange-rate-service/pkg/logger"
)

var (
	ErrInvalidCurrency    = errors.New("invalid currency")
	ErrDateOutOfRange     = errors.New("date is outside allowed range (older than 90 days)")
	ErrInvalidDateRange   = errors.New("invalid date range")
	ErrRateNotFound       = errors.New("exchange rate not found")
	ErrExternalAPIFailure = errors.New("external API failure")
	ErrInvalidAmount      = errors.New("invalid amount")
)

type ExchangeService struct {
	repository ports.RateRepository
	cache      ports.RateCache
	log        *logger.Logger
}

func NewExchangeService(repository ports.RateRepository, cache ports.RateCache, log *logger.Logger) *ExchangeService {
	return &ExchangeService{
		repository: repository,
		cache:      cache,
		log:        log,
	}
}

func (s *ExchangeService) GetLatestRate(ctx context.Context, from, to model.Currency) (*model.ExchangeRate, error) {

	if !from.IsSupported() || !to.IsSupported() {
		return nil, ErrInvalidCurrency
	}

	pair := model.CurrencyPair{
		BaseCurrency:   from,
		TargetCurrency: to,
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	if rate, found := s.cache.Get(ctx, pair, today); found {
		s.log.Info("Exchange rate found in cache", "pair", pair.String())
		return rate, nil
	}

	s.log.Info("Fetching exchange rate from repository", "pair", pair.String())
	rate, err := s.repository.FetchLatestRate(ctx, pair)
	if err != nil {
		s.log.Error("Failed to fetch exchange rate", "error", err, "pair", pair.String())
		return nil, fmt.Errorf("%w: %v", ErrExternalAPIFailure, err)
	}

	if err := s.cache.Set(ctx, rate); err != nil {
		s.log.Error("Failed to cache exchange rate", "error", err, "pair", pair.String())

	}

	return rate, nil
}

func (s *ExchangeService) GetHistoricalRate(ctx context.Context, from, to model.Currency, date time.Time) (*model.ExchangeRate, error) {

	if !from.IsSupported() || !to.IsSupported() {
		return nil, ErrInvalidCurrency
	}

	if err := validateDate(date); err != nil {
		return nil, err
	}

	pair := model.CurrencyPair{
		BaseCurrency:   from,
		TargetCurrency: to,
	}

	normalizedDate := date.UTC().Truncate(24 * time.Hour)
	if rate, found := s.cache.Get(ctx, pair, normalizedDate); found {
		return rate, nil
	}

	rate, err := s.repository.FetchHistoricalRate(ctx, pair, normalizedDate)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrExternalAPIFailure, err)
	}

	if err := s.cache.Set(ctx, rate); err != nil {

		s.log.Error("Failed to cache historical exchange rate", "error", err)
	}

	return rate, nil
}

func (s *ExchangeService) GetHistoricalRates(ctx context.Context, request model.HistoricalRateRequest) (*model.HistoricalRates, error) {

	if !request.BaseCurrency.IsSupported() || !request.TargetCurrency.IsSupported() {
		return nil, ErrInvalidCurrency
	}

	if err := validateDateRange(request.StartDate, request.EndDate); err != nil {
		return nil, err
	}

	rates, err := s.repository.FetchHistoricalRates(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrExternalAPIFailure, err)
	}

	return rates, nil
}

func (s *ExchangeService) ConvertCurrency(ctx context.Context, request model.ConversionRequest) (*model.ConversionResult, error) {

	if !request.FromCurrency.IsSupported() || !request.ToCurrency.IsSupported() {
		return nil, ErrInvalidCurrency
	}

	if request.Amount <= 0 {
		return nil, ErrInvalidAmount
	}

	var rate *model.ExchangeRate
	var err error

	if !request.Date.IsZero() {

		if err := validateDate(request.Date); err != nil {
			return nil, err
		}
		rate, err = s.GetHistoricalRate(ctx, request.FromCurrency, request.ToCurrency, request.Date)
	} else {

		rate, err = s.GetLatestRate(ctx, request.FromCurrency, request.ToCurrency)
	}

	if err != nil {
		return nil, err
	}

	convertedAmount := request.Amount * rate.Rate

	result := &model.ConversionResult{
		FromCurrency: request.FromCurrency,
		ToCurrency:   request.ToCurrency,
		FromAmount:   request.Amount,
		ToAmount:     convertedAmount,
		Rate:         rate.Rate,
		Date:         rate.Date,
	}

	return result, nil
}

func (s *ExchangeService) RefreshRates(ctx context.Context) error {
	s.log.Info("Refreshing exchange rates")

	err := s.repository.RefreshRates(ctx)
	if err != nil {
		s.log.Error("Failed to refresh exchange rates", "error", err)
		return fmt.Errorf("%w: %v", ErrExternalAPIFailure, err)
	}

	if err := s.cache.ClearExpired(ctx); err != nil {
		s.log.Error("Failed to clear expired cache entries", "error", err)

	}

	return nil
}

func validateDate(date time.Time) error {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	ninetyDaysAgo := today.AddDate(0, 0, -90)

	if date.Before(ninetyDaysAgo) {
		return ErrDateOutOfRange
	}

	return nil
}

func validateDateRange(startDate, endDate time.Time) error {

	if err := validateDate(startDate); err != nil {
		return err
	}

	if err := validateDate(endDate); err != nil {
		return err
	}

	if startDate.After(endDate) {
		return ErrInvalidDateRange
	}

	return nil
}
