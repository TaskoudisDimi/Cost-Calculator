package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"

	"cloud.google.com/go/firestore"
	"github.com/dimitris-taskou/cost-calculator/internal/agent"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AgentHandler struct {
	fs         *firestore.Client
	anthropicKey string
	encKey     []byte // 32-byte AES key for card decryption
}

func NewAgentHandler(fs *firestore.Client, anthropicKey string) *AgentHandler {
	var encKey []byte
	if k := os.Getenv("CARD_ENCRYPTION_KEY"); len(k) == 32 {
		encKey = []byte(k)
	}
	return &AgentHandler{
		fs:           fs,
		anthropicKey: anthropicKey,
		encKey:       encKey,
	}
}

type agentStartRequest struct {
	CredentialID string  `json:"credential_id"`
	CardID       string  `json:"card_id"`
	Amount       float64 `json:"amount"`
}

// active jobs: job_id → OTP channel
var (
	otpMu   sync.Mutex
	otpJobs = map[string]chan string{}
)

// Pay starts an AI payment job and streams SSE events back to the frontend.
func (h *AgentHandler) Pay(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	billID := c.Param("billId")

	var req agentStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Fetch bill
	billDoc, err := h.fs.Collection("bills").Doc(billID).Get(ctx)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bill not found"})
		return
	}
	var bill struct {
		UserID       string  `firestore:"user_id"`
		Amount       float64 `firestore:"amount"`
		PaymentCode  string  `firestore:"payment_code"`
		PaymentURL   string  `firestore:"payment_url"`
		UserProvider struct {
			Nickname string `firestore:"nickname"`
			Provider struct {
				Name       string `firestore:"name"`
				PaymentURL string `firestore:"payment_url"`
			} `firestore:"provider"`
		} `firestore:"user_provider"`
	}
	if err := billDoc.DataTo(&bill); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if bill.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		return
	}

	providerName := bill.UserProvider.Nickname
	if providerName == "" {
		providerName = bill.UserProvider.Provider.Name
	}
	paymentURL := bill.PaymentURL
	if paymentURL == "" {
		paymentURL = bill.UserProvider.Provider.PaymentURL
	}

	// Fetch credential from Firestore
	var credLabel, credUser, credPass string
	if req.CredentialID != "" {
		if credDoc, err := h.fs.Collection("userCredentials").Doc(req.CredentialID).Get(ctx); err == nil {
			var cred struct {
				UserID   string `firestore:"user_id"`
				Label    string `firestore:"label"`
				Username string `firestore:"username"`
				Password string `firestore:"password"`
			}
			if credDoc.DataTo(&cred) == nil && cred.UserID == userID {
				credLabel = cred.Label
				credUser = cred.Username
				credPass = cred.Password
			}
		}
	}

	// Fetch card from Firestore and decrypt
	var cardNum, cardExpiry, cardCVV, cardHolder string
	if req.CardID != "" && len(h.encKey) == 32 {
		if cardDoc, err := h.fs.Collection("cards").Doc(req.CardID).Get(ctx); err == nil {
			data := cardDoc.Data()
			if uid, _ := data["user_id"].(string); uid == userID {
				cardNum = h.decryptField(data, "number_enc")
				cardExpiry = h.decryptField(data, "expiry_enc")
				cardCVV = h.decryptField(data, "cvv_enc")
				if cn, ok := data["cardholder_name"].(string); ok {
					cardHolder = cn
				}
			}
		}
	}

	jobID := uuid.New().String()
	otpCh := make(chan string, 1)
	otpMu.Lock()
	otpJobs[jobID] = otpCh
	otpMu.Unlock()

	cfg := agent.JobConfig{
		JobID:        jobID,
		APIKey:       h.anthropicKey,
		ProviderName: providerName,
		Amount: func() float64 {
			if req.Amount > 0 {
				return req.Amount
			}
			return bill.Amount
		}(),
		PaymentCode:  bill.PaymentCode,
		PaymentURL:   paymentURL,
		CredLabel:    credLabel,
		CredUser:     credUser,
		CredPass:     credPass,
		CardNumber:   cardNum,
		CardExpiry:   cardExpiry,
		CardCVV:      cardCVV,
		CardHolder:   cardHolder,
		OTPCh:        otpCh,
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
	c.Header("Connection", "keep-alive")

	out := make(chan agent.Event, 32)

	go func() {
		defer close(out)
		defer func() {
			otpMu.Lock()
			delete(otpJobs, jobID)
			otpMu.Unlock()
		}()
		agent.RunLoop(c.Request.Context(), cfg, out)
	}()

	for ev := range out {
		b, _ := json.Marshal(ev)
		fmt.Fprintf(c.Writer, "data: %s\n\n", b)
		c.Writer.Flush()
	}
	fmt.Fprintf(c.Writer, "data: {\"type\":\"done\"}\n\n")
	c.Writer.Flush()
}

// SubmitOTP forwards an OTP to the waiting agent job.
func (h *AgentHandler) SubmitOTP(c *gin.Context) {
	jobID := c.Param("jobId")
	var body struct {
		OTP string `json:"otp"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	otpMu.Lock()
	ch, ok := otpJobs[jobID]
	otpMu.Unlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}
	select {
	case ch <- body.OTP:
		c.JSON(http.StatusOK, gin.H{"ok": true})
	default:
		c.JSON(http.StatusConflict, gin.H{"error": "OTP already submitted"})
	}
}

// GetCredentials returns the user's credential list (labels only) for the agent UI.
func (h *AgentHandler) GetCredentials(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	ctx := context.Background()

	docs, err := h.fs.Collection("userCredentials").
		Where("user_id", "==", userID).
		OrderBy("created_at", firestore.Desc).
		Documents(ctx).GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type credSummary struct {
		ID    string `json:"id"`
		Label string `json:"label"`
		URL   string `json:"url"`
	}
	result := make([]credSummary, 0, len(docs))
	for _, d := range docs {
		data := d.Data()
		label, _ := data["label"].(string)
		url, _ := data["url"].(string)
		result = append(result, credSummary{ID: d.Ref.ID, Label: label, URL: url})
	}
	c.JSON(http.StatusOK, result)
}

func (h *AgentHandler) decryptField(data map[string]any, field string) string {
	enc, _ := data[field].(string)
	if enc == "" || len(h.encKey) != 32 {
		return ""
	}
	plain, err := agent.Decrypt(h.encKey, enc)
	if err != nil {
		return ""
	}
	return plain
}
