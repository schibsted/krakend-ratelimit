//
// Copyright 2011 - 2018 Schibsted Products & Technology AS.
// Licensed under the terms of the Apache 2.0 license. See LICENSE in the project root.
//
package ratelimit

import (
	"os"
	"testing"
	"time"

	"github.com/devopsfaith/krakend/config"
	"github.com/devopsfaith/krakend/logging"
	"github.com/devopsfaith/krakend/proxy"
	kgin "github.com/devopsfaith/krakend/router/gin"

	"github.com/gin-gonic/gin"

	"github.com/smartystreets/assertions"
	"github.com/tsenart/vegeta/lib"
)

func setup() {
	go DummyServer()
	go ApiGateway()
	time.Sleep(10 * time.Second)
}

func TestMain(m *testing.M) {
	setup()
	retCode := m.Run()
	os.Exit(retCode)
}

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
	rateLimiter, err := GinRateLimit(rateLimitCfg, DefaultNodeCounter(), logger)
	if err != nil {
		panic(err)
	}

	ginMiddleware := GinContextRateLimit(rateLimiter, "SiteKey").RateLimit()
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

func DummyServer() {
	r := gin.Default()
	r.GET("/hello", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "hello!",
		})
	})
	r.Run(":8000")
}

func TestRateLimitDoesNotThrottleRequest(t *testing.T) {
	rate := uint64(3) // per second
	duration := 4 * time.Second
	targeter := vegeta.NewStaticTargeter(vegeta.Target{
		Method: "GET",
		URL:    "http://localhost:8080/hello",
	})
	attacker := vegeta.NewAttacker()

	var metrics vegeta.Metrics
	for res := range attacker.Attack(targeter, rate, duration) {
		metrics.Add(res)
	}
	metrics.Close()
	equal := assertions.ShouldNotContainKey(metrics.StatusCodes, "429")
	if equal != "" {
		t.Errorf(equal)
	}
}

func TestRateLimitThrottlesRequests(t *testing.T) {
	rate := uint64(22) // per second
	duration := 4 * time.Second
	targeter := vegeta.NewStaticTargeter(vegeta.Target{
		Method: "GET",
		URL:    "http://localhost:8080/hello",
	})
	attacker := vegeta.NewAttacker()

	var metrics vegeta.Metrics
	for res := range attacker.Attack(targeter, rate, duration) {
		metrics.Add(res)
	}
	metrics.Close()
	equal := assertions.ShouldContainKey(metrics.StatusCodes, "429")
	if equal != "" {
		t.Errorf(equal)
	}
}
