# krakend-ratelimit

Krankend rate limitir implementation based on [GCRA algorithm](https://en.wikipedia.org/wiki/Generic_cell_rate_algorithm) using [github.com/throttled/throttled](github.com/throttled/throttled) implementation

## Implementation
It's based on a middleware to in order to intercept request before reaching any endpoint

## Configuration
The configuration should be added in the service extra_config.

### Example
```yml
{
  "extra_config": {
    "github.com/schibsted/krakend-ratelimit": {
      "enabled": true,
      "default": {
        "max_requests": 600,
        "burst_size": 5
      },
      "custom": {
        "kufar.com": {
            "max_requests": 60,
             "burst_size": 10
         },
           "corotos.com": {
             "max_requests": 40,
             "burst_size": 5
         }
      }
    }
  },
  "version": 2,
  "max_idle_connections": 250,
  "timeout": "3000ms",
  "read_timeout": "0s",
  "write_timeout": "0s",
  "idle_timeout": "0s",
  "read_header_timeout": "0s",
  "name": "Test",
  "endpoints": [
    {
      "endpoint": "/hello",
      "method": "GET",
      "backend": [
        {
          "url_pattern": "/hello",
          "host": [
            "http://localhost:8000"
          ]
        }
      ],
      "timeout": "1500ms",
      "max_rate": "10000"
    }
  ]
}
```

We can override default rate limit settings for specific sites. We use the issuer value
stored in JWT token as rate-limiter key.

### Building and using the rate limiter:
See an usage example [here](./gin_rate_limit_integration_test.go)

```go
func ApiGateway() {
	logger, err := logging.NewLogger("INFO", os.Stdout, "[KRAKEND]")
	if err != nil {
		panic(err)
	}

	parser := config.NewParser()
	serviceConfig, err := parser.Parse("./test.json")
	if err != nil {
		panic(err)
	}

  rateLimitCfg := ConfigGetter(serviceConfig.ExtraConfig).(RateLimitConfig)
  nodeCounter :=  DefaultNodeCounter()
	rateLimiter, err := GinRateLimit(rateLimitCfg, nodeCounter), logger)
	if err != nil {
		panic(err)
	}

  contextRateLimiter := GinContextRateLimit(rateLimiter, "SiteKey")
  middleware := contextRateLimiter.RateLimit()
	middlewares := []gin.HandlerFunc{ginMiddleware}

	routerFactory := kgin.NewFactory(
		kgin.Config{
			Engine:         gin.Default(),
			Middlewares:    middlewares,
			HandlerFactory: kgin.EndpointHandler,
			ProxyFactory:   proxy.DefaultFactory(logger),
			Logger:         logger,
		},
	)

	routerFactory.New().Run(serviceConfig)
}
```

```go
nodeCounter :=  DefaultNodeCounter()
```

The **NodeCounter** interface is used to determine what is the request imit per server based on the global limit.
That is, if only one instance of the server is running, then **NodeCounter** should return 1. In consecuence the request server limit and the request global limit will be the same.

Imagine now, we have two servers. So **NodeCounter** should return 2. In that case, server request limit should be the half of the request global limit.

There's an existing implementation which looks for more AWS EC2 instances:

```go
//Node counter for amazon EC2
func NewAwsNodeCounter(EC2 *EC2, autoScaling *AutoScaling,
	iid *ec2metadata.EC2InstanceIdentityDocument, logger logging.Logger) NodeCounter {
```

Now, we build the RateLimitier with the node counter and the configuration
```go
rateLimiter, err := GinRateLimit(rateLimitCfg, nodeCounter), logger)
```

Afterwards we create the **GinRateLimiter** with the **GinRateLimit** and we wirethe middleware with gin
```go
contextRateLimiter := GinContextRateLimit(rateLimiter, "SiteKey")
middleware := contextRateLimiter.RateLimit()
middlewares := []gin.HandlerFunc{ginMiddleware}
```


There's two classes of r**GinRateLimiter**, per issuer or per ip:
```go

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
```