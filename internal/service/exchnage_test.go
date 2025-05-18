package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"exchange-rate-service/internal/domain/model"
	"exchange-rate-service/pkg/logger"
)

type MockRateCache struct {
	GetFunc          func(ctx context.Context, pair model.CurrencyPair, date time.Time) (*model.ExchangeRate, bool)
	SetFunc          func(ctx context.Context, rate *model.ExchangeRate) error
	ClearExpiredFunc func(ctx context.Context) error
}

func (m *MockRateCache) Get(ctx context.Context, pair model.CurrencyPair, date time.Time) (*model.ExchangeRate, bool) {
	return m.GetFunc(ctx, pair, date)
}

func (m *MockRateCache) Set(ctx context.Context, rate *model.ExchangeRate) error {
	return m.SetFunc(ctx, rate)
}

func (m *MockRateCache) ClearExpired(ctx context.Context) error {
	return m.ClearExpiredFunc(ctx)
}

type MockRateRepository struct {
	FetchLatestRateFunc      func(ctx context.Context, pair model.CurrencyPair) (*model.ExchangeRate, error)
	FetchHistoricalRateFunc  func(ctx context.Context, pair model.CurrencyPair, date time.Time) (*model.ExchangeRate, error)
	FetchHistoricalRatesFunc func(ctx context.Context, request model.HistoricalRateRequest) (*model.HistoricalRates, error)
	RefreshRatesFunc         func(ctx context.Context) error
}

func (m *MockRateRepository) FetchLatestRate(ctx context.Context, pair model.CurrencyPair) (*model.ExchangeRate, error) {
	return m.FetchLatestRateFunc(ctx, pair)
}

func (m *MockRateRepository) FetchHistoricalRate(ctx context.Context, pair model.CurrencyPair, date time.Time) (*model.ExchangeRate, error) {
	return m.FetchHistoricalRateFunc(ctx, pair, date)
}

func (m *MockRateRepository) FetchHistoricalRates(ctx context.Context, request model.HistoricalRateRequest) (*model.HistoricalRates, error) {
	return m.FetchHistoricalRatesFunc(ctx, request)
}

func (m *MockRateRepository) RefreshRates(ctx context.Context) error {
	return m.RefreshRatesFunc(ctx)
}

func TestExchangeService_GetLatestRate(t *testing.T) {

	log := logger.NewLogger("debug")

	testCases := []struct {
		name           string
		from           model.Currency
		to             model.Currency
		mockCache      MockRateCache
		mockRepository MockRateRepository
		expectedRate   *model.ExchangeRate
		expectedError  error
	}{
		{
			name: "Success - Cache Hit",
			from: model.USD,
			to:   model.INR,
			mockCache: MockRateCache{
				GetFunc: func(ctx context.Context, pair model.CurrencyPair, date time.Time) (*model.ExchangeRate, bool) {
					return &model.ExchangeRate{
						BaseCurrency:   model.USD,
						TargetCurrency: model.INR,
						Rate:           82.5,
						Date:           time.Now().Truncate(24 * time.Hour),
						LastUpdated:    time.Now(),
					}, true
				},
			},
			mockRepository: MockRateRepository{},
			expectedRate: &model.ExchangeRate{
				BaseCurrency:   model.USD,
				TargetCurrency: model.INR,
				Rate:           82.5,
			},
			expectedError: nil,
		},
		{
			name: "Success - Cache Miss, Repository Hit",
			from: model.USD,
			to:   model.INR,
			mockCache: MockRateCache{
				GetFunc: func(ctx context.Context, pair model.CurrencyPair, date time.Time) (*model.ExchangeRate, bool) {
					return nil, false
				},
				SetFunc: func(ctx context.Context, rate *model.ExchangeRate) error {
					return nil
				},
			},
			mockRepository: MockRateRepository{
				FetchLatestRateFunc: func(ctx context.Context, pair model.CurrencyPair) (*model.ExchangeRate, error) {
					return &model.ExchangeRate{
						BaseCurrency:   model.USD,
						TargetCurrency: model.INR,
						Rate:           82.5,
						Date:           time.Now().Truncate(24 * time.Hour),
						LastUpdated:    time.Now(),
					}, nil
				},
			},
			expectedRate: &model.ExchangeRate{
				BaseCurrency:   model.USD,
				TargetCurrency: model.INR,
				Rate:           82.5,
			},
			expectedError: nil,
		},
		{
			name:           "Error - Invalid Currency",
			from:           model.Currency("XYZ"),
			to:             model.INR,
			mockCache:      MockRateCache{},
			mockRepository: MockRateRepository{},
			expectedRate:   nil,
			expectedError:  ErrInvalidCurrency,
		},
		{
			name: "Error - Repository Error",
			from: model.USD,
			to:   model.INR,
			mockCache: MockRateCache{
				GetFunc: func(ctx context.Context, pair model.CurrencyPair, date time.Time) (*model.ExchangeRate, bool) {
					return nil, false
				},
			},
			mockRepository: MockRateRepository{
				FetchLatestRateFunc: func(ctx context.Context, pair model.CurrencyPair) (*model.ExchangeRate, error) {
					return nil, errors.New("API error")
				},
			},
			expectedRate:  nil,
			expectedError: ErrExternalAPIFailure,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			svc := NewExchangeService(&tc.mockRepository, &tc.mockCache, log)

			rate, err := svc.GetLatestRate(context.Background(), tc.from, tc.to)

			if (tc.expectedError != nil && err == nil) || (tc.expectedError == nil && err != nil) {
				t.Errorf("Expected error: %v, got: %v", tc.expectedError, err)
			}

			if tc.expectedError != nil && err != nil {
				if !errors.Is(err, tc.expectedError) {
					t.Errorf("Expected error to contain: %v, got: %v", tc.expectedError, err)
				}
			}

			if tc.expectedRate == nil && rate != nil {
				t.Errorf("Expected nil rate, got: %v", rate)
			}

			if tc.expectedRate != nil {
				if rate == nil {
					t.Fatal("Expected non-nil rate, got nil")
				}

				if tc.expectedRate.BaseCurrency != rate.BaseCurrency {
					t.Errorf("Expected base currency: %s, got: %s", tc.expectedRate.BaseCurrency, rate.BaseCurrency)
				}

				if tc.expectedRate.TargetCurrency != rate.TargetCurrency {
					t.Errorf("Expected target currency: %s, got: %s", tc.expectedRate.TargetCurrency, rate.TargetCurrency)
				}

				if tc.expectedRate.Rate != rate.Rate {
					t.Errorf("Expected rate: %f, got: %f", tc.expectedRate.Rate, rate.Rate)
				}
			}
		})
	}
}

