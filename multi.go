package ratelimit

import "gopkg.in/throttled/throttled.v2"

// Allows different GinRateLimit settings per siteKey (issuer)
//  implements UpdatableClusterRateLimiter (so it's cluster
//  aware and it can be updated in execution time)
type MultiRateLimiter struct {
	nodes     int
	customRL  map[string]UpdatableClusterRateLimiter
	defaultRL UpdatableClusterRateLimiter
}

func NewMultiRateLimiter(factory RateLimiterFactory, nodes int, defaultSettings RateLimiterSettings,
	customSettings map[string]RateLimiterSettings) (UpdatableClusterRateLimiter, error) {

	var err error
	customRL := make(map[string]UpdatableClusterRateLimiter)
	for k, v := range customSettings {
		customRL[k], err = NewClusterAwareRateLimiter(factory, nodes, v)
		if err != nil {
			return nil, err
		}
	}
	defaultRL, err := NewClusterAwareRateLimiter(factory, nodes, defaultSettings)
	if err != nil {
		return nil, err
	}
	return &MultiRateLimiter{
		nodes:     nodes,
		customRL:  customRL,
		defaultRL: defaultRL,
	}, nil
}

func (r *MultiRateLimiter) UpdateNodeCount(nodes int) error {
	if r.nodes != nodes {
		err := r.defaultRL.UpdateNodeCount(nodes)
		if err != nil {
			return err
		}
		for _, rl := range r.customRL {
			rl.UpdateNodeCount(nodes)
			if err != nil {
				return err
			}
		}
		r.nodes = nodes
	}
	return nil
}

func (r *MultiRateLimiter) RateLimit(key string, quantity int) (bool, throttled.RateLimitResult, error) {
	rateLimiter, ok := r.customRL[key]
	if !ok {
		rateLimiter = r.defaultRL
	}
	return rateLimiter.RateLimit(key, quantity)
}

func (r *MultiRateLimiter) Nodes() int { return r.nodes }
