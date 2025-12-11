package dto

import (
	"time"

	"github.com/google/uuid"
)

// RegisterExternalAccountRequest defines the request body for registering a new external account.
type RegisterExternalAccountRequest struct {
	BankName      string `json:"bank_name" validate:"required,min=2,max=100"`
	Nickname      string `json:"nickname" validate:"required,min=2,max=50"`
	AccountNumber string `json:"account_number" validate:"required,min=8,max=17"`
	RoutingNumber string `json:"routing_number" validate:"required,len=9,numeric"`
	NameOnAccount string `json:"name_on_account" validate:"required,min=2,max=100"`
}

// ExternalAccountResponse defines the structure for a successfully registered external account.
type ExternalAccountResponse struct {
	ID                uuid.UUID `json:"id"`
	Nickname          string    `json:"nickname"`
	AccountNumberMask string    `json:"account_number_mask"`
	NameOnAccount     string    `json:"name_on_account"`
	BankName          string    `json:"bank_name"`
}

// NorthwindCreateAccountRequest is the DTO for the request to Northwind's API.
type NorthwindCreateAccountRequest struct {
	AccountNumber string `json:"account_number"`
	RoutingNumber string `json:"routing_number"`
	AccountType   string `json:"account_type"`
	Currency      string `json:"currency"`
	NameOnAccount string `json:"name_on_account"`
}

// NorthwindExternalAccountResponse is the DTO for the response from Northwind's API.
type NorthwindExternalAccountResponse struct {
	ID uuid.UUID `json:"id"`
}