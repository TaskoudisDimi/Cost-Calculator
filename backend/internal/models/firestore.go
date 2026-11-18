package models

import "time"

// Provider is stored in the global "providers" collection (seeded, read-only for users)
type Provider struct {
	ID          string `json:"id" firestore:"-"`
	Name        string `json:"name" firestore:"name"`
	Category    string `json:"category" firestore:"category"`
	PaymentURL  string `json:"payment_url" firestore:"payment_url"`
	Color       string `json:"color" firestore:"color"`
	Description string `json:"description" firestore:"description"`
}

// UserProvider is stored in the "userProviders" collection
type UserProvider struct {
	ID         string   `json:"id" firestore:"-"`
	UserID     string   `json:"user_id" firestore:"user_id"`
	ProviderID string   `json:"provider_id" firestore:"provider_id"`
	Provider   Provider `json:"provider" firestore:"provider"`
	Nickname   string   `json:"nickname" firestore:"nickname"`
	AccountNum string   `json:"account_num" firestore:"account_num"`
}

// Member is stored in the "members" collection — a named person within one account
type Member struct {
	ID        string    `json:"id" firestore:"-"`
	UserID    string    `json:"user_id" firestore:"user_id"`
	Name      string    `json:"name" firestore:"name"`
	Color     string    `json:"color" firestore:"color"` // hex color for avatar
	CreatedAt time.Time `json:"created_at" firestore:"created_at"`
	UpdatedAt time.Time `json:"updated_at" firestore:"updated_at"`
}

// Bill is stored in the "bills" collection
type Bill struct {
	ID             string       `json:"id" firestore:"-"`
	UserID         string       `json:"user_id" firestore:"user_id"`
	MemberID       string       `json:"member_id" firestore:"member_id"`
	UserProviderID string       `json:"user_provider_id" firestore:"user_provider_id"`
	UserProvider   UserProvider `json:"user_provider" firestore:"user_provider"`
	Amount         float64      `json:"amount" firestore:"amount"`
	DueDate        time.Time    `json:"due_date" firestore:"due_date"`
	IssuedDate     *time.Time   `json:"issued_date" firestore:"issued_date"`
	Status         string       `json:"status" firestore:"status"`
	PaidAt         *time.Time   `json:"paid_at" firestore:"paid_at"`
	Notes          string       `json:"notes" firestore:"notes"`
	PaymentCode    string       `json:"payment_code" firestore:"payment_code"`
	PaymentURL     string       `json:"payment_url" firestore:"payment_url"`
	PaymentCardID  string       `json:"payment_card_id" firestore:"payment_card_id"`
	AutoPay        bool         `json:"auto_pay" firestore:"auto_pay"`
	Recurring      bool         `json:"recurring" firestore:"recurring"`
	AttachmentURL  string       `json:"attachment_url" firestore:"attachment_url"`
	CreatedAt      time.Time    `json:"created_at" firestore:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at" firestore:"updated_at"`
}

// Expense is stored in the "expenses" collection
type Expense struct {
	ID          string     `json:"id" firestore:"-"`
	UserID      string     `json:"user_id" firestore:"user_id"`
	Month       string     `json:"month" firestore:"month"` // YYYY-MM
	Name        string     `json:"name" firestore:"name"`
	Amount      float64    `json:"amount" firestore:"amount"`
	Category    string     `json:"category" firestore:"category"`
	Status      string     `json:"status" firestore:"status"`
	Store       string     `json:"store" firestore:"store"`
	Notes       string     `json:"notes" firestore:"notes"`
	PlannedDate *time.Time `json:"planned_date" firestore:"planned_date"`
	BoughtAt    *time.Time `json:"bought_at" firestore:"bought_at"`
	CreatedAt   time.Time  `json:"created_at" firestore:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" firestore:"updated_at"`
}

// UserProviderTemplate is stored in "userProviderTemplates" — user-created reusable provider shortcuts
type UserProviderTemplate struct {
	ID         string `json:"id" firestore:"-"`
	UserID     string `json:"user_id" firestore:"user_id"`
	Name       string `json:"name" firestore:"name"`
	Category   string `json:"category" firestore:"category"`
	Color      string `json:"color" firestore:"color"`
	PaymentURL string `json:"payment_url" firestore:"payment_url"`
}

// Card stores payment card metadata (no actual card numbers — PCI DSS)
type Card struct {
	ID        string    `json:"id" firestore:"-"`
	UserID    string    `json:"user_id" firestore:"user_id"`
	Nickname  string    `json:"nickname" firestore:"nickname"`
	LastFour  string    `json:"last_four" firestore:"last_four"`
	ExpiryMM  int       `json:"expiry_mm" firestore:"expiry_mm"`
	ExpiryYY  int       `json:"expiry_yy" firestore:"expiry_yy"`
	CardType  string    `json:"card_type" firestore:"card_type"` // visa, mastercard, amex, other
	IsDefault bool      `json:"is_default" firestore:"is_default"`
	Color     string    `json:"color" firestore:"color"`
	CreatedAt time.Time `json:"created_at" firestore:"created_at"`
	UpdatedAt time.Time `json:"updated_at" firestore:"updated_at"`
}

// BankConnection stores the Nordigen/GoCardless bank link for a user
type BankConnection struct {
	ID              string    `json:"id" firestore:"-"`
	UserID          string    `json:"user_id" firestore:"user_id"`
	RequisitionID   string    `json:"requisition_id" firestore:"requisition_id"`
	InstitutionID   string    `json:"institution_id" firestore:"institution_id"`
	InstitutionName string    `json:"institution_name" firestore:"institution_name"`
	Status          string    `json:"status" firestore:"status"` // CR, LN, RJ, ER, SU, EX
	AccountIDs      []string  `json:"account_ids" firestore:"account_ids"`
	LastSyncedAt    *time.Time `json:"last_synced_at" firestore:"last_synced_at"`
	CreatedAt       time.Time `json:"created_at" firestore:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" firestore:"updated_at"`
}

// Income is stored in the "income" collection
type Income struct {
	ID          string    `json:"id" firestore:"-"`
	UserID      string    `json:"user_id" firestore:"user_id"`
	MemberID    string    `json:"member_id" firestore:"member_id"`
	Description string    `json:"description" firestore:"description"`
	Amount      float64   `json:"amount" firestore:"amount"`
	Month       string    `json:"month" firestore:"month"` // YYYY-MM
	CreatedAt   time.Time `json:"created_at" firestore:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" firestore:"updated_at"`
}
