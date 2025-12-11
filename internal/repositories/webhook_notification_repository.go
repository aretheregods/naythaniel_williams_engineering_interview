package repositories

import (
	"fmt"
	"time"

	"github.comcom/array/banking-api/internal/models"
	"gorm.io/gorm"
)

type webhookNotificationRepository struct {
	db *gorm.DB
}

func NewWebhookNotificationRepository(db *gorm.DB) WebhookNotificationRepositoryInterface {
	return &webhookNotificationRepository{db: db}
}

func (r *webhookNotificationRepository) Create(notification *models.WebhookNotification) error {
	if err := r.db.Create(notification).Error; err != nil {
		return fmt.Errorf("failed to create webhook notification: %w", err)
	}
	return nil
}

func (r *webhookNotificationRepository) Update(notification *models.WebhookNotification) error {
	if err := r.db.Save(notification).Error; err != nil {
		return fmt.Errorf("failed to update webhook notification: %w", err)
	}
	return nil
}

// FindPending retrieves notifications that are pending or failed and ready for a retry.
func (r *webhookNotificationRepository) FindPending(limit int) ([]models.WebhookNotification, error) {
	var notifications []models.WebhookNotification
	now := time.Now()

	err := r.db.Preload("Transfer").Where("status IN ? AND next_attempt_at <= ? AND attempts < ?",
		[]string{models.WebhookStatusPending, models.WebhookStatusFailed},
		now,
		models.WebhookMaxAttempts,
	).
		Limit(limit).
		Order("next_attempt_at ASC").
		Find(&notifications).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find pending webhook notifications: %w", err)
	}
	return notifications, nil
}
