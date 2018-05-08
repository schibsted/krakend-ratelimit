//
// Copyright 2011 - 2018 Schibsted Products & Technology AS.
// Licensed under the terms of the Apache 2.0 license. See LICENSE in the project root.
//

package ratelimit

import (
	"time"

	"github.com/devopsfaith/krakend/logging"
	"github.com/throttled/throttled"
	"github.com/throttled/throttled/store/memstore"
)

type NodeCounter func() int

func DefaultNodeCounter() NodeCounter {
	return func() int {
		return 1
	}
}

func BuildRateLimiter(c RateLimitConfig, nodes NodeCounter, logger logging.Logger) UpdatableClusterRateLimiter {
	factory := InMemoryGCRARateLimiterFactory{}

	defaultSettings := getRLSettings(c.Default)
	customSettings := make(map[string]RateLimiterSettings)
	for k, v := range c.Custom {
		s := getRLSettings(v)
		customSettings[k] = s
		logger.Info("Starting RateLimit with reqsMin:", s.reqsMinute, " and burstSize:", s.burstSize)
	}

	rateLimiter, err := NewMultiRateLimiter(factory, nodes(), defaultSettings, customSettings)
	if err != nil {
		logger.Fatal("ERROR:", err.Error())
	}
	return rateLimiter
}

// Update internal GinRateLimit settings depending on service configuration on ApiGW nodes amount
func RateLimitUpdater(rateLimiter UpdatableClusterRateLimiter, interval time.Duration, nodeCounter NodeCounter, logger logging.Logger) {
	for {
		time.Sleep(interval)
		nodeCount := nodeCounter()
		if nodeCount != rateLimiter.Nodes() {
			logger.Info("Updating RateLimit node count", nodeCount)
			rateLimiter.UpdateNodeCount(nodeCount)
		}
	}
}

func getRLSettings(s RateLimitSettings) RateLimiterSettings {
	return RateLimiterSettings{
		reqsMinute: s.MaxRequests,
		burstSize:  s.BurstSize,
	}
}

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
