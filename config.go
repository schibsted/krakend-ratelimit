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
