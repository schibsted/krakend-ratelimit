// Package containing all rate-limit logic
//
// Internally we use GCRA rate limiters from throttled package (https://github.com/throttled/throttled)
//
// Krakend configuration example:
//
//     {
//       "endpoints: [
//         ...
//       ],
//       "rate_limit": {
//         "default": {
//            max_requests": 20,
//           "burst_size": 5
//          ,
//         "custom": {
//           "kufar.com": {
//             "max_requests": 60,
//             "burst_size": 10
//           },
//           "corotos.com": {
//             "max_requests": 40,
//             "burst_size": 5
//           }
//         }
//       }
//     }
//
// We can override default rate limit settings for specific sites. We use the issuer value
// stored in JWT token as rate-limiter key.
//
// Building and using the rate limiter:
//
//     config := apigw_config.NewGatewayConfig(...)
//     nodes := 3
//     rateLimiter := rate_limit.BuildRateLimiter(config, nodes)
//
//     issuer := "www.mysite.com"
//     weight := 1
//     limited, context, err := rateLimiter.GinRateLimit(issuer, weight)
//
//     rateLimiter.UpdateNodeCount(6)
//
//
// BuildRateLimiter returns a MultiRateLimiter object as a result, which implements the interface:
//
//     type UpdatableClusterRateLimiter interface {
//          GinRateLimit(key string, quantity int) (bool, throttled.RateLimitResult, error)
//          UpdateNodeCount(nodes int) error
//          Nodes() int
//     }
//
// Anytime we call UpdateNodeCount, limits per node change so we restart all internal RateLimiters.
//
// Internal RateLimiters are instances of DynamicRateLimiter which implements interface:
//
//     type UpdatableRateLimiter interface {
//          Update(settings RateLimiterSettings) error
//          GinRateLimit(key string, quantity int) (bool, throttled.RateLimitResult, error)
//     }
//
//     type RateLimiterSettings struct {
//          reqsMinute int
//          burstSize  int
//     }
//
package rate_limit
