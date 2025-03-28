package metrics

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMetricsCollector(t *testing.T) {
	mc := NewCollector()

	// Simulate some results concurrently
	var wg sync.WaitGroup
	numRequests := 100
	numErrors := 10
	latencySuccess := 50 * time.Millisecond
	latencyError := 100 * time.Millisecond

	startTime := time.Now()

	wg.Add(numRequests)
	for i := 0; i < numRequests; i++ {
		go func(isError bool) {
			defer wg.Done()
			if isError {
				mc.RecordFailure(latencyError)
			} else {
				mc.RecordSuccess(latencySuccess)
			}
		}(i < numErrors)
	}
	wg.Wait()

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	// Test basic counts
	assert.Equal(t, int64(numRequests-numErrors), mc.successCount)
	assert.Equal(t, int64(numErrors), mc.errorCount)
	assert.Equal(t, int64(numRequests), mc.TotalRequests())

	// Test total latency calculation (approximate due to concurrency)
	expectedTotalLatency := time.Duration(numRequests-numErrors)*latencySuccess + time.Duration(numErrors)*latencyError
	assert.InDelta(t, expectedTotalLatency, mc.totalLatencySuccess+mc.totalLatencyError, float64(50*time.Millisecond))

	// Test Summary generation
	summary := mc.Summarize(duration)

	assert.Equal(t, int64(numRequests), summary.TotalRequests)
	assert.Equal(t, int64(numRequests-numErrors), summary.SuccessCount)
	assert.Equal(t, int64(numErrors), summary.ErrorCount)
	assert.InDelta(t, float64(numRequests)/duration.Seconds(), summary.Throughput, 0.1)

	// Calculate expected average latency
	if summary.SuccessCount > 0 {
		expectedAvgSuccessLatency := float64(mc.totalLatencySuccess.Milliseconds()) / float64(summary.SuccessCount)
		assert.InDelta(t, expectedAvgSuccessLatency, summary.AvgLatencySuccessMs, 0.1)
	} else {
		assert.Equal(t, float64(0), summary.AvgLatencySuccessMs)
	}

	expectedErrorRate := float64(numErrors) / float64(numRequests) * 100.0
	assert.InDelta(t, expectedErrorRate, summary.ErrorRatePercent, 0.01)
}

func TestMetricsCollectorZeroRequests(t *testing.T) {
	mc := NewCollector()
	summary := mc.Summarize(10 * time.Second)

	assert.Equal(t, int64(0), summary.TotalRequests)
	assert.Equal(t, int64(0), summary.SuccessCount)
	assert.Equal(t, int64(0), summary.ErrorCount)
	assert.Equal(t, float64(0), summary.Throughput)
	assert.Equal(t, float64(0), summary.AvgLatencySuccessMs)
	assert.Equal(t, float64(0), summary.ErrorRatePercent)
}

func TestMetricsCollectorOnlyErrors(t *testing.T) {
	mc := NewCollector()
	mc.RecordFailure(100 * time.Millisecond)
	mc.RecordFailure(200 * time.Millisecond)
	summary := mc.Summarize(1 * time.Second)

	assert.Equal(t, int64(2), summary.TotalRequests)
	assert.Equal(t, int64(0), summary.SuccessCount)
	assert.Equal(t, int64(2), summary.ErrorCount)
	assert.Equal(t, float64(2.0), summary.Throughput)
	assert.Equal(t, float64(0), summary.AvgLatencySuccessMs)
	assert.Equal(t, float64(100.0), summary.ErrorRatePercent)
} 