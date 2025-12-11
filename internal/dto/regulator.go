package dto

import (
	"time"

	"github.com/google/uuid"
)

// RegulatorNotificationPayload is the data sent to the regulator's webhook.
type RegulatorNotificationPayload struct {
	TransferID  uuid.UUID  `json:"transfer_id"`
	Status      string     `json:"status"` // "completed" or "failed"
	Amount      string     `json:"amount"`
	Currency    string     `json:"currency"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	FailedAt    *time.Time `json:"failed_at,omitempty"`
	Reason      *string    `json:"reason,omitempty"` // Reason for failure
}
