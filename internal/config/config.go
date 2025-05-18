package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server     ServerConfig
	ExchangeAPI ExchangeAPIConfig
	Cache      CacheConfig
}

type ServerConfig struct {
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type ExchangeAPIConfig struct {
	BaseURL     string
	APIKey      string
	Timeout     time.Duration
	RefreshRate time.Duration
}

type CacheConfig struct {
	TTL time.Duration
}

func LoadConfig() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Port:         getEnvInt("SERVER_PORT", 8080),
			ReadTimeout:  getEnvDuration("SERVER_READ_TIMEOUT", 5*time.Second),
			WriteTimeout: getEnvDuration("SERVER_WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:  getEnvDuration("SERVER_IDLE_TIMEOUT", 120*time.Second),
		},
		ExchangeAPI: ExchangeAPIConfig{
			BaseURL:     getEnvString("EXCHANGE_API_BASE_URL", "https://api.exchangerate.host"),
			APIKey:      getEnvString("EXCHANGE_API_KEY", ""),
			Timeout:     getEnvDuration("EXCHANGE_API_TIMEOUT", 10*time.Second),
			RefreshRate: getEnvDuration("EXCHANGE_API_REFRESH_RATE", 1*time.Hour),
		},
		Cache: CacheConfig{
			TTL: getEnvDuration("CACHE_TTL", 30*time.Minute),
		},
	}
	
	return config, nil
}

func getEnvString(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		fmt.Printf("Warning: Invalid value for %s, using default: %d\n", key, defaultValue)
		return defaultValue
	}
	
	return value
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	
	value, err := time.ParseDuration(valueStr)
	if err != nil {
		fmt.Printf("Warning: Invalid duration for %s, using default: %s\n", key, defaultValue)
		return defaultValue
	}
	
	return value
}
