package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"exchange-rate-service/internal/domain/model"
	"exchange-rate-service/pkg/logger"
)

type MemoryCache struct {
	cacheMap     map[string]*model.ExchangeRate
	mutex        sync.RWMutex
	cacheTTL     time.Duration
	log          *logger.Logger
}

func NewMemoryCache(cacheTTL time.Duration, log *logger.Logger) *MemoryCache {
	return &MemoryCache{
		cacheMap: make(map[string]*model.ExchangeRate),
		cacheTTL: cacheTTL,
		log:      log,
	}
}

func getCacheKey(pair model.CurrencyPair, date time.Time) string {
	dateStr := date.Format("2006-01-02")
	return fmt.Sprintf("%s-%s-%s", pair.BaseCurrency, pair.TargetCurrency, dateStr)
}

func (c *MemoryCache) Get(ctx context.Context, pair model.CurrencyPair, date time.Time) (*model.ExchangeRate, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	key := getCacheKey(pair, date)
	rate, found := c.cacheMap[key]
	
	if found {
		if time.Since(rate.LastUpdated) > c.cacheTTL {
			c.log.Debug("Cache entry expired", "key", key)
			return nil, false
		}
		c.log.Debug("Cache hit", "key", key)
		return rate, true
	}
	
	c.log.Debug("Cache miss", "key", key)
	return nil, false
}

func (c *MemoryCache) Set(ctx context.Context, rate *model.ExchangeRate) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	pair := model.CurrencyPair{
		BaseCurrency:   rate.BaseCurrency,
		TargetCurrency: rate.TargetCurrency,
	}
	
	key := getCacheKey(pair, rate.Date)
	c.cacheMap[key] = rate
	c.log.Debug("Cache set", "key", key)
	
	return nil
}

func (c *MemoryCache) ClearExpired(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	now := time.Now()
	expiredKeys := make([]string, 0)
	
	for key, rate := range c.cacheMap {
		if now.Sub(rate.LastUpdated) > c.cacheTTL {
			expiredKeys = append(expiredKeys, key)
		}
	}
	
	for _, key := range expiredKeys {
		delete(c.cacheMap, key)
		c.log.Debug("Removed expired cache entry", "key", key)
	}
	
	c.log.Info("Cleared expired cache entries", "count", len(expiredKeys))
	return nil
}