func TestExchangeService_ConvertCurrency(t *testing.T) {

	log := logger.NewLogger("debug")

	testCases := []struct {
		name           string
		request        model.ConversionRequest
		mockCache      MockRateCache
		mockRepository MockRateRepository
		expectedResult *model.ConversionResult
		expectedError  error
	}{
		{
			name: "Success - Latest Rate",
			request: model.ConversionRequest{
				FromCurrency: model.USD,
				ToCurrency:   model.INR,
				Amount:       100,
			},
			mockCache: MockRateCache{
				GetFunc: func(ctx context.Context, pair model.CurrencyPair, date time.Time) (*model.ExchangeRate, bool) {
					return &model.ExchangeRate{
						BaseCurrency:   model.USD,
						TargetCurrency: model.INR,
						Rate:           82.5,
						Date:           time.Now().Truncate(24 * time.Hour),
						LastUpdated:    time.Now(),
					}, true
				},
			},
			mockRepository: MockRateRepository{},
			expectedResult: &model.ConversionResult{
				FromCurrency: model.USD,
				ToCurrency:   model.INR,
				FromAmount:   100,
				ToAmount:     8250,
				Rate:         82.5,
			},
			expectedError: nil,
		},
		{
			name: "Error - Invalid Amount",
			request: model.ConversionRequest{
				FromCurrency: model.USD,
				ToCurrency:   model.INR,
				Amount:       -100,
			},
			mockCache:      MockRateCache{},
			mockRepository: MockRateRepository{},
			expectedResult: nil,
			expectedError:  ErrInvalidAmount,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			svc := NewExchangeService(&tc.mockRepository, &tc.mockCache, log)

			result, err := svc.ConvertCurrency(context.Background(), tc.request)

			if (tc.expectedError != nil && err == nil) || (tc.expectedError == nil && err != nil) {
				t.Errorf("Expected error: %v, got: %v", tc.expectedError, err)
			}

			if tc.expectedError != nil && err != nil {
				if !errors.Is(err, tc.expectedError) {
					t.Errorf("Expected error to contain: %v, got: %v", tc.expectedError, err)
				}
			}

			if tc.expectedResult == nil && result != nil {
				t.Errorf("Expected nil result, got: %v", result)
			}

			if tc.expectedResult != nil {
				if result == nil {
					t.Fatal("Expected non-nil result, got nil")
				}

				if tc.expectedResult.FromCurrency != result.FromCurrency {
					t.Errorf("Expected from currency: %s, got: %s", tc.expectedResult.FromCurrency, result.FromCurrency)
				}

				if tc.expectedResult.ToCurrency != result.ToCurrency {
					t.Errorf("Expected to currency: %s, got: %s", tc.expectedResult.ToCurrency, result.ToCurrency)
				}

				if tc.expectedResult.FromAmount != result.FromAmount {
					t.Errorf("Expected from amount: %f, got: %f", tc.expectedResult.FromAmount, result.FromAmount)
				}

				if tc.expectedResult.ToAmount != result.ToAmount {
					t.Errorf("Expected to amount: %f, got: %f", tc.expectedResult.ToAmount, result.ToAmount)
				}

				if tc.expectedResult.Rate != result.Rate {
					t.Errorf("Expected rate: %f, got: %f", tc.expectedResult.Rate, result.Rate)
				}
			}
		})
	}
}
