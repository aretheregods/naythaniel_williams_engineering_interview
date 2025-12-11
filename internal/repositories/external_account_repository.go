package repositories

import (
	"errors"
	"fmt"

	"github.com/array/banking-api/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type externalAccountRepository struct {
	db *gorm.DB
}

var ErrExternalAccountNotFound = errors.New("external account not found")

func NewExternalAccountRepository(db *gorm.DB) ExternalAccountRepositoryInterface {
	return &externalAccountRepository{db: db}
}

func (r *externalAccountRepository) Create(account *models.ExternalAccount) error {
	if err := r.db.Create(account).Error; err != nil {
		return fmt.Errorf("failed to create external account: %w", err)
	}
	return nil
}

func (r *externalAccountRepository) GetByID(id uuid.UUID) (*models.ExternalAccount, error) {
	var account models.ExternalAccount
	if err := r.db.First(&account, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExternalAccountNotFound
		}
		return nil, fmt.Errorf("failed to find external account by id: %w", err)
	}
	return &account, nil
}

func (r *externalAccountRepository) ListByUserID(userID uuid.UUID) ([]models.ExternalAccount, error) {
	var accounts []models.ExternalAccount
	if err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&accounts).Error; err != nil {
		return nil, fmt.Errorf("failed to list external accounts for user: %w", err)
	}
	return accounts, nil
}
