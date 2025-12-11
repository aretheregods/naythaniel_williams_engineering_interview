package services

import (
	"context"
	"fmt"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/array/banking-api/internal/dto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type NorthwindClientTestSuite struct {
	suite.Suite
}

func TestNorthwindClientTestSuite(t *testing.T) {
	suite.Run(t, new(NorthwindClientTestSuite))
}

func (s *NorthwindClientTestSuite) TestNewNorthwindClient() {
	apiKey := "my-secret-key"
	client := NewNorthwindClient(apiKey)
	s.NotNil(client)

	// Use type assertion to inspect internal fields
	c, ok := client.(*northwindClient)
	s.True(ok)
	s.Equal(apiKey, c.apiKey)
	s.Equal(northwindBaseURL, c.baseURL)
	s.NotNil(c.httpClient)
	s.Equal(defaultTimeout, c.httpClient.Timeout)
}

func (s *NorthwindClientTestSuite) TestHealthCheck_Success() {
	apiKey := "test-api-key"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/v1/health", r.URL.Path)
		s.Equal(apiKey, r.Header.Get("X-Api-Key"))
		s.Equal("application/json", r.Header.Get("Accept"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &northwindClient{
		httpClient: server.Client(),
		apiKey:     apiKey,
		baseURL:    server.URL + "/api/v1",
	}

	err := client.HealthCheck(context.Background())
	s.NoError(err)
}

func (s *NorthwindClientTestSuite) TestHealthCheck_APIError() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &northwindClient{
		httpClient: server.Client(),
		apiKey:     "any-key",
		baseURL:    server.URL + "/api/v1",
	}

	err := client.HealthCheck(context.Background())
	s.Error(err)
	s.Contains(err.Error(), fmt.Sprintf("northwind client: health check returned non-200 status: %d", http.StatusInternalServerError))
}

func (s *NorthwindClientTestSuite) TestHealthCheck_RequestFailure() {
	// Intentionally don't start a server to simulate a network error
	client := NewNorthwindClient("any-key")
	// Point to a non-existent server
	client.(*northwindClient).baseURL = "http://127.0.0.1:9999"

	err := client.HealthCheck(context.Background())
	s.Error(err)
	s.Contains(err.Error(), "northwind client: health check request failed")
}

func (s *NorthwindClientTestSuite) TestCreateExternalAccount_Success() {
	apiKey := "test-api-key"
	expectedNorthwindID := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/api/v1/accounts", r.URL.Path)
		s.Equal(apiKey, r.Header.Get("X-Api-Key"))
		s.Equal("application/json", r.Header.Get("Content-Type"))

		var reqBody dto.NorthwindCreateAccountRequest
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		s.NoError(err)
		s.Equal("123456789", reqBody.AccountNumber)
		s.Equal("987654321", reqBody.RoutingNumber)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(dto.NorthwindExternalAccountResponse{ID: expectedNorthwindID})
	}))
	defer server.Close()

	client := &northwindClient{
		httpClient: server.Client(),
		apiKey:     apiKey,
		baseURL:    server.URL + "/api/v1",
	}

	req := &dto.NorthwindCreateAccountRequest{
		AccountNumber: "123456789",
		RoutingNumber: "987654321",
		NameOnAccount: "John Doe",
		AccountType:   "checking",
		Currency:      "USD",
	}

	resp, err := client.CreateExternalAccount(context.Background(), req)
	s.NoError(err)
	s.NotNil(resp)
	s.Equal(expectedNorthwindID, resp.ID)
}

func (s *NorthwindClientTestSuite) TestCreateExternalAccount_APIError() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := &northwindClient{httpClient: server.Client(), apiKey: "any-key", baseURL: server.URL + "/api/v1"}
	req := &dto.NorthwindCreateAccountRequest{}

	resp, err := client.CreateExternalAccount(context.Background(), req)
	s.Error(err)
	s.Nil(resp)
	s.Contains(err.Error(), "northwind client: create external account returned non-201 status: 400")
}

func (s *NorthwindClientTestSuite) TestCreateExternalAccount_BadResponseBody() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("this is not json"))
	}))
	defer server.Close()

	client := &northwindClient{httpClient: server.Client(), apiKey: "any-key", baseURL: server.URL + "/api/v1"}
	req := &dto.NorthwindCreateAccountRequest{}

	resp, err := client.CreateExternalAccount(context.Background(), req)
	s.Error(err)
	s.Nil(resp)
	s.Contains(err.Error(), "northwind client: failed to decode response body")
}

func (s *NorthwindClientTestSuite) TestInitiateTransfer_Success() {
	apiKey := "test-api-key"
	expectedTransferID := "txn_12345"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/api/v1/transfers", r.URL.Path)
		s.Equal(apiKey, r.Header.Get("X-Api-Key"))

		var reqBody dto.NorthwindInitiateTransferRequest
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		s.NoError(err)
		s.Equal("12345", reqBody.SourceAccountID)
		s.Equal("debit", reqBody.Direction)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(dto.NorthwindInitiateTransferResponse{
			ID:     expectedTransferID,
			Status: "processing",
		})
	}))
	defer server.Close()

	client := &northwindClient{
		httpClient: server.Client(),
		apiKey:     apiKey,
		baseURL:    server.URL + "/api/v1",
	}

	req := &dto.NorthwindInitiateTransferRequest{
		SourceAccountID:      "12345",
		DestinationAccountID: uuid.New().String(),
		Amount:               "100.00",
		Direction:            "debit",
		TransferType:         "standard",
	}

	resp, err := client.InitiateTransfer(context.Background(), req)
	s.NoError(err)
	s.NotNil(resp)
	s.Equal(expectedTransferID, resp.ID)
	s.Equal("processing", resp.Status)
}
}
