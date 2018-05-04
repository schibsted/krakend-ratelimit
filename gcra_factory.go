package ratelimit

import (
	"gopkg.in/throttled/throttled.v2"
	"gopkg.in/throttled/throttled.v2/store/memstore"
)

type RateLimiterFactory interface {
	Build(reqsMinute int, burstSize int) (throttled.RateLimiter, error)
}

type InMemoryGCRARateLimiterFactory struct{}

func (f InMemoryGCRARateLimiterFactory) Build(reqsMinute int, burstSize int) (throttled.RateLimiter, error) {
	// Use in-memory storage
	maxKeys := 0 // no LRU (no keys limit)
	store, err := memstore.New(maxKeys)
	if err != nil {
		return nil, err
	}

	quota := throttled.RateQuota{throttled.PerMin(reqsMinute), burstSize}
	rateLimiter, err := throttled.NewGCRARateLimiter(store, quota)
	if err != nil {
		return nil, err
	}
	return rateLimiter, nil
}
