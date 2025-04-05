# Stage 1: Build the Go binary
FROM golang:1.21-alpine AS builder
WORKDIR /app
# Copy Go module files and download dependencies first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download
# Copy the rest of the app src
COPY . .
# Build the Go application statically linked
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /k8s-loadtester .

# Stage 2: Create the final minimal image
FROM alpine:latest
# Copy the static binary from the builder stage
COPY --from=builder /k8s-loadtester /k8s-loadtester
# (Optional) Add certs if targeting HTTPS endpoints and using scratch/minimal base
# COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["/k8s-loadtester"]
CMD ["--help"] 