package loadtester

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"

	"github.com/NemanjaMilenkovic/k8s-loadtester/pkg/metrics"
)

// Mock HTTP Server Handler
func createMockHandler(statusCode int, responseBody string, delay time.Duration) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		time.Sleep(delay) // Simulate network latency or processing time
		ctx.SetStatusCode(statusCode)
		ctx.SetBodyString(responseBody)
	}
}

func runMockServer(t *testing.T, handler fasthttp.RequestHandler) (net.Listener, string) {
	ln := fasthttputil.NewInmemoryListener()
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		err := fasthttp.Serve(ln, handler)
		if err != nil {
			// fmt.Printf("Mock server error: %v\n", err)
		}
	}()

	return ln, "http://localhost: N/A (in-memory)"
}

func TestRunLoadTest_SuccessfulRequests(t *testing.T) {
	handler := createMockHandler(http.StatusOK, "OK", 10*time.Millisecond)
	ln, _ := runMockServer(t, handler)

	mc := metrics.NewCollector()
	opts := Options{
		URL:         "http://dummy-url",
		Concurrency: 5,
		Duration:    200 * time.Millisecond,
		Client: &fasthttp.Client{
			Dial: func(addr string) (net.Conn, error) {
				return ln.Dial()
			},
		},
	}

	startTime := time.Now()
	RunLoadTest(context.Background(), opts, mc)
	testDuration := time.Since(startTime)

	summary := mc.Summarize(testDuration)

	assert.Greater(t, summary.TotalRequests, int64(0), "Should have made some requests")
	assert.Equal(t, summary.TotalRequests, summary.SuccessCount, "All requests should succeed")
	assert.Equal(t, int64(0), summary.ErrorCount, "No errors expected")
	assert.Greater(t, summary.AvgLatencySuccessMs, float64(0))
	assert.LessOrEqual(t, testDuration, opts.Duration+50*time.Millisecond, "Test duration should be close to requested")
}

func TestRunLoadTest_WithErrorRequests(t *testing.T) {
	var requestCount int32
	handler := func(ctx *fasthttp.RequestCtx) {
		count := atomic.AddInt32(&requestCount, 1)
		if count%2 == 0 { // Fail every second request
			ctx.SetStatusCode(http.StatusInternalServerError)
			ctx.SetBodyString("Internal Error")
		} else {
			ctx.SetStatusCode(http.StatusOK)
			ctx.SetBodyString("OK")
		}
		time.Sleep(5 * time.Millisecond)
	}
	ln, _ := runMockServer(t, handler)

	mc := metrics.NewCollector()
	opts := Options{
		URL:         "http://dummy-url",
		Concurrency: 2,
		Duration:    200 * time.Millisecond,
		Client: &fasthttp.Client{
			Dial: func(addr string) (net.Conn, error) { return ln.Dial() },
		},
	}

	startTime := time.Now()
	RunLoadTest(context.Background(), opts, mc)
	testDuration := time.Since(startTime)

	summary := mc.Summarize(testDuration)

	assert.Greater(t, summary.TotalRequests, int64(0))
	assert.Greater(t, summary.SuccessCount, int64(0))
	assert.Greater(t, summary.ErrorCount, int64(0))
	assert.InDelta(t, summary.SuccessCount, summary.ErrorCount, float64(summary.TotalRequests)*0.5)
	assert.LessOrEqual(t, testDuration, opts.Duration+50*time.Millisecond)
}

func TestRunLoadTest_ContextCancellation(t *testing.T) {
	handler := createMockHandler(http.StatusOK, "OK", 50*time.Millisecond)
	ln, _ := runMockServer(t, handler)

	mc := metrics.NewCollector()
	opts := Options{
		URL:         "http://dummy-url",
		Concurrency: 2,
		Duration:    5 * time.Second,
		 Client: &fasthttp.Client{
			Dial: func(addr string) (net.Conn, error) { return ln.Dial() },
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	startTime := time.Now()
	RunLoadTest(ctx, opts, mc)
	testDuration := time.Since(startTime)

	summary := mc.Summarize(testDuration)

	assert.Less(t, testDuration, 500*time.Millisecond, "Test should stop quickly due to context cancellation")
	assert.GreaterOrEqual(t, summary.TotalRequests, int64(0))
}

func TestRunLoadTest_InvalidURL(t *testing.T) {
	mc := metrics.NewCollector()
	opts := Options{
		URL:         ":invalid-url:",
		Concurrency: 1,
		Duration:    50 * time.Millisecond,
	}

	ctx := context.Background()
	RunLoadTest(ctx, opts, mc)
	summary := mc.Summarize(opts.Duration)

	assert.Equal(t, int64(0), summary.SuccessCount)
} 