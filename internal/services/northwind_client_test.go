package services

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

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
