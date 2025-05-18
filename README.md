# Exchange Rate Service

Currency Exchange Rate Service that allows clients to fetch real-time exchange rates and perform currency conversions, with support for caching and historical data retrieval.

## Features

- **Real-time Exchange Rates**: Fetch the latest exchange rates for currency pairs
- **Currency Conversion**: Convert amounts between currencies
- **Historical Data**: Retrieve exchange rates for specific dates (up to 90 days)
- **In-memory Caching**: Efficient caching of exchange rates to reduce API calls
- **Concurrent Request Handling**: Thread-safe operations with proper synchronization
- **Graceful Error Handling**: Robust error handling for third-party API failures
- **API Refresh**: Automatic hourly refresh of exchange rates
- **Monitoring**: Prometheus and Grafana integration (optional)

## Supported Currencies

- United States Dollar (USD)
- Indian Rupee (INR)
- Euro (EUR)
- Japanese Yen (JPY)
- British Pound Sterling (GBP)

## Technologies

- **Go**: Core language (Go 1.21+)
- **Docker**: Containerization
- **Prometheus & Grafana**: Monitoring (optional)

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/rates?from=USD&to=INR` | GET | Get the latest exchange rate |
| `/api/v1/convert?from=USD&to=INR&amount=100&date=2025-01-01` | GET | Convert an amount between currencies |
| `/api/v1/historical?from=USD&to=INR&date=2025-01-01` | GET | Get the exchange rate for a specific date |
| `/api/v1/historical/range?from=USD&to=INR&start_date=2025-01-01&end_date=2025-01-10` | GET | Get exchange rates for a date range |
| `/health` | GET | Health check endpoint |

## Getting Started

### Prerequisites

- Docker and Docker Compose
- Go 1.24+ (for local development)

### Running with Docker

1. Clone the repository:

   ```bash
   git clone <repository-url>
   cd exchange-rate-service
   ```

2. (Optional) Set up environment variables:

   ```bash
   export EXCHANGE_API_KEY=your_api_key_if_needed
   ```

3. Build and run the service:

   ```bash
   docker-compose up -d
   ```

4. The service will be available at `http://localhost:8080`

## API Usage Examples

### Get Latest Exchange Rate

```bash
curl "http://localhost:8080/api/v1/rates?from=USD&to=INR"
```

Expected Response:

```json
{
  "success": true,
  "data": {
    "base_currency": "USD",
    "target_currency": "INR",
    "rate": 82.5,
    "date": "2025-05-15T00:00:00Z",
    "last_updated": "2025-05-15T12:30:45Z"
  }
}
```

### Convert Currency

```bash
curl "http://localhost:8080/api/v1/convert?from=USD&to=INR&amount=100"
```

Expected Response:

```json
{
  "success": true,
  "data": {
    "amount": 8250.0
  }
}
```

### Get Historical Rate

```bash
curl "http://localhost:8080/api/v1/historical?from=USD&to=INR&date=2025-04-01"
```

Expected Response:

```json
{
  "success": true,
  "data": {
    "base_currency": "USD",
    "target_currency": "INR",
    "rate": 82.3,
    "date": "2025-04-01T00:00:00Z",
    "last_updated": "2025-05-15T12:35:22Z"
  }
}
```

## Configuration Options

The service can be configured using environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_PORT` | HTTP server port | 8080 |
| `EXCHANGE_API_BASE_URL` | Base URL for the exchange rate API | <https://api.exchangerate.host> |
| `EXCHANGE_API_KEY` | API key for the exchange rate service | - |
| `EXCHANGE_API_REFRESH_RATE` | How often to refresh rates | 1h |
| `CACHE_TTL` | How long to cache rates | 30m |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | info |

## Monitoring

The service includes Prometheus and Grafana integration for monitoring. Access Grafana at `http://localhost:3000` with default credentials (admin/admin).

## Testing

Run the tests with:

```bash
go test -v ./...
```
