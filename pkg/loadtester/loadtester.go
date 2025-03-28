package loadtester

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/NemanjaMilenkovic/k8s-loadtester/pkg/metrics"
)

type Options struct {
	URL         string
	Concurrency int
	Duration    time.Duration
	Client      *fasthttp.Client
}

// RunLoadTest executes the load test according to the provided options.
func RunLoadTest(ctx context.Context, opts Options, mc *metrics.Collector) {
	var wg sync.WaitGroup
	wg.Add(opts.Concurrency)

	client := opts.Client
	if client == nil {
		client = &fasthttp.Client{
			MaxConnsPerHost: opts.Concurrency,
			ReadTimeout:     10 * time.Second,
			WriteTimeout:    10 * time.Second,
		}
	}

	// Prepare request outside the loop for efficiency
	req := fasthttp.AcquireRequest()
	req.SetRequestURI(opts.URL)

	// Validate URL early
	uri := fasthttp.AcquireURI()
	err := uri.Parse(nil, []byte(opts.URL))
	fasthttp.ReleaseURI(uri)
	if err != nil {
		fmt.Printf("Error: Invalid target URL '%s': %v\n", opts.URL, err)
		fasthttp.ReleaseRequest(req)
		for i := 0; i < opts.Concurrency; i++ {
			wg.Done()
		}
		return
	}

	testCtx, cancel := context.WithTimeout(ctx, opts.Duration)
	defer cancel() // Ensure context is cancelled even if RunLoadTest exits early

	for i := 0; i < opts.Concurrency; i++ {
		go func() {
			defer wg.Done()
			resp := fasthttp.AcquireResponse()
			defer fasthttp.ReleaseResponse(resp) // Ensure response is released

			// Create a *copy* of the request for each goroutine if modifying it later
			// For simple GET, sharing might be okay, but copying is safer.
			localReq := fasthttp.AcquireRequest()
			req.CopyTo(localReq)
			defer fasthttp.ReleaseRequest(localReq)

			for {
				select {
				case <-testCtx.Done(): // Check if test duration expired or parent context cancelled
					return
				default:
					// Continue with request
				}

				startTime := time.Now()
				err := client.Do(localReq, resp) // Use the per-goroutine request copy
				latency := time.Since(startTime)

				if err != nil {
					// Network error or client timeout, etc.
					mc.RecordFailure(latency)
				} else {
					statusCode := resp.StatusCode()
					if statusCode >= 200 && statusCode < 400 { // Consider 2xx/3xx as success
						mc.RecordSuccess(latency)
					} else {
						mc.RecordFailure(latency)
					}
				}
				// Always clear the response body to reuse the response object
				resp.Reset()
			}
		}()
	}

	wg.Wait() // Wait for all workers to finish (either by duration or error)
	// Release the template request object
	fasthttp.ReleaseRequest(req)
} 