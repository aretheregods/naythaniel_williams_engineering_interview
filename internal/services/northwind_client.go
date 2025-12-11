package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/array/banking-api/internal/dto"
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

// CreateExternalAccount registers a new external account with the Northwind API.
// This is necessary to designate an account as a valid destination for transfers.
func (c *northwindClient) CreateExternalAccount(ctx context.Context, details *dto.NorthwindCreateAccountRequest) (*dto.NorthwindExternalAccountResponse, error) {
	requestBody, err := json.Marshal(details)
	if err != nil {
		return nil, fmt.Errorf("northwind client: failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/accounts", c.baseURL), bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("northwind client: failed to create request: %w", err)
	}

	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("northwind client: request to create external account failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("northwind client: create external account returned non-201 status: %d", resp.StatusCode)
	}

	var response dto.NorthwindExternalAccountResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("northwind client: failed to decode response body: %w", err)
	}

	return &response, nil
}

// InitiateTransfer starts a new transfer with the Northwind API.
func (c *northwindClient) InitiateTransfer(ctx context.Context, req *dto.NorthwindInitiateTransferRequest) (*dto.NorthwindInitiateTransferResponse, error) {
	requestBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("northwind client: failed to marshal transfer request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/transfers", c.baseURL), bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("northwind client: failed to create transfer request: %w", err)
	}

	httpReq.Header.Set("X-Api-Key", c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("northwind client: transfer request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("northwind client: initiate transfer returned non-201 status: %d", resp.StatusCode)
	}

	var response dto.NorthwindInitiateTransferResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("northwind client: failed to decode transfer response: %w", err)
	}

	return &response, nil
}
}
