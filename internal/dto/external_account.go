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

// NorthwindInitiateTransferRequest is the DTO for the request to Northwind's /transfers endpoint.
type NorthwindInitiateTransferRequest struct {
	SourceAccountID      string `json:"source_account_id"`      // Our internal account number
	DestinationAccountID string `json:"destination_account_id"` // The Northwind account UUID
	Amount               string `json:"amount"`
	Direction            string `json:"direction"`     // "debit" for sending money out
	TransferType         string `json:"transfer_type"` // "standard" or "express"
}

// NorthwindInitiateTransferResponse is the DTO for the response from Northwind's /transfers endpoint.
type NorthwindInitiateTransferResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// NorthwindGetTransferResponse is the DTO for the response from Northwind's GET /transfers/{id} endpoint.
type NorthwindGetTransferResponse struct {
	ID                   string    `json:"id"`
	SourceAccountID      string    `json:"source_account_id"`
	DestinationAccountID string    `json:"destination_account_id"`
	Amount               string    `json:"amount"`
	Direction            string    `json:"direction"`
	Status               string    `json:"status"`
	CreatedAt            time.Time `json:"created_at"`
}
