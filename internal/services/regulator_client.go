package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/array/banking-api/internal/config"
	"github.com/array/banking-api/internal/dto"
)

const regulatorTimeout = 15 * time.Second

type regulatorClient struct {
	httpClient *http.Client
	config     config.RegulatorConfig
}

// NewRegulatorClient creates a new client for sending webhooks to the regulator.
func NewRegulatorClient(cfg config.RegulatorConfig) RegulatorClientInterface {
	return &regulatorClient{
		httpClient: &http.Client{
			Timeout: regulatorTimeout,
		},
		config: cfg,
	}
}

// SendTransferNotification sends a webhook notification about a transfer's final status.
func (c *regulatorClient) SendTransferNotification(ctx context.Context, payload *dto.RegulatorNotificationPayload) error {
	if c.config.WebhookURL == "" {
		// If no URL is configured, we consider the notification "sent" to prevent queue blockage.
		// In a real-world scenario, this might trigger an alert.
		return nil
	}

	requestBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("regulator client: failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.WebhookURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("regulator client: failed to create request: %w", err)
	}

	if c.config.WebhookAPIKey != "" {
		req.Header.Set("X-Api-Key", c.config.WebhookAPIKey)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("regulator client: webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	// Regulators often return 200 OK or 202 Accepted. We'll treat any 2xx as success.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("regulator client: webhook returned non-2xx status: %d", resp.StatusCode)
	}

	return nil
}
