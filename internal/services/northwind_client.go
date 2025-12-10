package services

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const (
	defaultTimeout   = 10 * time.Second
	northwindBaseURL = "https://northwind.dev.array.io/api/v1"
)

// northwindClient implements the NorthwindClientInterface.
type northwindClient struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
}

// NewNorthwindClient creates a new authenticated client for the Northwind API.
func NewNorthwindClient(apiKey string) NorthwindClientInterface {
	return &northwindClient{
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		apiKey:  apiKey,
		baseURL: northwindBaseURL,
	}
}

// HealthCheck checks the health of the Northwind API.
func (c *northwindClient) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/health", c.baseURL), nil)
	if err != nil {
		return fmt.Errorf("northwind client: failed to create health check request: %w", err)
	}

	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("northwind client: health check request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("northwind client: health check returned non-200 status: %d", resp.StatusCode)
	}

	return nil
}
