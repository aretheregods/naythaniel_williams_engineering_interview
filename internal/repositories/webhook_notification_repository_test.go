package repositories

import (
	"testing"
	"time"

	"github.com/array/banking-api/internal/database"
	"github.com/array/banking-api/internal/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

type WebhookNotificationRepositoryTestSuite struct {
	suite.Suite
	db   *database.DB
	repo WebhookNotificationRepositoryInterface
}

func (s *WebhookNotificationRepositoryTestSuite) SetupTest() {
	s.db = database.SetupTestDB(s.T())
	s.NoError(s.db.AutoMigrate(&models.Transfer{}, &models.WebhookNotification{}))
	s.repo = NewWebhookNotificationRepository(s.db.DB)
}

func (s *WebhookNotificationRepositoryTestSuite) TearDownTest() {
	database.CleanupTestDB(s.T(), s.db)
}

func TestWebhookNotificationRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(WebhookNotificationRepositoryTestSuite))
}

func (s *WebhookNotificationRepositoryTestSuite) createTestTransfer() *models.Transfer {
	toAccountID := uuid.New()
	transfer := &models.Transfer{
		FromAccountID:  uuid.New(),
		ToAccountID:    &toAccountID,
		Amount:         decimal.NewFromInt(100),
		Description:    "test",
		IdempotencyKey: uuid.New().String(),
	}
	s.NoError(s.db.Create(transfer).Error)
	return transfer
}

func (s *WebhookNotificationRepositoryTestSuite) TestCreate_Success() {
	transfer := s.createTestTransfer()
	notification := &models.WebhookNotification{
		TransferID: transfer.ID,
		URL:        "https://example.com/webhook",
		Status:     models.WebhookStatusPending,
	}

	err := s.repo.Create(notification)
	s.NoError(err)
	s.NotEqual(uuid.Nil, notification.ID)

	var found models.WebhookNotification
	err = s.db.First(&found, "id = ?", notification.ID).Error
	s.NoError(err)
	s.Equal(notification.URL, found.URL)
	s.Equal(notification.TransferID, found.TransferID)
}

func (s *WebhookNotificationRepositoryTestSuite) TestUpdate_Success() {
	transfer := s.createTestTransfer()
	notification := &models.WebhookNotification{
		TransferID: transfer.ID,
		URL:        "https://example.com/webhook",
		Status:     models.WebhookStatusPending,
	}
	s.NoError(s.repo.Create(notification))

	now := time.Now()
	notification.Status = models.WebhookStatusSent
	notification.Attempts = 1
	notification.LastAttemptAt = &now

	err := s.repo.Update(notification)
	s.NoError(err)

	var found models.WebhookNotification
	s.NoError(s.db.First(&found, "id = ?", notification.ID).Error)
	s.Equal(models.WebhookStatusSent, found.Status)
	s.Equal(1, found.Attempts)
	s.NotNil(found.LastAttemptAt)
}

func (s *WebhookNotificationRepositoryTestSuite) TestFindPending() {
	// 1. Pending and due now (should be found)
	pendingDue := &models.WebhookNotification{TransferID: s.createTestTransfer().ID, URL: "url1", Status: models.WebhookStatusPending}
	s.NoError(s.repo.Create(pendingDue))

	// 2. Pending but not due yet (should NOT be found)
	nextAttemptFuture := time.Now().Add(1 * time.Hour)
	pendingNotDue := &models.WebhookNotification{TransferID: s.createTestTransfer().ID, URL: "url2", Status: models.WebhookStatusPending, NextAttemptAt: &nextAttemptFuture}
	s.NoError(s.repo.Create(pendingNotDue))

	// 3. Failed and ready for retry (should be found)
	nextAttemptPast := time.Now().Add(-1 * time.Minute)
	failedRetry := &models.WebhookNotification{TransferID: s.createTestTransfer().ID, URL: "url3", Status: models.WebhookStatusFailed, Attempts: 2, NextAttemptAt: &nextAttemptPast}
	s.NoError(s.repo.Create(failedRetry))

	// 4. Sent (should NOT be found)
	sent := &models.WebhookNotification{TransferID: s.createTestTransfer().ID, URL: "url4", Status: models.WebhookStatusSent}
	s.NoError(s.repo.Create(sent))

	// 5. Failed and max attempts reached (should NOT be found)
	failedMaxed := &models.WebhookNotification{TransferID: s.createTestTransfer().ID, URL: "url5", Status: models.WebhookStatusFailed, Attempts: models.WebhookMaxAttempts, NextAttemptAt: &nextAttemptPast}
	s.NoError(s.repo.Create(failedMaxed))

	notifications, err := s.repo.FindPending(10)
	s.NoError(err)
	s.Len(notifications, 2)

	foundIDs := make(map[uuid.UUID]bool)
	for _, n := range notifications {
		foundIDs[n.ID] = true
	}

	s.True(foundIDs[pendingDue.ID])
	s.True(foundIDs[failedRetry.ID])
	s.False(foundIDs[pendingNotDue.ID])
	s.False(foundIDs[sent.ID])
	s.False(foundIDs[failedMaxed.ID])
}
