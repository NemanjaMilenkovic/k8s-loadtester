package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/NemanjaMilenkovic/k8s-loadtester/pkg/loadtester"
	"github.com/NemanjaMilenkovic/k8s-loadtester/pkg/metrics"
)

var (
	targetURL   string
	concurrency int
	duration    time.Duration
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "k8s-loadtester",
	Short: "A simple CLI tool to load test services, potentially in Kubernetes.",
	Long: `k8s-loadtester generates HTTP traffic to a specified URL
    with configurable concurrency and duration.

    Example:
    k8s-loadtester --url http://my-service.default.svc.cluster.local --concurrency 10 --duration 30s`,
	Run: func(cmd *cobra.Command, args []string) {
		// Basic validation
		if targetURL == "" {
			fmt.Println("Error: target URL cannot be empty")
			_ = cmd.Usage()
			os.Exit(1)
		}
		if concurrency <= 0 {
			fmt.Println("Error: concurrency must be positive")
			os.Exit(1)
		}
		if duration <= 0 {
			fmt.Println("Error: duration must be positive")
			os.Exit(1)
		}

		fmt.Printf("Starting load test...\n")
		fmt.Printf("  Target URL: %s\n", targetURL)
		fmt.Printf("  Concurrency: %d\n", concurrency)
		fmt.Printf("  Duration: %s\n", duration)
		fmt.Println("Press Ctrl+C to stop early.")
		fmt.Println("---")

		// context for graceful shutdown
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		// Initialize components
		mc := metrics.NewCollector()
		opts := loadtester.Options{
			URL:         targetURL,
			Concurrency: concurrency,
			Duration:    duration,
		}

		// Real-time reporting setup
		var wgReporter sync.WaitGroup
		wgReporter.Add(1)
		reporterCtx, cancelReporter := context.WithCancel(ctx)
		startTime := time.Now()

		go func() {
			defer wgReporter.Done()
			ticker := time.NewTicker(2 * time.Second) // Report every 2 seconds
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					currentDuration := time.Since(startTime)
					summary := mc.Summarize(currentDuration)
					fmt.Printf("[%s] Requests: %d | RPS: %.2f | Avg Latency: %.2fms | Errors: %d (%.2f%%)\n",
						currentDuration.Round(time.Second),
						summary.TotalRequests,
						summary.Throughput,
						summary.AvgLatencySuccessMs,
						summary.ErrorCount,
						summary.ErrorRatePercent,
					)
				case <-reporterCtx.Done():
					return
				}
			}
		}()

		// Run the load test
		loadtester.RunLoadTest(ctx, opts, mc)
		actualDuration := time.Since(startTime)

		// Stop the reporter and wait for it
		cancelReporter()
		wgReporter.Wait()

		// --- Final Report ---
		fmt.Println("---")
		fmt.Println("Load test finished.")
		summary := mc.Summarize(actualDuration)
		fmt.Println(summary.String())

		if ctx.Err() == context.Canceled {
			fmt.Println("Load test cancelled by user or signal.")
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&targetURL, "url", "u", "", "Target URL to test (required)")
	rootCmd.Flags().IntVarP(&concurrency, "concurrency", "c", 10, "Number of concurrent workers")
	rootCmd.Flags().DurationVarP(&duration, "duration", "d", 10*time.Second, "Duration of the test (e.g., 30s, 1m)")
} 