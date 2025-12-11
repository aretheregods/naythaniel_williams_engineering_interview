package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/array/banking-api/internal/dto"
	"github.com/array/banking-api/internal/models"
	"github.com/array/banking-api/internal/repositories"
	"github.com/google/uuid"
)

var (
	ErrRegistrationFailed = errors.New("failed to register external account with Northwind Bank")
)

type externalAccountService struct {
	externalAccountRepo repositories.ExternalAccountRepositoryInterface
	northwindClient     NorthwindClientInterface
}

func NewExternalAccountService(
	externalAccountRepo repositories.ExternalAccountRepositoryInterface,
	northwindClient NorthwindClientInterface,
) ExternalAccountServiceInterface {
	return &externalAccountService{
		externalAccountRepo: externalAccountRepo,
		northwindClient:     northwindClient,
	}
}

func (s *externalAccountService) Register(ctx context.Context, userID uuid.UUID, req *dto.RegisterExternalAccountRequest) (*models.ExternalAccount, error) {
	northwindReq := &dto.NorthwindCreateAccountRequest{
		AccountNumber: req.AccountNumber,
		RoutingNumber: req.RoutingNumber,
		NameOnAccount: req.NameOnAccount,
		AccountType:   "checking", // Defaulting as per Northwind API spec for external accounts
		Currency:      "USD",
	}

	northwindResp, err := s.northwindClient.CreateExternalAccount(ctx, northwindReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRegistrationFailed, err)
	}

	account := &models.ExternalAccount{
		UserID:            userID,
		ExternalAccountID: northwindResp.ID,
		Nickname:          req.Nickname,
		AccountNumberMask: req.AccountNumber[len(req.AccountNumber)-4:],
		NameOnAccount:     req.NameOnAccount,
		BankName:          req.BankName,
	}

	if err := s.externalAccountRepo.Create(account); err != nil {
		return nil, fmt.Errorf("failed to save external account locally: %w", err)
	}

	return account, nil
}
