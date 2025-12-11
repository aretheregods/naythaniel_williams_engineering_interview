package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	WebhookStatusPending = "pending"
	WebhookStatusSent    = "sent"
	WebhookStatusFailed  = "failed"
	WebhookMaxAttempts   = 5
)

// WebhookNotification stores the state of an outgoing webhook to a regulator.
// This provides a persistent, auditable record of compliance notifications.
type WebhookNotification struct {
	ID                 uuid.UUID `gorm:"type:uuid;primary_key;"`
	TransferID         uuid.UUID `gorm:"type:uuid;not null;index"`
	Transfer           Transfer  `gorm:"foreignKey:TransferID"`
	URL                string    `gorm:"type:varchar(512);not null"`
	Status             string    `gorm:"type:varchar(20);not null;default:'pending';index"`
	Attempts           int       `gorm:"not null;default:0"`
	LastAttemptAt      *time.Time
	NextAttemptAt      *time.Time `gorm:"index"`
	ResponseBody       *string    `gorm:"type:text"`
	ResponseStatusCode *int
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// BeforeCreate will set a UUID rather than an integer ID.
func (w *WebhookNotification) BeforeCreate(tx *gorm.DB) (err error) {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	if w.NextAttemptAt == nil {
		now := time.Now()
		w.NextAttemptAt = &now
	}
	return
}
