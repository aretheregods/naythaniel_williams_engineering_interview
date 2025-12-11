package services

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/array/banking-api/internal/config"
	"github.com/array/banking-api/internal/models"
	"github.com/array/banking-api/internal/repositories/repository_mocks"
	"github.com/array/banking-api/internal/services/service_mocks"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

type WebhookServiceTestSuite struct {
	suite.Suite
	ctrl            *gomock.Controller
	webhookRepo     *repository_mocks.MockWebhookNotificationRepositoryInterface
	regulatorClient *service_mocks.MockRegulatorClientInterface
	service         WebhookServiceInterface
}

func (s *WebhookServiceTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.webhookRepo = repository_mocks.NewMockWebhookNotificationRepositoryInterface(s.ctrl)
	s.regulatorClient = service_mocks.NewMockRegulatorClientInterface(s.ctrl)
	cfg := config.RegulatorConfig{WebhookURL: "https://example.com/webhook"}
	s.service = NewWebhookService(s.webhookRepo, s.regulatorClient, cfg)
}

func (s *WebhookServiceTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func TestWebhookServiceTestSuite(t *testing.T) {
	suite.Run(t, new(WebhookServiceTestSuite))
}

func (s *WebhookServiceTestSuite) TestQueueTransferNotification_Completed() {
	transfer := &models.Transfer{ID: uuid.New(), Status: models.TransferStatusCompleted}

	s.webhookRepo.EXPECT().Create(gomock.Any()).DoAndReturn(func(n *models.WebhookNotification) error {
		s.Equal(transfer.ID, n.TransferID)
		s.Equal(models.WebhookStatusPending, n.Status)
		return nil
	}).Times(1)

	err := s.service.QueueTransferNotification(context.Background(), transfer)
	s.NoError(err)
}

func (s *WebhookServiceTestSuite) TestQueueTransferNotification_Failed() {
	transfer := &models.Transfer{ID: uuid.New(), Status: models.TransferStatusFailed}

	s.webhookRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1)

	err := s.service.QueueTransferNotification(context.Background(), transfer)
	s.NoError(err)
}

func (s *WebhookServiceTestSuite) TestQueueTransferNotification_NonTerminalStatus() {
	transfer := &models.Transfer{ID: uuid.New(), Status: models.TransferStatusPending}

	// Create should NOT be called
	s.webhookRepo.EXPECT().Create(gomock.Any()).Times(0)

	err := s.service.QueueTransferNotification(context.Background(), transfer)
	s.NoError(err)
}

func (s *WebhookServiceTestSuite) TestProcessPendingWebhooks_Success() {
	notification := models.WebhookNotification{
		ID:         uuid.New(),
		TransferID: uuid.New(),
		Status:     models.WebhookStatusPending,
		Transfer: models.Transfer{
			ID:     uuid.New(),
			Status: models.TransferStatusCompleted,
			Amount: decimal.NewFromInt(100),
		},
	}

	s.webhookRepo.EXPECT().FindPending(gomock.Any()).Return([]models.WebhookNotification{notification}, nil)
	s.regulatorClient.EXPECT().SendTransferNotification(gomock.Any(), gomock.Any()).Return(http.StatusAccepted, `{"status":"received"}`, nil)
	s.webhookRepo.EXPECT().Update(gomock.Any()).DoAndReturn(func(n *models.WebhookNotification) error {
		s.Equal(models.WebhookStatusSent, n.Status)
		s.Equal(1, n.Attempts)
		s.NotNil(n.LastAttemptAt)
		s.Equal(http.StatusAccepted, *n.ResponseStatusCode)
		s.Contains(*n.ResponseBody, "received")
		return nil
	})

	s.service.ProcessPendingWebhooks(context.Background())
}

func (s *WebhookServiceTestSuite) TestProcessPendingWebhooks_FindPendingError() {
	s.webhookRepo.EXPECT().FindPending(gomock.Any()).Return(nil, errors.New("database is down"))

	// No other calls should be made if fetching fails
	s.regulatorClient.EXPECT().SendTransferNotification(gomock.Any(), gomock.Any()).Times(0)
	s.webhookRepo.EXPECT().Update(gomock.Any()).Times(0)

	// The service should log the error and return, not panic.
	s.service.ProcessPendingWebhooks(context.Background())
}

func (s *WebhookServiceTestSuite) TestProcessPendingWebhooks_RetryLogic() {
	notification := models.WebhookNotification{
		ID:         uuid.New(),
		TransferID: uuid.New(),
		Status:     models.WebhookStatusPending,
		Attempts:   0,
		Transfer: models.Transfer{
			ID:     uuid.New(),
			Status: models.TransferStatusFailed,
			Amount: decimal.NewFromInt(200),
		},
	}

	s.webhookRepo.EXPECT().FindPending(gomock.Any()).Return([]models.WebhookNotification{notification}, nil)
	s.regulatorClient.EXPECT().SendTransferNotification(gomock.Any(), gomock.Any()).Return(http.StatusInternalServerError, `{"error":"server unavailable"}`, errors.New("regulator is down"))
	s.webhookRepo.EXPECT().Update(gomock.Any()).DoAndReturn(func(n *models.WebhookNotification) error {
		s.Equal(models.WebhookStatusFailed, n.Status)
		s.Equal(1, n.Attempts)
		s.NotNil(n.LastAttemptAt)
		s.NotNil(n.NextAttemptAt)
		s.Equal(http.StatusInternalServerError, *n.ResponseStatusCode)
		s.Contains(*n.ResponseBody, "server unavailable")
		s.True(n.NextAttemptAt.After(time.Now())) // Next attempt is in the future
		return nil
	})

	s.service.ProcessPendingWebhooks(context.Background())
}

func (s *WebhookServiceTestSuite) TestProcessPendingWebhooks_NoPending() {
	s.webhookRepo.EXPECT().FindPending(gomock.Any()).Return([]models.WebhookNotification{}, nil)
	// No other calls should be made
	s.regulatorClient.EXPECT().SendTransferNotification(gomock.Any(), gomock.Any()).Times(0)
	s.webhookRepo.EXPECT().Update(gomock.Any()).Times(0)

	s.service.ProcessPendingWebhooks(context.Background())
}
