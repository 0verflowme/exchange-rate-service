package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"exchange-rate-service/internal/domain/model"
	"exchange-rate-service/pkg/logger"
)

type ExchangeAPI struct {
	baseURL     string
	apiKey      string
	httpClient  *http.Client
	log         *logger.Logger
	mutex       sync.RWMutex
	latestRates map[string]*model.ExchangeRate
}

type exchangerateAPIResponse struct {
	Success   bool               `json:"success"`
	Terms     string             `json:"terms,omitempty"`
	Privacy   string             `json:"privacy,omitempty"`
	Timestamp int64              `json:"timestamp"`
	Source    string             `json:"source"`
	Quotes    map[string]float64 `json:"quotes"`
}

func NewExchangeAPI(baseURL, apiKey string, timeout time.Duration, log *logger.Logger) *ExchangeAPI {
	return &ExchangeAPI{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		log:         log,
		latestRates: make(map[string]*model.ExchangeRate),
	}
}

func (e *ExchangeAPI) FetchLatestRate(ctx context.Context, pair model.CurrencyPair) (*model.ExchangeRate, error) {

	cacheKey := fmt.Sprintf("%s-%s", pair.BaseCurrency, pair.TargetCurrency)

	e.mutex.RLock()
	if rate, exists := e.latestRates[cacheKey]; exists {
		e.mutex.RUnlock()
		return rate, nil
	}
	e.mutex.RUnlock()

	rates, err := e.fetchAllLatestRates(ctx)
	if err != nil {
		return nil, err
	}

	rate, err := e.extractRate(rates, pair)
	if err != nil {
		return nil, err
	}

	return rate, nil
}

func (e *ExchangeAPI) fetchAllLatestRates(ctx context.Context) (map[string]float64, error) {

	url := fmt.Sprintf("%s/live?base=USD", e.baseURL)

	if e.apiKey != "" {
		url += "&access_key=" + e.apiKey
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned non-OK status: %d", resp.StatusCode)
	}

	var apiResp exchangerateAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("API reported failure")
	}

	return apiResp.Quotes, nil
}

func (e *ExchangeAPI) extractRate(quotes map[string]float64, pair model.CurrencyPair) (*model.ExchangeRate, error) {

	if pair.BaseCurrency == model.USD {
		rateKey := fmt.Sprintf("USD%s", pair.TargetCurrency)
		rate, exists := quotes[rateKey]
		if !exists {
			return nil, fmt.Errorf("rate not found for currency: %s", pair.TargetCurrency)
		}

		exchangeRate := &model.ExchangeRate{
			BaseCurrency:   pair.BaseCurrency,
			TargetCurrency: pair.TargetCurrency,
			Rate:           rate,
			Date:           time.Now().UTC().Truncate(24 * time.Hour),
			LastUpdated:    time.Now(),
		}

		cacheKey := fmt.Sprintf("%s-%s", pair.BaseCurrency, pair.TargetCurrency)
		e.mutex.Lock()
		e.latestRates[cacheKey] = exchangeRate
		e.mutex.Unlock()

		return exchangeRate, nil
	}

	if pair.TargetCurrency == model.USD {
		rateKey := fmt.Sprintf("USD%s", pair.BaseCurrency)
		rate, exists := quotes[rateKey]
		if !exists {
			return nil, fmt.Errorf("rate not found for currency: %s", pair.BaseCurrency)
		}

		inverseRate := 1.0 / rate

		exchangeRate := &model.ExchangeRate{
			BaseCurrency:   pair.BaseCurrency,
			TargetCurrency: pair.TargetCurrency,
			Rate:           inverseRate,
			Date:           time.Now().UTC().Truncate(24 * time.Hour),
			LastUpdated:    time.Now(),
		}

		cacheKey := fmt.Sprintf("%s-%s", pair.BaseCurrency, pair.TargetCurrency)
		e.mutex.Lock()
		e.latestRates[cacheKey] = exchangeRate
		e.mutex.Unlock()

		return exchangeRate, nil
	}

	baseUsdKey := fmt.Sprintf("USD%s", pair.BaseCurrency)
	targetUsdKey := fmt.Sprintf("USD%s", pair.TargetCurrency)

	baseRate, baseExists := quotes[baseUsdKey]
	targetRate, targetExists := quotes[targetUsdKey]

	if !baseExists {
		return nil, fmt.Errorf("rate not found for currency: %s", pair.BaseCurrency)
	}
	if !targetExists {
		return nil, fmt.Errorf("rate not found for currency: %s", pair.TargetCurrency)
	}

	crossRate := targetRate / baseRate

	exchangeRate := &model.ExchangeRate{
		BaseCurrency:   pair.BaseCurrency,
		TargetCurrency: pair.TargetCurrency,
		Rate:           crossRate,
		Date:           time.Now().UTC().Truncate(24 * time.Hour),
		LastUpdated:    time.Now(),
	}

	cacheKey := fmt.Sprintf("%s-%s", pair.BaseCurrency, pair.TargetCurrency)
	e.mutex.Lock()
	e.latestRates[cacheKey] = exchangeRate
	e.mutex.Unlock()

	return exchangeRate, nil
}

