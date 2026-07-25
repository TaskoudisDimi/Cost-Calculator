package nordigen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const baseURL = "https://bankaccountdata.gocardless.com/api/v2"

// Service is a thread-safe Nordigen/GoCardless Open Banking API client.
type Service struct {
	secretID  string
	secretKey string

	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

func New(secretID, secretKey string) *Service {
	return &Service{secretID: secretID, secretKey: secretKey}
}

func (s *Service) Configured() bool {
	return s.secretID != "" && s.secretKey != ""
}

// ── Token management ──────────────────────────────────────────────────────────

type tokenResp struct {
	Access        string `json:"access"`
	AccessExpires int    `json:"access_expires"`
}

func (s *Service) token(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.accessToken != "" && time.Now().Before(s.tokenExpiry) {
		return s.accessToken, nil
	}

	body, _ := json.Marshal(map[string]string{
		"secret_id":  s.secretID,
		"secret_key": s.secretKey,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/token/new/", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("nordigen token error %d: %s", resp.StatusCode, string(b))
	}

	var tr tokenResp
	if err := json.Unmarshal(b, &tr); err != nil || tr.Access == "" {
		return "", fmt.Errorf("nordigen: bad token response: %s", string(b))
	}

	s.accessToken = tr.Access
	expSecs := tr.AccessExpires
	if expSecs <= 0 {
		expSecs = 86400 // default 24h
	}
	s.tokenExpiry = time.Now().Add(time.Duration(expSecs-120) * time.Second)
	return s.accessToken, nil
}

func (s *Service) do(ctx context.Context, method, path string, body any) ([]byte, int, error) {
	tok, err := s.token(ctx)
	if err != nil {
		return nil, 0, err
	}

	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, bodyReader)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)
	return b, resp.StatusCode, nil
}

// ── Institutions ──────────────────────────────────────────────────────────────

type Institution struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	BIC  string `json:"bic"`
	Logo string `json:"logo"`
}

func (s *Service) ListInstitutions(ctx context.Context, country string) ([]Institution, error) {
	b, status, err := s.do(ctx, http.MethodGet, "/institutions/?country="+country, nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("nordigen institutions error %d: %s", status, string(b))
	}
	var list []Institution
	if err := json.Unmarshal(b, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// ── End-user Agreement ────────────────────────────────────────────────────────

type EUA struct {
	ID string `json:"id"`
}

func (s *Service) CreateEUA(ctx context.Context, institutionID string) (*EUA, error) {
	payload := map[string]any{
		"institution_id":        institutionID,
		"max_historical_days":   90,
		"access_valid_for_days": 90,
		"access_scope":          []string{"details", "balances", "transactions"},
	}
	b, status, err := s.do(ctx, http.MethodPost, "/agreements/enduser/", payload)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("nordigen EUA error %d: %s", status, string(b))
	}
	var eua EUA
	return &eua, json.Unmarshal(b, &eua)
}

// ── Requisitions ──────────────────────────────────────────────────────────────

type Requisition struct {
	ID       string   `json:"id"`
	Status   string   `json:"status"` // CR, LN, RJ, ER, SU, EX, GA, SA, UA
	Link     string   `json:"link"`
	Accounts []string `json:"accounts"`
}

func (s *Service) CreateRequisition(ctx context.Context, institutionID, euaID, redirectURL, reference string) (*Requisition, error) {
	payload := map[string]any{
		"redirect":       redirectURL,
		"institution_id": institutionID,
		"agreement":      euaID,
		"reference":      reference,
		"user_language":  "EL",
	}
	b, status, err := s.do(ctx, http.MethodPost, "/requisitions/", payload)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("nordigen requisition error %d: %s", status, string(b))
	}
	var req Requisition
	return &req, json.Unmarshal(b, &req)
}

func (s *Service) GetRequisition(ctx context.Context, id string) (*Requisition, error) {
	b, status, err := s.do(ctx, http.MethodGet, "/requisitions/"+id+"/", nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("nordigen requisition get error %d: %s", status, string(b))
	}
	var req Requisition
	return &req, json.Unmarshal(b, &req)
}

func (s *Service) DeleteRequisition(ctx context.Context, id string) error {
	_, status, err := s.do(ctx, http.MethodDelete, "/requisitions/"+id+"/", nil)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("nordigen requisition delete error %d", status)
	}
	return nil
}

// ── Transactions ──────────────────────────────────────────────────────────────

type TxAmount struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

type Transaction struct {
	TransactionID string   `json:"transactionId"`
	BookingDate   string   `json:"bookingDate"`
	ValueDate     string   `json:"valueDate"`
	Amount        TxAmount `json:"transactionAmount"`
	CreditorName  string   `json:"creditorName"`
	DebtorName    string   `json:"debtorName"`
	RemittanceInfo string  `json:"remittanceInformationUnstructured"`
}

type transactionsResp struct {
	Transactions struct {
		Booked  []Transaction `json:"booked"`
		Pending []Transaction `json:"pending"`
	} `json:"transactions"`
}

func (s *Service) GetTransactions(ctx context.Context, accountID, dateFrom string) ([]Transaction, error) {
	path := "/accounts/" + accountID + "/transactions/"
	if dateFrom != "" {
		path += "?date_from=" + dateFrom
	}
	b, status, err := s.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("nordigen transactions error %d: %s", status, string(b))
	}
	var tr transactionsResp
	if err := json.Unmarshal(b, &tr); err != nil {
		return nil, err
	}
	return tr.Transactions.Booked, nil
}
