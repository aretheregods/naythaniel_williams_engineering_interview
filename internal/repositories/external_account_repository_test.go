package repositories

import (
	"testing"

	"github.com/array/banking-api/internal/database"
	"github.com/array/banking-api/internal/models"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type ExternalAccountRepositoryTestSuite struct {
	suite.Suite
	db   *database.DB
	repo ExternalAccountRepositoryInterface
	user *models.User
}

func (s *ExternalAccountRepositoryTestSuite) SetupTest() {
	s.db = database.SetupTestDB(s.T())
	s.repo = NewExternalAccountRepository(s.db.DB)
	s.user = database.CreateTestUser(s.T(), s.db, gofakeit.Email())
}

func (s *ExternalAccountRepositoryTestSuite) TearDownTest() {
	database.CleanupTestDB(s.T(), s.db)
}

func TestExternalAccountRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(ExternalAccountRepositoryTestSuite))
}

func (s *ExternalAccountRepositoryTestSuite) TestCreate_Success() {
	account := &models.ExternalAccount{
		UserID:            s.user.ID,
		ExternalAccountID: uuid.New(),
		Nickname:          "My Savings",
		AccountNumberMask: "1234",
		NameOnAccount:     gofakeit.Name(),
		BankName:          "Test Bank",
	}

	err := s.repo.Create(account)
	s.NoError(err)
	s.NotEqual(uuid.Nil, account.ID)

	var found models.ExternalAccount
	err = s.db.First(&found, "id = ?", account.ID).Error
	s.NoError(err)
	s.Equal(account.Nickname, found.Nickname)
	s.Equal(account.ExternalAccountID, found.ExternalAccountID)
}

func (s *ExternalAccountRepositoryTestSuite) TestGetByID_Success() {
	account := &models.ExternalAccount{
		UserID:            s.user.ID,
		ExternalAccountID: uuid.New(),
		Nickname:          "Test Get By ID",
		AccountNumberMask: "5678",
		NameOnAccount:     gofakeit.Name(),
		BankName:          "Test Bank",
	}
	s.NoError(s.repo.Create(account))

	found, err := s.repo.GetByID(account.ID)
	s.NoError(err)
	s.NotNil(found)
	s.Equal(account.ID, found.ID)
	s.Equal(account.Nickname, found.Nickname)
}

func (s *ExternalAccountRepositoryTestSuite) TestGetByID_NotFound() {
	found, err := s.repo.GetByID(uuid.New())
	s.Error(err)
	s.Nil(found)
	s.ErrorIs(err, ErrExternalAccountNotFound)
}

func (s *ExternalAccountRepositoryTestSuite) TestListByUserID() {
	// Create 2 accounts for the main user
	acc1 := &models.ExternalAccount{
		UserID: s.user.ID, ExternalAccountID: uuid.New(), Nickname: "User1 Acc1", AccountNumberMask: "1111", NameOnAccount: gofakeit.Name(), BankName: "Test Bank 1",
	}
	s.NoError(s.repo.Create(acc1))

	acc2 := &models.ExternalAccount{
		UserID: s.user.ID, ExternalAccountID: uuid.New(), Nickname: "User1 Acc2", AccountNumberMask: "2222", NameOnAccount: gofakeit.Name(), BankName: "Test Bank 2",
	}
	s.NoError(s.repo.Create(acc2))

	// Create 1 account for another user
	otherUser := database.CreateTestUser(s.T(), s.db, gofakeit.Email())
	acc3 := &models.ExternalAccount{
		UserID: otherUser.ID, ExternalAccountID: uuid.New(), Nickname: "User2 Acc1", AccountNumberMask: "3333", NameOnAccount: gofakeit.Name(), BankName: "Test Bank 3",
	}
	s.NoError(s.repo.Create(acc3))

	// List accounts for the main user
	accounts, err := s.repo.ListByUserID(s.user.ID)
	s.NoError(err)
	s.Len(accounts, 2)

	// Verify the correct accounts are returned
	foundNicknames := []string{accounts[0].Nickname, accounts[1].Nickname}
	s.Contains(foundNicknames, "User1 Acc1")
	s.Contains(foundNicknames, "User1 Acc2")
	s.NotContains(foundNicknames, "User2 Acc1")
}

func (s *ExternalAccountRepositoryTestSuite) TestListByUserID_NoAccounts() {
	accounts, err := s.repo.ListByUserID(s.user.ID)
	s.NoError(err)
	s.Empty(accounts)
	s.Len(accounts, 0)
}
