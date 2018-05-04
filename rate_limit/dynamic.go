package rate_limit

import "gopkg.in/throttled/throttled.v2"

type RateLimiterSettings struct {
	reqsMinute int
	burstSize  int
}

type UpdatableRateLimiter interface {
	Update(settings RateLimiterSettings) error
	RateLimit(key string, quantity int) (bool, throttled.RateLimitResult, error)
}

// Implements UpdatableRateLimiter, so, we can modify
//  rate-limit settings in execution time
type DynamicRateLimiter struct {
	throttled.RateLimiter
	factory RateLimiterFactory
}

func NewDynamicRateLimiter(factory RateLimiterFactory, settings RateLimiterSettings) (UpdatableRateLimiter, error) {
	rateLimiter, err := factory.Build(settings.reqsMinute, settings.burstSize)
	if err != nil {
		return nil, err
	}
	return &DynamicRateLimiter{RateLimiter: rateLimiter, factory: factory}, nil
}

func (r *DynamicRateLimiter) Update(settings RateLimiterSettings) error {
	rateLimiter, err := r.factory.Build(settings.reqsMinute, settings.burstSize)
	if err != nil {
		return err
	}
	r.RateLimiter = rateLimiter
	return nil
}
