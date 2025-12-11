package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/array/banking-api/internal/dto"
	"github.com/array/banking-api/internal/models"
	"github.com/array/banking-api/internal/repositories/repository_mocks"
	"github.com/array/banking-api/internal/services/service_mocks"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type TransferMonitorServiceTestSuite struct {
	suite.Suite
	ctrl            *gomock.Controller
	transferRepo    *repository_mocks.MockTransferRepositoryInterface
	accountService  *service_mocks.MockAccountServiceInterface
	northwindClient *service_mocks.MockNorthwindClientInterface
	webhookService  *service_mocks.MockWebhookServiceInterface
	service         TransferMonitorServiceInterface
}

func (s *TransferMonitorServiceTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.transferRepo = repository_mocks.NewMockTransferRepositoryInterface(s.ctrl)
	s.accountService = service_mocks.NewMockAccountServiceInterface(s.ctrl)
	s.northwindClient = service_mocks.NewMockNorthwindClientInterface(s.ctrl)
	s.webhookService = service_mocks.NewMockWebhookServiceInterface(s.ctrl)
	s.service = NewTransferMonitorService(s.transferRepo, s.accountService, s.northwindClient, s.webhookService)
}

func (s *TransferMonitorServiceTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func TestTransferMonitorServiceTestSuite(t *testing.T) {
	suite.Run(t, new(TransferMonitorServiceTestSuite))
}

func (s *TransferMonitorServiceTestSuite) TestMonitorPendingTransfers_Completed() {
	externalID := "nw_txn_completed"
	transfer := models.Transfer{
		ID:                 uuid.New(),
		ExternalTransferID: &externalID,
		Status:             "processing",
	}

	s.transferRepo.EXPECT().FindPendingExternal(gomock.Any()).Return([]models.Transfer{transfer}, nil)
	s.northwindClient.EXPECT().GetTransfer(gomock.Any(), externalID).Return(&dto.NorthwindGetTransferResponse{
		ID:     externalID,
		Status: "completed",
	}, nil)
	s.transferRepo.EXPECT().Update(gomock.Any()).DoAndReturn(func(t *models.Transfer) error {
		s.Equal(models.TransferStatusCompleted, t.Status)
		s.NotNil(t.CompletedAt)
		return nil
	})
	s.webhookService.EXPECT().QueueTransferNotification(gomock.Any(), gomock.Any()).Return(nil)

	s.service.MonitorPendingTransfers(context.Background())
}

func (s *TransferMonitorServiceTestSuite) TestMonitorPendingTransfers_Failed() {
	externalID := "nw_txn_failed"
	transfer := models.Transfer{
		ID:                 uuid.New(),
		ExternalTransferID: &externalID,
		Status:             "processing",
	}

	s.transferRepo.EXPECT().FindPendingExternal(gomock.Any()).Return([]models.Transfer{transfer}, nil)
	s.northwindClient.EXPECT().GetTransfer(gomock.Any(), externalID).Return(&dto.NorthwindGetTransferResponse{
		ID:     externalID,
		Status: "failed",
	}, nil)
	s.accountService.EXPECT().HandleFailedExternalTransfer(gomock.Any(), gomock.Any(), "Transfer failed at external bank.").Return(nil)

	s.service.MonitorPendingTransfers(context.Background())
}

func (s *TransferMonitorServiceTestSuite) TestMonitorPendingTransfers_StillProcessing() {
	externalID := "nw_txn_processing"
	transfer := models.Transfer{
		ID:                 uuid.New(),
		ExternalTransferID: &externalID,
		Status:             models.TransferStatusPending, // Our initial status
	}

	s.transferRepo.EXPECT().FindPendingExternal(gomock.Any()).Return([]models.Transfer{transfer}, nil)
	s.northwindClient.EXPECT().GetTransfer(gomock.Any(), externalID).Return(&dto.NorthwindGetTransferResponse{
		ID:     externalID,
		Status: "processing", // Northwind's processing status
	}, nil)
	s.transferRepo.EXPECT().Update(gomock.Any()).DoAndReturn(func(t *models.Transfer) error {
		s.Equal("processing", t.Status)
		return nil
	})

	s.service.MonitorPendingTransfers(context.Background())
}

func (s *TransferMonitorServiceTestSuite) TestMonitorPendingTransfers_StuckWithoutExternalID() {
	transfer := models.Transfer{
		ID:                 uuid.New(),
		ExternalTransferID: nil, // No external ID
		Status:             models.TransferStatusPending,
		CreatedAt:          time.Now().Add(-10 * time.Minute), // Older than 5 minutes
	}

	s.transferRepo.EXPECT().FindPendingExternal(gomock.Any()).Return([]models.Transfer{transfer}, nil)
	s.accountService.EXPECT().HandleFailedExternalTransfer(gomock.Any(), gomock.Any(), "Transfer initiation failed; no external ID received.").Return(nil)

	s.service.MonitorPendingTransfers(context.Background())
}

func (s *TransferMonitorServiceTestSuite) TestMonitorPendingTransfers_NoPending() {
	s.transferRepo.EXPECT().FindPendingExternal(gomock.Any()).Return([]models.Transfer{}, nil)
	// No other calls should be made
	s.northwindClient.EXPECT().GetTransfer(gomock.Any(), gomock.Any()).Times(0)
	s.accountService.EXPECT().HandleFailedExternalTransfer(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	s.transferRepo.EXPECT().Update(gomock.Any()).Times(0)

	s.service.MonitorPendingTransfers(context.Background())
}

func (s *TransferMonitorServiceTestSuite) TestMonitorPendingTransfers_NorthwindAPIFailure() {
	externalID := "nw_txn_api_fail"
	transfer := models.Transfer{
		ID:                 uuid.New(),
		ExternalTransferID: &externalID,
		Status:             "processing",
	}

	s.transferRepo.EXPECT().FindPendingExternal(gomock.Any()).Return([]models.Transfer{transfer}, nil)
	s.northwindClient.EXPECT().GetTransfer(gomock.Any(), externalID).Return(nil, errors.New("API is down"))
	// No status updates should happen
	s.accountService.EXPECT().HandleFailedExternalTransfer(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	s.transferRepo.EXPECT().Update(gomock.Any()).Times(0)

	s.service.MonitorPendingTransfers(context.Background())
}
