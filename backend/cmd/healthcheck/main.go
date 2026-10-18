package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

// healthcheck is a minimal binary used as the Docker HEALTHCHECK command.
// It sends an HTTP GET to the liveness endpoint and exits 0 on success, 1 on failure.
// This exists because distroless images have no curl or wget.
//
// Usage: healthcheck [url]
// Default URL: http://localhost:8080/health/live
func main() {
	url := "http://localhost:8080/health/live"
	if len(os.Args) > 1 {
		url = os.Args[1]
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "healthcheck failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "healthcheck unhealthy: status %d\n", resp.StatusCode)
		os.Exit(1)
	}

	os.Exit(0)
}
