package model

import (
	"fmt"
	"time"
)

type ExchangeRate struct {
	BaseCurrency   Currency  `json:"base_currency"`
	TargetCurrency Currency  `json:"target_currency"`
	Rate           float64   `json:"rate"`
	Date           time.Time `json:"date"`
	LastUpdated    time.Time `json:"last_updated"`
}

type CurrencyPair struct {
	BaseCurrency   Currency `json:"base_currency"`
	TargetCurrency Currency `json:"target_currency"`
}

func (p CurrencyPair) String() string {
	return fmt.Sprintf("%s-%s", p.BaseCurrency, p.TargetCurrency)
}

type ConversionRequest struct {
	FromCurrency Currency  `json:"from_currency"`
	ToCurrency   Currency  `json:"to_currency"`
	Amount       float64   `json:"amount"`
	Date         time.Time `json:"date,omitempty"`
}

type ConversionResult struct {
	FromCurrency Currency  `json:"from_currency"`
	ToCurrency   Currency  `json:"to_currency"`
	FromAmount   float64   `json:"from_amount"`
	ToAmount     float64   `json:"to_amount"`
	Rate         float64   `json:"rate"`
	Date         time.Time `json:"date"`
}

type HistoricalRateRequest struct {
	BaseCurrency   Currency  `json:"base_currency"`
	TargetCurrency Currency  `json:"target_currency"`
	StartDate      time.Time `json:"start_date"`
	EndDate        time.Time `json:"end_date"`
}

type HistoricalRates struct {
	BaseCurrency   Currency                `json:"base_currency"`
	TargetCurrency Currency                `json:"target_currency"`
	Rates          map[string]ExchangeRate `json:"rates"`
}
