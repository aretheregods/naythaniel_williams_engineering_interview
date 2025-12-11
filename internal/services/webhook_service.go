package services

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/array/banking-api/internal/config"
	"github.com/array/banking-api/internal/dto"
	"github.com/array/banking-api/internal/models"
	"github.com/array/banking-api/internal/repositories"
)

const (
	webhookBatchLimit    = 100
	initialBackoffPeriod = 1 * time.Minute
)

type webhookService struct {
	webhookRepo     repositories.WebhookNotificationRepositoryInterface
	regulatorClient RegulatorClientInterface
	regulatorConfig config.RegulatorConfig
	logger          *slog.Logger
}

func NewWebhookService(
	webhookRepo repositories.WebhookNotificationRepositoryInterface,
	regulatorClient RegulatorClientInterface,
	regulatorConfig config.RegulatorConfig,
) WebhookServiceInterface {
	return &webhookService{
		webhookRepo:     webhookRepo,
		regulatorClient: regulatorClient,
		regulatorConfig: regulatorConfig,
		logger:          slog.Default().With("service", "WebhookService"),
	}
}

// QueueTransferNotification creates a database record to send a webhook for a transfer.
func (s *webhookService) QueueTransferNotification(ctx context.Context, transfer *models.Transfer) error {
	if transfer.Status != models.TransferStatusCompleted && transfer.Status != models.TransferStatusFailed {
		s.logger.Warn("attempted to queue webhook for transfer with non-terminal status", "transfer_id", transfer.ID, "status", transfer.Status)
		return nil // Don't queue for non-terminal states
	}

	notification := &models.WebhookNotification{
		TransferID: transfer.ID,
		URL:        s.regulatorConfig.WebhookURL,
		Status:     models.WebhookStatusPending,
	}

	if err := s.webhookRepo.Create(notification); err != nil {
		s.logger.Error("failed to create webhook notification record", "error", err, "transfer_id", transfer.ID)
		return fmt.Errorf("failed to queue webhook notification: %w", err)
	}

	s.logger.Info("queued webhook notification for transfer", "transfer_id", transfer.ID, "notification_id", notification.ID)
	return nil
}

// ProcessPendingWebhooks fetches and sends pending webhooks.
func (s *webhookService) ProcessPendingWebhooks(ctx context.Context) {
	s.logger.Info("starting check for pending webhook notifications")

	notifications, err := s.webhookRepo.FindPending(webhookBatchLimit)
	if err != nil {
		s.logger.Error("failed to fetch pending webhooks", "error", err)
		return
	}

	if len(notifications) == 0 {
		s.logger.Info("no pending webhooks to process")
		return
	}

	s.logger.Info("found pending webhooks to process", "count", len(notifications))

	for _, notification := range notifications {
		payload := &dto.RegulatorNotificationPayload{
			TransferID:  notification.Transfer.ID,
			Status:      notification.Transfer.Status,
			Amount:      notification.Transfer.Amount.String(),
			Currency:    "USD", // Assuming USD
			CompletedAt: notification.Transfer.CompletedAt,
			FailedAt:    notification.Transfer.FailedAt,
			Reason:      notification.Transfer.ErrorMessage,
		}

		err := s.regulatorClient.SendTransferNotification(ctx, payload)

		now := time.Now()
		notification.LastAttemptAt = &now
		notification.Attempts++

		if err != nil {
			s.logger.Warn("failed to send webhook notification", "error", err, "notification_id", notification.ID, "attempt", notification.Attempts)
			notification.Status = models.WebhookStatusFailed
			// Exponential backoff: 1m, 2m, 4m, 8m, 16m
			backoffDuration := initialBackoffPeriod * time.Duration(math.Pow(2, float64(notification.Attempts-1)))
			nextAttempt := now.Add(backoffDuration)
			notification.NextAttemptAt = &nextAttempt
		} else {
			s.logger.Info("successfully sent webhook notification", "notification_id", notification.ID)
			notification.Status = models.WebhookStatusSent
		}

		if err := s.webhookRepo.Update(&notification); err != nil {
			s.logger.Error("failed to update webhook notification status", "error", err, "notification_id", notification.ID)
		}
	}
}
