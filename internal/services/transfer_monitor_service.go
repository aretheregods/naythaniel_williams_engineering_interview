package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/array/banking-api/internal/models"
	"github.com/array/banking-api/internal/repositories"
)

const (
	monitorBatchLimit = 100
)

type transferMonitorService struct {
	transferRepo    repositories.TransferRepositoryInterface
	accountService  AccountServiceInterface
	northwindClient NorthwindClientInterface
	webhookService  WebhookServiceInterface
	logger          *slog.Logger
}

func NewTransferMonitorService(
	transferRepo repositories.TransferRepositoryInterface,
	accountService AccountServiceInterface,
	northwindClient NorthwindClientInterface,
	webhookService WebhookServiceInterface,
) TransferMonitorServiceInterface {
	return &transferMonitorService{
		transferRepo:    transferRepo,
		accountService:  accountService,
		northwindClient: northwindClient,
		webhookService:  webhookService,
		logger:          slog.Default().With("service", "TransferMonitor"),
	}
}

// MonitorPendingTransfers checks the status of pending external transfers and updates them.
func (s *transferMonitorService) MonitorPendingTransfers(ctx context.Context) {
	s.logger.Info("starting check for pending external transfers")

	transfers, err := s.transferRepo.FindPendingExternal(monitorBatchLimit)
	if err != nil {
		s.logger.Error("failed to fetch pending external transfers", "error", err)
		return
	}

	if len(transfers) == 0 {
		s.logger.Info("no pending external transfers to monitor")
		return
	}

	s.logger.Info("found pending external transfers", "count", len(transfers))

	for _, transfer := range transfers {
		if transfer.ExternalTransferID == nil || *transfer.ExternalTransferID == "" {
			if time.Since(transfer.CreatedAt) > 5*time.Minute {
				s.logger.Warn("failing transfer that is missing external ID", "transfer_id", transfer.ID)
				if err := s.accountService.HandleFailedExternalTransfer(ctx, &transfer, "Transfer initiation failed; no external ID received."); err != nil {
					s.logger.Error("failed to handle internally failed transfer", "transfer_id", transfer.ID, "error", err)
				}
			}
			continue
		}

		s.checkAndUpdateTransferStatus(ctx, transfer)
	}
}

func (s *transferMonitorService) checkAndUpdateTransferStatus(ctx context.Context, transfer models.Transfer) {
	s.logger.Info("checking status for external transfer", "transfer_id", transfer.ID, "external_id", *transfer.ExternalTransferID)

	nwTransfer, err := s.northwindClient.GetTransfer(ctx, *transfer.ExternalTransferID)
	if err != nil {
		s.logger.Error("failed to get transfer status from Northwind", "transfer_id", transfer.ID, "external_id", *transfer.ExternalTransferID, "error", err)
		return
	}

	if nwTransfer.Status == transfer.Status {
		return // No change
	}

	s.logger.Info("status change detected for external transfer", "transfer_id", transfer.ID, "old_status", transfer.Status, "new_status", nwTransfer.Status)

	switch nwTransfer.Status {
	case "completed":
		transfer.Status = models.TransferStatusCompleted
		now := time.Now()
		transfer.CompletedAt = &now
		if err := s.transferRepo.Update(&transfer); err != nil {
			s.logger.Error("failed to update transfer status to completed", "transfer_id", transfer.ID, "error", err)
		} else {
			s.webhookService.QueueTransferNotification(ctx, &transfer)
		}
	case "failed":
		if err := s.accountService.HandleFailedExternalTransfer(ctx, &transfer, "Transfer failed at external bank."); err != nil {
			s.logger.Error("failed to handle failed external transfer", "transfer_id", transfer.ID, "error", err)
		}
	case "processing":
		transfer.Status = "processing"
		if err := s.transferRepo.Update(&transfer); err != nil {
			s.logger.Error("failed to update transfer status to processing", "transfer_id", transfer.ID, "error", err)
		}
	default:
		s.logger.Warn("unknown transfer status from Northwind", "status", nwTransfer.Status, "transfer_id", transfer.ID)
	}
}
