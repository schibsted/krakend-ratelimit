package rate_limit

import (
	"fmt"

	"github.com/devopsfaith/krakend/logging"

	"os"
	"testing"
	"time"

	"gopkg.in/throttled/throttled.v2"
)

type rateLimitRequest struct {
	key      string
	quantity int
}

type rateLimitResponse struct {
	limited bool
	result  throttled.RateLimitResult
	err     error
}

type mockRateLimiter struct {
	requests map[rateLimitRequest]rateLimitResponse
}

type buildParams struct {
	reqsMinute int
	burstSize  int
}

type rateLimiterMockFactory struct {
	mocks map[buildParams]throttled.RateLimiter
}

func (r *mockRateLimiter) RateLimit(key string, quantity int) (bool, throttled.RateLimitResult, error) {
	req := rateLimitRequest{key: key, quantity: quantity}
	response, ok := r.requests[req]
	if !ok {
		return true, throttled.RateLimitResult{}, fmt.Errorf("Unexpected request key: %s", key)
	}
	return response.limited, response.result, response.err
}

func (r *mockRateLimiter) mockRequest(request rateLimitRequest, response rateLimitResponse) {
	if r.requests == nil {
		r.requests = make(map[rateLimitRequest]rateLimitResponse)
	}
	r.requests[request] = response
}

func (f *rateLimiterMockFactory) addMock(request buildParams, mock throttled.RateLimiter) {
	if f.mocks == nil {
		f.mocks = make(map[buildParams]throttled.RateLimiter)
	}
	f.mocks[request] = mock
}

func (f *rateLimiterMockFactory) Build(reqsMinute int, burstSize int) (throttled.RateLimiter, error) {
	req := buildParams{reqsMinute: reqsMinute, burstSize: burstSize}
	mock, ok := f.mocks[req]
	if !ok {
		return nil, fmt.Errorf("Unexpected build params reqsMinute: %d, burstSize %d", reqsMinute, burstSize)
	}
	return mock, nil
}

func TestBuildRateLimiterFromConfig(t *testing.T) {
	// default rate limit settings
	reqsMinute1 := 50
	burstSize1 := 15

	// rate limit settings for issuer 'siteKey'
	key := "siteKey"
	reqsMinute2 := 30
	burstSize2 := 10

	// Prepare APIGWConfig including GinRateLimit part
	rateLimitCfg := RateLimitConfig{
		Default: RateLimitSettings{
			MaxRequests: reqsMinute1,
			BurstSize:   burstSize1,
		},
		Custom: map[string]RateLimitSettings{
			key: {
				MaxRequests: reqsMinute2,
				BurstSize:   burstSize2,
			},
		},
	}

	// Build UpdatableRateLimiter
	nodes := 3
	logger, _ := logging.NewLogger("INFO", os.Stdout, "[KRAKEND]")
	rl := BuildRateLimiter(rateLimitCfg, func() int { return nodes }, logger).(*MultiRateLimiter)

	// Verify DefaultRateLimiter was properly created
	defaultRL := rl.defaultRL.(*ClusterAwareRateLimiter)
	settings1 := defaultRL.settings
	if settings1.reqsMinute != reqsMinute1 {
		t.Errorf("Unexpected RateLimit reqsMinute (got: %d, expected %d)", settings1.reqsMinute, reqsMinute1)
	}
	if settings1.burstSize != burstSize1 {
		t.Errorf("Unexpected RateLimit burstSize (got: %d, expected %d)", settings1.burstSize, burstSize1)
	}
	if defaultRL.Nodes() != nodes {
		t.Errorf("Unexpected RateLimit burstSize (got: %d, expected %d)", defaultRL.Nodes(), nodes)
	}

	// Verify CustomRateLimiter map was properly created
	if len(rl.customRL) != 1 {
		t.Errorf("Unexpected CustomRL size (got: %d, expected: 1)", len(rl.customRL))
	}
	customRL, ok := rl.customRL[key]
	if !ok {
		t.Errorf("Custom RL not found for siteKey: %s", key)
	}
	siteRL := customRL.(*ClusterAwareRateLimiter)
	settings2 := siteRL.settings
	if settings2.reqsMinute != reqsMinute2 {
		t.Errorf("Unexpected RateLimit reqsMinute (got: %d, expected %d)", settings2.reqsMinute, reqsMinute2)
	}
	if settings2.burstSize != burstSize2 {
		t.Errorf("Unexpected RateLimit burstSize (got: %d, expected %d)", settings2.burstSize, burstSize2)
	}
	if siteRL.Nodes() != nodes {
		t.Errorf("Unexpected RateLimit burstSize (got: %d, expected %d)", siteRL.Nodes(), nodes)
	}
}

