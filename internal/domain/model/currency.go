package model

type Currency string

const (
	USD Currency = "USD"
	INR Currency = "INR"
	EUR Currency = "EUR"
	JPY Currency = "JPY"
	GBP Currency = "GBP"
)

var SupportedCurrencies = []Currency{USD, INR, EUR, JPY, GBP}

func (c Currency) IsSupported() bool {
	for _, supportedCurrency := range SupportedCurrencies {
		if c == supportedCurrency {
			return true
		}
	}
	return false
}

func (c Currency) String() string {
	return string(c)
}
