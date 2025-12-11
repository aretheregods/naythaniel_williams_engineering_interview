package services

import (
	"context"
	"errors"
	"testing"

	"github.com/array/banking-api/internal/dto"
	"github.com/array/banking-api/internal/models"
	"github.com/array/banking-api/internal/repositories/repository_mocks"
	"github.com/array/banking-api/internal/services/service_mocks"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type ExternalAccountServiceTestSuite struct {
	suite.Suite
	ctrl                *gomock.Controller
	externalAccountRepo *repository_mocks.MockExternalAccountRepositoryInterface
	northwindClient     *service_mocks.MockNorthwindClientInterface
	service             ExternalAccountServiceInterface
}

func (s *ExternalAccountServiceTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.externalAccountRepo = repository_mocks.NewMockExternalAccountRepositoryInterface(s.ctrl)
	s.northwindClient = service_mocks.NewMockNorthwindClientInterface(s.ctrl)
	s.service = NewExternalAccountService(s.externalAccountRepo, s.northwindClient)
}

func (s *ExternalAccountServiceTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func TestExternalAccountServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ExternalAccountServiceTestSuite))
}

func (s *ExternalAccountServiceTestSuite) TestRegister_Success() {
	userID := uuid.New()
	northwindID := uuid.New()
	req := &dto.RegisterExternalAccountRequest{
		BankName:      "Northwind Bank",
		Nickname:      "Vacation Fund",
		AccountNumber: "123456789012",
		RoutingNumber: "123456789",
		NameOnAccount: "John Doe",
	}

	s.northwindClient.EXPECT().
		CreateExternalAccount(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, details *dto.NorthwindCreateAccountRequest) (*dto.NorthwindExternalAccountResponse, error) {
			s.Equal(req.AccountNumber, details.AccountNumber)
			s.Equal(req.RoutingNumber, details.RoutingNumber)
			s.Equal(req.NameOnAccount, details.NameOnAccount)
			return &dto.NorthwindExternalAccountResponse{ID: northwindID}, nil
		}).Times(1)

	s.externalAccountRepo.EXPECT().
		Create(gomock.Any()).
		DoAndReturn(func(account *models.ExternalAccount) error {
			s.Equal(userID, account.UserID)
			s.Equal(northwindID, account.ExternalAccountID)
			s.Equal(req.Nickname, account.Nickname)
			s.Equal("9012", account.AccountNumberMask) // last 4 digits
			s.Equal(req.NameOnAccount, account.NameOnAccount)
			s.Equal(req.BankName, account.BankName)
			return nil
		}).Times(1)

	account, err := s.service.Register(context.Background(), userID, req)
	s.NoError(err)
	s.NotNil(account)
	s.Equal(northwindID, account.ExternalAccountID)
}

func (s *ExternalAccountServiceTestSuite) TestRegister_NorthwindAPIFailure() {
	userID := uuid.New()
	req := &dto.RegisterExternalAccountRequest{
		BankName:      "Northwind Bank",
		Nickname:      "Test",
		AccountNumber: "123456789012",
		RoutingNumber: "123456789",
		NameOnAccount: "Jane Doe",
	}

	s.northwindClient.EXPECT().
		CreateExternalAccount(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("northwind is down")).
		Times(1)

	// Ensure the local repo is not called
	s.externalAccountRepo.EXPECT().Create(gomock.Any()).Times(0)

	account, err := s.service.Register(context.Background(), userID, req)
	s.Error(err)
	s.Nil(account)
	s.ErrorIs(err, ErrRegistrationFailed)
}