func TestUpdateNodeCount(t *testing.T) {
	// cluster settings
	reqsMinute := 50
	burstSize := 15
	nodes := 2

	// node settings (depends on nodes)
	nodeReqsMinute1 := 25
	nodeBurstSize1 := 8
	nodeReqsMinute2 := 17
	nodeBurstSize2 := 5

	// mock rate limit factory (we expect 2 calls to Build method)
	factory := &rateLimiterMockFactory{}
	rateLimtierSettings1 := buildParams{reqsMinute: nodeReqsMinute1, burstSize: nodeBurstSize1}
	rateLimtierSettings2 := buildParams{reqsMinute: nodeReqsMinute2, burstSize: nodeBurstSize2}
	factory.addMock(rateLimtierSettings1, &mockRateLimiter{})
	factory.addMock(rateLimtierSettings2, &mockRateLimiter{})

	// Build ClusterAwareRateLimiter (first call to RateLimitFactory.Build(...))
	clusterSettings := RateLimiterSettings{reqsMinute: reqsMinute, burstSize: burstSize}
	updatableRL, err := NewClusterAwareRateLimiter(factory, nodes, clusterSettings)
	if err != nil {
		t.Errorf("Error building ClusterAwareRateLimiter: %s", err.Error())
	}
	clusterRL := updatableRL.(*ClusterAwareRateLimiter)
	if clusterRL.nodes != nodes {
		t.Errorf("Unexpected cluster nodes (got: %d, expected %d)", clusterRL.nodes, nodes)
	}
	// Update node count (second call to RateLimitFactory.Build(...))
	newNodeCount := 3
	updatableRL.UpdateNodeCount(newNodeCount)
	clusterRL = updatableRL.(*ClusterAwareRateLimiter)
	if clusterRL.nodes != newNodeCount {
		t.Errorf("Unexpected cluster nodes (got: %d, expected %d)", clusterRL.nodes, newNodeCount)
	}
}

func TestClusterRateLimit(t *testing.T) {
	// mock RateLimit request
	key := "myKey"
	quantity := 2
	limited := false
	request := rateLimitRequest{key: key, quantity: quantity}

	fakeResult := getFakeRateLimitResult(1, 2, 3, 4)
	response := rateLimitResponse{limited: limited, result: fakeResult}
	mock := mockRateLimiter{}
	mock.mockRequest(request, response)

	// mock rate limit factory
	factory := &rateLimiterMockFactory{}
	nodeSettings := buildParams{reqsMinute: 20, burstSize: 4}
	factory.addMock(nodeSettings, &mock)

	// Build ClusterAwareRateLimit
	clusterSettings := RateLimiterSettings{reqsMinute: 60, burstSize: 10}
	updatableRL, err := NewClusterAwareRateLimiter(factory, 3, clusterSettings)

	// Call RateLimit() and check results
	throttled, result, err := updatableRL.RateLimit(key, quantity)
	if err != nil {
		t.Errorf("TestRateLimit: request failed %s", err.Error())
	}
	if fakeResult != result {
		t.Errorf("TestRateLimit: unexpected response (expected: %s, got %s)", fakeResult, result)
	}
	if limited != throttled {
		t.Errorf("TestRateLimit: unexpected limited value (expected: %s, got %s)", limited, throttled)
	}
}

func TestMultiRateLimit(t *testing.T) {
	// mock request using DefaultRateLimiter
	defaultKey := "other"
	defaultQuantity := 1
	defaultLimited := true
	defaultRequest := rateLimitRequest{key: defaultKey, quantity: defaultQuantity}

	defaultFakeResult := getFakeRateLimitResult(5, 6, 7, 8)
	defaultResponse := rateLimitResponse{limited: defaultLimited, result: defaultFakeResult}
	defaultMock := mockRateLimiter{}
	defaultMock.mockRequest(defaultRequest, defaultResponse)

	// mock request using customized RateLimiter
	siteKey := "siteKey"
	siteQuantity := 2
	siteLimited := false
	siteRequest := rateLimitRequest{key: siteKey, quantity: siteQuantity}

	siteFakeResult := getFakeRateLimitResult(1, 2, 3, 4)
	siteResponse := rateLimitResponse{limited: siteLimited, result: siteFakeResult}
	siteMock := mockRateLimiter{}
	siteMock.mockRequest(siteRequest, siteResponse)

	// mock rate limit factory
	factory := &rateLimiterMockFactory{}
	siteNodeSettings := buildParams{reqsMinute: 20, burstSize: 4}
	defaultNodeSettings := buildParams{reqsMinute: 200, burstSize: 34}
	factory.addMock(siteNodeSettings, &siteMock)
	factory.addMock(defaultNodeSettings, &defaultMock)

	// Init/build MultiRateLimiter
	siteSettings := RateLimiterSettings{reqsMinute: 60, burstSize: 10}
	customSettings := map[string]RateLimiterSettings{siteKey: siteSettings}
	defaultSettings := RateLimiterSettings{reqsMinute: 600, burstSize: 100}
	multiRL, err := NewMultiRateLimiter(factory, 3, defaultSettings, customSettings)
	if err != nil {
		t.Errorf("TestRateLimit: build failed %s", err.Error())
	}

	// Use RateLimit method for site using default RateLimit
	throttled, result, err := multiRL.RateLimit(defaultKey, defaultQuantity)
	if err != nil {
		t.Errorf("TestRateLimit: request failed %s", err.Error())
	}
	if defaultFakeResult != result {
		t.Errorf("TestRateLimit: unexpected response (expected: %s, got %s)", defaultFakeResult, result)
	}
	if defaultLimited != throttled {
		t.Errorf("TestRateLimit: unexpected limited value (expected: %s, got %s)", defaultLimited, throttled)
	}

	// Use RateLimit method for customized site
	throttled, result, err = multiRL.RateLimit(siteKey, siteQuantity)
	if err != nil {
		t.Errorf("TestRateLimit: request failed %s", err.Error())
	}
	if siteFakeResult != result {
		t.Errorf("TestRateLimit: unexpected response (expected: %s, got %s)", siteFakeResult, result)
	}
	if siteLimited != throttled {
		t.Errorf("TestRateLimit: unexpected limited value (expected: %s, got %s)", siteLimited, throttled)
	}
}

func getFakeRateLimitResult(limit int, remaining int, reset int, retry int) throttled.RateLimitResult {
	return throttled.RateLimitResult{
		Limit:      limit,
		Remaining:  remaining,
		ResetAfter: time.Second * time.Duration(reset),
		RetryAfter: time.Second * time.Duration(retry),
	}
}
