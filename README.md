# k8s-loadtester

A simple CLI tool written in Go to generate HTTP load against a target URL.
Intended to be easily runnable locally or within a Kubernetes cluster.

## Features

- Configurable target URL
- Configurable concurrency level (number of parallel workers)
- Configurable test duration
- Real-time progress reporting (Requests/sec, Latency, Errors)
- Final summary report

## Prerequisites

- Go (version 1.21 or later recommended)

## Building

1.  **Ensure Dependencies:** Make sure all Go module dependencies are downloaded and the `go.sum` file is up-to-date:
    ```bash
    go mod tidy
    ```
2.  **Build the Binary:** Compile the Go source code into an executable:
    ```bash
    go build -o k8s-loadtester .
    ```
    This will create an executable file named `k8s-loadtester` in the current directory.

## Running

Execute the compiled binary, providing the required target URL and optional concurrency/duration flags:

```bash
./k8s-loadtester --url <TARGET_URL> [flags]
```

**Required Flags:**

- `--url` or `-u`: The target URL to send HTTP GET requests to (e.g., `http://my-service.com`, `http://localhost:8080`).

**Optional Flags:**

- `--concurrency` or `-c`: Number of concurrent workers making requests (default: `10`).
- `--duration` or `-d`: Duration of the load test (e.g., `30s`, `1m`, `5m30s`) (default: `10s`).
- `--help`: Show usage information.

**Example:**

```bash
# Test example.com with 20 concurrent users for 1 minute
./k8s-loadtester --url http://example.com --concurrency 20 --duration 1m
```

## Docker

A `Dockerfile` is included to build a container image for the load tester.

```bash
# Build the image (replace 'your-repo/k8s-loadtester' with your desired image name)
docker build -t your-repo/k8s-loadtester:latest .

# Run the container (replace URL and flags as needed)
docker run --rm your-repo/k8s-loadtester:latest --url http://your-target-service --concurrency 10 --duration 30s
```

### Kubernetes

The `kubernetes/` directory contains example manifests:

- `job.yaml`: Run a one-off load test as a Kubernetes Job.
- `cronjob.yaml`: Run the load test on a schedule as a Kubernetes CronJob.

**Important:** Remember to update the `image:` field in the Kubernetes manifests to point to the container image you built and pushed to a registry accessible by your cluster. Also, update the target `--url` within the `args` section.

### Test on simple local Python HTTP server:

### Start the server:

```bash
python -m http.server 8000
```

### Run load test:

```bash
./k8s-loadtester --url http://localhost:8000 --concurrency 5 --duration 10s
```

Output:

```bash
Starting load test...
  Target URL: http://localhost:8000
  Concurrency: 5
  Duration: 10s
Press Ctrl+C to stop early.
---
[2s] Requests: 4910 | RPS: 2454.82 | Avg Latency: 2.04ms | Errors: 0 (0.00%)
[4s] Requests: 10610 | RPS: 2652.40 | Avg Latency: 1.88ms | Errors: 0 (0.00%)
[6s] Requests: 15330 | RPS: 2554.95 | Avg Latency: 1.95ms | Errors: 0 (0.00%)
[8s] Requests: 16302 | RPS: 2037.48 | Avg Latency: 2.04ms | Errors: 0 (0.00%)
[10s] Requests: 16308 | RPS: 1630.61 | Avg Latency: 2.04ms | Errors: 5 (0.03%)
[12s] Requests: 16308 | RPS: 1358.91 | Avg Latency: 2.04ms | Errors: 5 (0.03%)
---
Load test finished.
--- Summary ---
Total Duration:      12.649s
Total Requests:      16313
Successful Requests: 16303
Failed Requests:     10
Throughput (req/s):  1289.66
Avg Latency (ms):    2.04
Error Rate:          0.06%
```
