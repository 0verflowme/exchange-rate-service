FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go.mod first (without go.sum)
COPY go.mod ./

# Initialize the module and download dependencies
RUN go mod tidy

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /exchange-rate-service ./cmd/server

# Use a small image for the final container
FROM alpine:3.18

WORKDIR /

# Copy the binary from the builder stage
COPY --from=builder /exchange-rate-service /exchange-rate-service

# Expose the port the server listens on
EXPOSE 8080

# Run the service
CMD ["/exchange-rate-service"]
