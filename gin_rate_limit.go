//
// Copyright 2011 - 2018 Schibsted Products & Technology AS.
// Licensed under the terms of the Apache 2.0 license. See LICENSE in the project root.
//

package ratelimit

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/tomasen/realip"

	"github.com/throttled/throttled"

	"github.com/devopsfaith/krakend/config"
	"github.com/devopsfaith/krakend/logging"

	"math"
	"net/http"
	"strconv"
	"time"
)

// Namespace is the key to look for extra configuration details
const Namespace = "github.com/tgracchus/krakend-ratelimit"

// ConfigGetter implements the config.ConfigGetter interface
func ConfigGetter(e config.ExtraConfig) interface{} {
	v, ok := e[Namespace]
	if !ok {
		return nil
	}
	tmp, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	cfg := RateLimitConfig{}
	if val, ok := tmp["enabled"]; ok {
		cfg.Enabled = val.(bool)
	}

	if val, ok := tmp["default"]; ok {
		defaults, ok := val.(map[string]interface{})
		if !ok {
			return nil
		}
		settings := RateLimitSettings{}
		if val, ok := defaults["max_requests"]; ok {
			settings.MaxRequests = int(val.(float64))
		}
		if val, ok := defaults["burst_size"]; ok {
			settings.BurstSize = int(val.(float64))
		}
		cfg.Default = settings
	}

	return cfg
}

var (
	rateLimiterUpdateRate = 10 * time.Second
)

func GinRateLimit(cfg RateLimitConfig, nodeCounter NodeCounter, logger logging.Logger) (UpdatableClusterRateLimiter, error) {
	rateLimiter := BuildRateLimiter(cfg, nodeCounter, logger)
	go RateLimitUpdater(rateLimiter, rateLimiterUpdateRate, nodeCounter, logger)

	return rateLimiter, nil
}

// Build context key based rate limiter (e.g. we can store the issuer/tenant as a context param)
func GinContextRateLimit(rateLimiter throttled.RateLimiter, contextKey string) *GinRateLimiter {
	varyBy := readContextKey(contextKey)
	return &GinRateLimiter{
		RateLimiter: rateLimiter,
		VaryBy:      varyBy,
	}
}

// Build IP based rate limiter
func GinIpRateLimit(rateLimiter throttled.RateLimiter, contextKey string) *GinRateLimiter {
	return &GinRateLimiter{
		RateLimiter: rateLimiter,
		VaryBy:      getRequestIp,
	}
}

type VaryByFunc func(*gin.Context) string

// Use site-key stored in context (and previously extracted from JWT token
func readContextKey(contextKey string) VaryByFunc {
	return func(c *gin.Context) string {
		value, found := c.Get(contextKey)
		if !found {
			return "unknown"
		} else {
			return value.(string)
		}
	}
}

func getRequestIp(c *gin.Context) string {
	return realip.RealIP(c.Request)
}

var (
	// DefaultDeniedHandler is the default DeniedHandler for an
	// HTTPRateLimiter. It returns a 429 status code with a generic
	// message.
	DefaultDeniedHandler = func(c *gin.Context) {
		c.JSON(http.StatusTooManyRequests, "limit exceeded")
		c.AbortWithStatus(http.StatusTooManyRequests)
	}

	// DefaultError is the default Error function for an HTTPRateLimiter.
	// It returns a 500 status code with a generic message.
	DefaultError = func(c *gin.Context, err error) {
		c.JSON(http.StatusInternalServerError, "internal error")
		c.AbortWithStatus(http.StatusInternalServerError)
	}
)

// GinRateLimiter faciliates using a Limiter to limit HTTP requests.
type GinRateLimiter struct {
	// DeniedHandler is called if the request is disallowed. If it is
	// nil, the DefaultDeniedHandler variable is used.
	DeniedHandler gin.HandlerFunc

	// Error is called if the RateLimiter returns an error. If it is
	// nil, the DefaultErrorFunc is used.
	Error func(*gin.Context, error)

	// Limiter is call for each request to determine whether the
	// request is permitted and update internal state. It must be set.
	RateLimiter throttled.RateLimiter

	// VaryBy is called for each request to generate a key for the
	// limiter. If it is nil, all requests use an empty string key.
	VaryBy func(*gin.Context) string
}

// Requests that are not limited will be passed to the handler
// unchanged.  Limited requests will be passed to the DeniedHandler.
// X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset and
// Retry-After headers will be written to the response based on the
// values in the RateLimitResult.
func (t *GinRateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if t.RateLimiter == nil {
			t.error(c, errors.New("You must set a RateLimiter on HTTPRateLimiter"))
		}

		var key string
		if t.VaryBy != nil {
			key = t.VaryBy(c)
		}

		limited, context, err := t.RateLimiter.RateLimit(key, 1)

		if err != nil {
			t.error(c, err)
			return
		}

		setRateLimitHeaders(c, context)

		if !limited {
			c.Next()
		} else {
			dh := t.DeniedHandler
			if dh == nil {
				dh = DefaultDeniedHandler
			}
			dh(c)
		}
	}
}

func (t *GinRateLimiter) error(c *gin.Context, err error) {
	e := t.Error
	if e == nil {
		e = DefaultError
	}
	e(c, err)
}

func setRateLimitHeaders(c *gin.Context, context throttled.RateLimitResult) {
	w := c.Writer
	if v := context.Limit; v >= 0 {
		w.Header().Add("X-RateLimit-Limit", strconv.Itoa(v))
	}

	if v := context.Remaining; v >= 0 {
		w.Header().Add("X-RateLimit-Remaining", strconv.Itoa(v))
	}

	if v := context.ResetAfter; v >= 0 {
		vi := int(math.Ceil(v.Seconds()))
		w.Header().Add("X-RateLimit-Reset", strconv.Itoa(vi))
	}

	if v := context.RetryAfter; v >= 0 {
		vi := int(math.Ceil(v.Seconds()))
		w.Header().Add("Retry-After", strconv.Itoa(vi))
	}
}
