//
// Copyright 2011 - 2018 Schibsted Products & Technology AS.
// Licensed under the terms of the Apache 2.0 license. See LICENSE in the project root.
//

package ratelimit

type RateLimitConfig struct {
	Enabled bool                         `mapstructure:"enabled"`
	Default RateLimitSettings            `mapstructure:"default"`
	Custom  map[string]RateLimitSettings `mapstructure:"custom"`
}

type RateLimitSettings struct {
	MaxRequests int `mapstructure:"max_requests"`
	BurstSize   int `mapstructure:"burst_size"`
}

type RateLimiterSettings struct {
	reqsMinute int
	burstSize  int
}
