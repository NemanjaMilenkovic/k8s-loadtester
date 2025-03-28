package metrics

import (
	"fmt"
	"sync"
	"time"
)

// Summary holds the calculated metrics summary.
type Summary struct {
	TotalRequests       int64
	SuccessCount        int64
	ErrorCount          int64
	TotalDuration       time.Duration
	Throughput          float64
	AvgLatencySuccessMs float64
	ErrorRatePercent    float64
}

// Collector handles metrics collection safely from multiple goroutines.
type Collector struct {
	mu                sync.Mutex
	successCount      int64
	errorCount        int64
	totalLatencySuccess time.Duration
	totalLatencyError time.Duration
}

// NewCollector creates a new metrics collector.
func NewCollector() *Collector {
	return &Collector{}
}

// RecordSuccess records a successful request.
func (c *Collector) RecordSuccess(latency time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.successCount++
	c.totalLatencySuccess += latency
}

// RecordFailure records a failed request.
func (c *Collector) RecordFailure(latency time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errorCount++
	c.totalLatencyError += latency
}

// TotalRequests returns the total number of requests recorded.
func (c *Collector) TotalRequests() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.successCount + c.errorCount
}

// Summarize calculates and returns the final metrics summary.
func (c *Collector) Summarize(totalDuration time.Duration) Summary {
	c.mu.Lock()
	defer c.mu.Unlock()

	totalRequests := c.successCount + c.errorCount
	var throughput float64
	if totalDuration.Seconds() > 0 {
		throughput = float64(totalRequests) / totalDuration.Seconds()
	}

	var avgLatencySuccessMs float64
	if c.successCount > 0 {
		avgLatencySuccessMs = float64(c.totalLatencySuccess.Milliseconds()) / float64(c.successCount)
	}

	var errorRatePercent float64
	if totalRequests > 0 {
		errorRatePercent = float64(c.errorCount) / float64(totalRequests) * 100.0
	}

	return Summary{
		TotalRequests:       totalRequests,
		SuccessCount:        c.successCount,
		ErrorCount:          c.errorCount,
		TotalDuration:       totalDuration,
		Throughput:          throughput,
		AvgLatencySuccessMs: avgLatencySuccessMs,
		ErrorRatePercent:    errorRatePercent,
	}
}

// String provides a formatted string representation of the summary.
func (s Summary) String() string {
	return fmt.Sprintf(
		"--- Summary ---\n"+
			"Total Duration:      %v\n"+
			"Total Requests:      %d\n"+
			"Successful Requests: %d\n"+
			"Failed Requests:     %d\n"+
			"Throughput (req/s):  %.2f\n"+
			"Avg Latency (ms):    %.2f\n"+
			"Error Rate:          %.2f%%\n",
		s.TotalDuration.Round(time.Millisecond),
		s.TotalRequests,
		s.SuccessCount,
		s.ErrorCount,
		s.Throughput,
		s.AvgLatencySuccessMs,
		s.ErrorRatePercent,
	)
} 