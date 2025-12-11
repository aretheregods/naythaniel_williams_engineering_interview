package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/array/banking-api/internal/config"
	"github.com/array/banking-api/internal/dto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type RegulatorClientTestSuite struct {
	suite.Suite
}

func TestRegulatorClientTestSuite(t *testing.T) {
	suite.Run(t, new(RegulatorClientTestSuite))
}

func (s *RegulatorClientTestSuite) TestSendTransferNotification_Success() {
	apiKey := "regulator-secret-key"
	var receivedPayload dto.RegulatorNotificationPayload

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal(apiKey, r.Header.Get("X-Api-Key"))
		s.Equal("application/json", r.Header.Get("Content-Type"))

		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		s.NoError(err)

		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	cfg := config.RegulatorConfig{
		WebhookURL:    server.URL,
		WebhookAPIKey: apiKey,
	}
	client := NewRegulatorClient(cfg)

	payload := &dto.RegulatorNotificationPayload{
		TransferID: uuid.New(),
		Status:     "completed",
		Amount:     "123.45",
	}

	err := client.SendTransferNotification(context.Background(), payload)
	s.NoError(err)
	s.Equal(payload.TransferID, receivedPayload.TransferID)
	s.Equal(payload.Status, receivedPayload.Status)
}

func (s *RegulatorClientTestSuite) TestSendTransferNotification_APIFailure() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := config.RegulatorConfig{WebhookURL: server.URL}
	client := NewRegulatorClient(cfg)
	payload := &dto.RegulatorNotificationPayload{TransferID: uuid.New()}

	err := client.SendTransferNotification(context.Background(), payload)
	s.Error(err)
	s.Contains(err.Error(), "regulator client: webhook returned non-2xx status: 500")
}

func (s *RegulatorClientTestSuite) TestSendTransferNotification_NoURLConfigured() {
	cfg := config.RegulatorConfig{WebhookURL: ""} // No URL
	client := NewRegulatorClient(cfg)
	payload := &dto.RegulatorNotificationPayload{TransferID: uuid.New()}

	err := client.SendTransferNotification(context.Background(), payload)
	s.NoError(err) // Should return nil and not block the queue
}

func (s *RegulatorClientTestSuite) TestSendTransferNotification_NetworkError() {
	// Point to a non-existent server
	cfg := config.RegulatorConfig{WebhookURL: "http://127.0.0.1:9999"}
	client := NewRegulatorClient(cfg)
	payload := &dto.RegulatorNotificationPayload{TransferID: uuid.New()}

	err := client.SendTransferNotification(context.Background(), payload)
	s.Error(err)
	s.Contains(err.Error(), "regulator client: webhook request failed")
}