func (e *ExchangeAPI) FetchHistoricalRate(ctx context.Context, pair model.CurrencyPair, date time.Time) (*model.ExchangeRate, error) {

	dateStr := date.Format("2006-01-02")

	url := fmt.Sprintf("%s/historical?date=%s&base=USD",
		e.baseURL,
		dateStr,
	)

	if e.apiKey != "" {
		url += "&access_key=" + e.apiKey
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned non-OK status: %d", resp.StatusCode)
	}

	var apiResp exchangerateAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("API reported failure")
	}

	tempPair := model.CurrencyPair{
		BaseCurrency:   pair.BaseCurrency,
		TargetCurrency: pair.TargetCurrency,
	}

	rate, err := e.extractHistoricalRate(apiResp.Quotes, tempPair, date)
	if err != nil {
		return nil, err
	}

	return rate, nil
}

func (e *ExchangeAPI) extractHistoricalRate(quotes map[string]float64, pair model.CurrencyPair, date time.Time) (*model.ExchangeRate, error) {

	if pair.BaseCurrency == model.USD {
		rateKey := fmt.Sprintf("USD%s", pair.TargetCurrency)
		rate, exists := quotes[rateKey]
		if !exists {
			return nil, fmt.Errorf("rate not found for currency: %s", pair.TargetCurrency)
		}

		return &model.ExchangeRate{
			BaseCurrency:   pair.BaseCurrency,
			TargetCurrency: pair.TargetCurrency,
			Rate:           rate,
			Date:           date,
			LastUpdated:    time.Now(),
		}, nil
	}

	if pair.TargetCurrency == model.USD {
		rateKey := fmt.Sprintf("USD%s", pair.BaseCurrency)
		rate, exists := quotes[rateKey]
		if !exists {
			return nil, fmt.Errorf("rate not found for currency: %s", pair.BaseCurrency)
		}

		return &model.ExchangeRate{
			BaseCurrency:   pair.BaseCurrency,
			TargetCurrency: pair.TargetCurrency,
			Rate:           1.0 / rate,
			Date:           date,
			LastUpdated:    time.Now(),
		}, nil
	}

	baseUsdKey := fmt.Sprintf("USD%s", pair.BaseCurrency)
	targetUsdKey := fmt.Sprintf("USD%s", pair.TargetCurrency)

	baseRate, baseExists := quotes[baseUsdKey]
	targetRate, targetExists := quotes[targetUsdKey]

	if !baseExists {
		return nil, fmt.Errorf("rate not found for currency: %s", pair.BaseCurrency)
	}
	if !targetExists {
		return nil, fmt.Errorf("rate not found for currency: %s", pair.TargetCurrency)
	}

	return &model.ExchangeRate{
		BaseCurrency:   pair.BaseCurrency,
		TargetCurrency: pair.TargetCurrency,
		Rate:           targetRate / baseRate,
		Date:           date,
		LastUpdated:    time.Now(),
	}, nil
}

func (e *ExchangeAPI) FetchHistoricalRates(ctx context.Context, request model.HistoricalRateRequest) (*model.HistoricalRates, error) {

	result := &model.HistoricalRates{
		BaseCurrency:   request.BaseCurrency,
		TargetCurrency: request.TargetCurrency,
		Rates:          make(map[string]model.ExchangeRate),
	}

	currentDate := request.StartDate
	for !currentDate.After(request.EndDate) {

		pair := model.CurrencyPair{
			BaseCurrency:   request.BaseCurrency,
			TargetCurrency: request.TargetCurrency,
		}

		rate, err := e.FetchHistoricalRate(ctx, pair, currentDate)
		if err != nil {
			e.log.Error("Failed to fetch historical rate", "error", err, "date", currentDate.Format("2006-01-02"))

			currentDate = currentDate.AddDate(0, 0, 1)
			continue
		}

		dateKey := currentDate.Format("2006-01-02")
		result.Rates[dateKey] = *rate

		currentDate = currentDate.AddDate(0, 0, 1)
	}

	return result, nil
}

func (e *ExchangeAPI) RefreshRates(ctx context.Context) error {
	e.log.Info("Refreshing all exchange rates")

	e.mutex.Lock()
	e.latestRates = make(map[string]*model.ExchangeRate)
	e.mutex.Unlock()

	rates, err := e.fetchAllLatestRates(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch latest rates: %w", err)
	}

	for _, base := range model.SupportedCurrencies {
		for _, target := range model.SupportedCurrencies {
			if base == target {
				continue
			}

			pair := model.CurrencyPair{
				BaseCurrency:   base,
				TargetCurrency: target,
			}

			_, err := e.extractRate(rates, pair)
			if err != nil {
				e.log.Error("Failed to extract rate", "error", err, "pair", pair.String())
			}
		}
	}

	e.log.Info("Successfully refreshed all exchange rates")
	return nil
}
