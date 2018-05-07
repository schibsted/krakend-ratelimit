package ratelimit

import (
	"math"

	"github.com/throttled/throttled"
)

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

type UpdatableClusterRateLimiter interface {
	throttled.RateLimiter
	UpdateNodeCount(nodes int) error
	Nodes() int
}

// Updatable RateLimiter. Cluster aware (track node count).
// Update node sttings depending on total node count
type ClusterAwareRateLimiter struct {
	nodes       int
	settings    RateLimiterSettings
	rateLimiter UpdatableRateLimiter
}

func NewClusterAwareRateLimiter(factory RateLimiterFactory, nodes int, settings RateLimiterSettings) (UpdatableClusterRateLimiter, error) {
	nodeSettings := nodeSettings(settings, nodes)
	rateLimiter, err := NewDynamicRateLimiter(factory, nodeSettings)
	if err != nil {
		return nil, err
	}
	return &ClusterAwareRateLimiter{
		nodes:       nodes,
		settings:    settings,
		rateLimiter: rateLimiter,
	}, nil
}

func (r *ClusterAwareRateLimiter) Update(settings RateLimiterSettings) error {
	nodeSettings := nodeSettings(settings, r.nodes)
	err := r.rateLimiter.Update(nodeSettings)
	if err != nil {
		return err
	}
	r.settings = settings
	return nil
}

func (r *ClusterAwareRateLimiter) UpdateNodeCount(nodes int) error {
	if r.nodes != nodes {
		nodeSettings := nodeSettings(r.settings, nodes)
		err := r.rateLimiter.Update(nodeSettings)
		if err != nil {
			return err
		}
		r.nodes = nodes
	}
	return nil
}

func (r *ClusterAwareRateLimiter) RateLimit(key string, quantity int) (bool, throttled.RateLimitResult, error) {
	return r.rateLimiter.RateLimit(key, quantity)
}

func (r *ClusterAwareRateLimiter) Nodes() int { return r.nodes }

func nodeSettings(settings RateLimiterSettings, nodes int) RateLimiterSettings {
	return RateLimiterSettings{
		reqsMinute: valuePerNode(settings.reqsMinute, nodes),
		burstSize:  valuePerNode(settings.burstSize, nodes),
	}
}

func valuePerNode(clusterValue int, n int) int {
	return int(math.Ceil(float64(clusterValue) / float64(n)))
}
