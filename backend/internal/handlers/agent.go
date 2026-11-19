package handlers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AgentHandler struct {
	fs         *firestore.Client
	agentURL   string
	agentSecret string
}

func NewAgentHandler(fs *firestore.Client) *AgentHandler {
	return &AgentHandler{
		fs:          fs,
		agentURL:    os.Getenv("AGENT_SERVICE_URL"),
		agentSecret: os.Getenv("AGENT_SECRET"),
	}
}

type agentStartRequest struct {
	BillID              string  `json:"bill_id"`
	CredentialID        string  `json:"credential_id"`
	// Card details provided by the user at payment time (never stored full)
	CardNumber          string  `json:"card_number"`
	CardExpiry          string  `json:"card_expiry"`
	CardCVV             string  `json:"card_cvv"`
	CardHolder          string  `json:"card_holder"`
}

type agentJobPayload struct {
	JobID               string  `json:"job_id"`
	BillID              string  `json:"bill_id"`
	ProviderName        string  `json:"provider_name"`
	Amount              float64 `json:"amount"`
	PaymentCode         string  `json:"payment_code,omitempty"`
	PaymentURL          string  `json:"payment_url,omitempty"`
	CredentialLabel     string  `json:"credential_label,omitempty"`
	CredentialUsername  string  `json:"credential_username,omitempty"`
	CredentialPassword  string  `json:"credential_password,omitempty"`
	CardNumber          string  `json:"card_number,omitempty"`
	CardExpiry          string  `json:"card_expiry,omitempty"`
	CardCVV             string  `json:"card_cvv,omitempty"`
	CardHolder          string  `json:"card_holder,omitempty"`
}

// Pay starts an AI payment job and streams SSE events back to the frontend.
func (h *AgentHandler) Pay(c *gin.Context) {
	if h.agentURL == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Agent service not configured"})
		return
	}

	userID := c.MustGet("user_id").(string)
	billID := c.Param("billId")

	var req agentStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()

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

	// Fetch credentials if requested
	var credLabel, credUsername, credPassword string
	if req.CredentialID != "" {
		credDoc, err := h.fs.Collection("userCredentials").Doc(req.CredentialID).Get(ctx)
		if err == nil {
			var cred struct {
				UserID   string `firestore:"user_id"`
				Label    string `firestore:"label"`
				Username string `firestore:"username"`
				Password string `firestore:"password"`
			}
			if credDoc.DataTo(&cred) == nil && cred.UserID == userID {
				credLabel = cred.Label
				credUsername = cred.Username
				credPassword = cred.Password
			}
		}
	}

	jobID := uuid.New().String()
	payload := agentJobPayload{
		JobID:              jobID,
		BillID:             billID,
		ProviderName:       providerName,
		Amount:             bill.Amount,
		PaymentCode:        bill.PaymentCode,
		PaymentURL:         paymentURL,
		CredentialLabel:    credLabel,
		CredentialUsername: credUsername,
		CredentialPassword: credPassword,
		CardNumber:         req.CardNumber,
		CardExpiry:         req.CardExpiry,
		CardCVV:            req.CardCVV,
		CardHolder:         req.CardHolder,
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
	c.Header("Connection", "keep-alive")

	// Forward SSE stream from agent service
	h.streamAgentJob(c, payload)
}

// SubmitOTP forwards an OTP code to the agent service for a running job.
func (h *AgentHandler) SubmitOTP(c *gin.Context) {
	if h.agentURL == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Agent service not configured"})
		return
	}

	jobID := c.Param("jobId")
	var body struct {
		OTP string `json:"otp"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	jsonBody, _ := json.Marshal(map[string]string{"otp": body.OTP})
	req, _ := http.NewRequest("POST", h.agentURL+"/otp/"+jobID, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Secret", h.agentSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	c.JSON(resp.StatusCode, gin.H{"ok": resp.StatusCode == 200})
}

func (h *AgentHandler) streamAgentJob(c *gin.Context, payload agentJobPayload) {
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		writeSSEEvent(c, map[string]any{"type": "error", "message": err.Error()})
		return
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), "POST", h.agentURL+"/pay", bytes.NewReader(jsonBody))
	if err != nil {
		writeSSEEvent(c, map[string]any{"type": "error", "message": err.Error()})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Secret", h.agentSecret)

	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		writeSSEEvent(c, map[string]any{"type": "error", "message": fmt.Sprintf("Agent unreachable: %v", err)})
		return
	}
	defer resp.Body.Close()

	// Forward SSE lines from agent to frontend
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		fmt.Fprintf(c.Writer, "%s\n", line)
		c.Writer.Flush()
	}
}

func writeSSEEvent(c *gin.Context, data map[string]any) {
	b, _ := json.Marshal(data)
	fmt.Fprintf(c.Writer, "data: %s\n\n", b)
	c.Writer.Flush()
}

// GetCredentials returns the user's credential list (labels only, no passwords)
// for the agent trigger UI to let user select which credentials to use.
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
		var label, url string
		if v, ok := d.Data()["label"].(string); ok {
			label = v
		}
		if v, ok := d.Data()["url"].(string); ok {
			url = v
		}
		result = append(result, credSummary{ID: d.Ref.ID, Label: label, URL: url})
	}
	c.JSON(http.StatusOK, result)
}

// Ensure io is used
var _ io.Reader
