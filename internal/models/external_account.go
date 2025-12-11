package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ExternalAccount represents a registered payee account at an external bank (like Northwind).
// This allows users to save beneficiary details for future transfers.
type ExternalAccount struct {
	ID                uuid.UUID `gorm:"type:uuid;primary_key;"`
	UserID            uuid.UUID `gorm:"type:uuid;not null;index"`
	User              User      `gorm:"foreignKey:UserID"`
	ExternalAccountID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"` // The ID from the external system.
	Nickname          string    `gorm:"type:varchar(100);not null"`
	AccountNumberMask string    `gorm:"type:varchar(4);not null"` // Store only the last 4 digits for display.
	NameOnAccount     string    `gorm:"type:varchar(255);not null"`
	BankName          string    `gorm:"type:varchar(100);not null"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         gorm.DeletedAt `gorm:"index"`
}

// BeforeCreate will set a UUID rather than an integer ID.
func (a *ExternalAccount) BeforeCreate(tx *gorm.DB) (err error) {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return
}
