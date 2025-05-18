package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"exchange-rate-service/internal/domain/model"
	"exchange-rate-service/internal/domain/ports"
	"exchange-rate-service/internal/metrics"
	"exchange-rate-service/internal/service"
	"exchange-rate-service/pkg/logger"
)

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type Handler struct {
	service ports.ExchangeService
	log     *logger.Logger
	metrics *metrics.Metrics
}

func NewHandler(service ports.ExchangeService, log *logger.Logger, metrics *metrics.Metrics) *Handler {
	return &Handler{
		service: service,
		log:     log,
		metrics: metrics,
	}
}

func parseDate(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, nil
	}
	return time.Parse("2006-01-02", dateStr)
}

func (h *Handler) GetLatestRateHandler(w http.ResponseWriter, r *http.Request) {
	h.metrics.RateRequestsTotal.Inc()
	
	from := model.Currency(r.URL.Query().Get("from"))
	to := model.Currency(r.URL.Query().Get("to"))
	
	if from == "" || to == "" {
		h.sendErrorResponse(w, http.StatusBadRequest, "missing required parameters: from and to")
		return
	}
	
	ctx := r.Context()
	rate, err := h.service.GetLatestRate(ctx, from, to)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	
	h.sendSuccessResponse(w, rate)
}

func (h *Handler) ConvertCurrencyHandler(w http.ResponseWriter, r *http.Request) {
	h.metrics.ConversionRequestsTotal.Inc()
	
	from := model.Currency(r.URL.Query().Get("from"))
	to := model.Currency(r.URL.Query().Get("to"))
	amountStr := r.URL.Query().Get("amount")
	dateStr := r.URL.Query().Get("date")
	
	if from == "" || to == "" {
		h.sendErrorResponse(w, http.StatusBadRequest, "missing required parameters: from and to")
		return
	}
	
	amount := 1.0
	if amountStr != "" {
		var err error
		amount, err = strconv.ParseFloat(amountStr, 64)
		if err != nil {
			h.sendErrorResponse(w, http.StatusBadRequest, "invalid amount parameter")
			return
		}
	}
	
	var date time.Time
	var err error
	if dateStr != "" {
		date, err = parseDate(dateStr)
		if err != nil {
			h.sendErrorResponse(w, http.StatusBadRequest, "invalid date format, use YYYY-MM-DD")
			return
		}
	}
	
	request := model.ConversionRequest{
		FromCurrency: from,
		ToCurrency:   to,
		Amount:       amount,
		Date:         date,
	}
	
	ctx := r.Context()
	result, err := h.service.ConvertCurrency(ctx, request)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	
	simplifiedResult := map[string]float64{
		"amount": result.ToAmount,
	}
	h.sendSuccessResponse(w, simplifiedResult)
}

func (h *Handler) GetHistoricalRateHandler(w http.ResponseWriter, r *http.Request) {
	h.metrics.HistoricalRequestsTotal.Inc()
	
	from := model.Currency(r.URL.Query().Get("from"))
	to := model.Currency(r.URL.Query().Get("to"))
	dateStr := r.URL.Query().Get("date")
	
	if from == "" || to == "" || dateStr == "" {
		h.sendErrorResponse(w, http.StatusBadRequest, "missing required parameters: from, to, and date")
		return
	}
	
	date, err := parseDate(dateStr)
	if err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, "invalid date format, use YYYY-MM-DD")
		return
	}
	
	ctx := r.Context()
	rate, err := h.service.GetHistoricalRate(ctx, from, to, date)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	
	h.sendSuccessResponse(w, rate)
}

func (h *Handler) GetHistoricalRatesHandler(w http.ResponseWriter, r *http.Request) {
	h.metrics.HistoricalRequestsTotal.Inc()
	
	from := model.Currency(r.URL.Query().Get("from"))
	to := model.Currency(r.URL.Query().Get("to"))
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")
	
	if from == "" || to == "" || startDateStr == "" || endDateStr == "" {
		h.sendErrorResponse(w, http.StatusBadRequest, "missing required parameters: from, to, start_date, and end_date")
		return
	}
	
	startDate, err := parseDate(startDateStr)
	if err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, "invalid start_date format, use YYYY-MM-DD")
		return
	}
	
	endDate, err := parseDate(endDateStr)
	if err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, "invalid end_date format, use YYYY-MM-DD")
		return
	}
	
	request := model.HistoricalRateRequest{
		BaseCurrency:   from,
		TargetCurrency: to,
		StartDate:      startDate,
		EndDate:        endDate,
	}
	
	ctx := r.Context()
	rates, err := h.service.GetHistoricalRates(ctx, request)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	
	h.sendSuccessResponse(w, rates)
}

func (h *Handler) sendSuccessResponse(w http.ResponseWriter, data interface{}) {
	response := Response{
		Success: true,
		Data:    data,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.log.Error("Failed to encode response", "error", err)
	}
}

func (h *Handler) sendErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	response := Response{
		Success: false,
		Error:   message,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.log.Error("Failed to encode error response", "error", err)
	}
}

func (h *Handler) handleServiceError(w http.ResponseWriter, err error) {
	statusCode := http.StatusInternalServerError
	errorMessage := "internal server error"
	
	switch {
	case errors.Is(err, service.ErrInvalidCurrency):
		statusCode = http.StatusBadRequest
		errorMessage = "invalid currency"
	case errors.Is(err, service.ErrDateOutOfRange):
		statusCode = http.StatusBadRequest
		errorMessage = "date is outside allowed range (older than 90 days)"
	case errors.Is(err, service.ErrInvalidDateRange):
		statusCode = http.StatusBadRequest
		errorMessage = "invalid date range"
	case errors.Is(err, service.ErrRateNotFound):
		statusCode = http.StatusNotFound
		errorMessage = "exchange rate not found"
	case errors.Is(err, service.ErrExternalAPIFailure):
		statusCode = http.StatusServiceUnavailable
		errorMessage = "external API failure"
	case errors.Is(err, service.ErrInvalidAmount):
		statusCode = http.StatusBadRequest
		errorMessage = "invalid amount"
	}
	
	h.log.Error("Service error", "error", err, "status_code", statusCode)
	h.sendErrorResponse(w, statusCode, errorMessage)
}
