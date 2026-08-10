package handlers

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/dimitris-taskou/cost-calculator/internal/models"
	"github.com/dimitris-taskou/cost-calculator/internal/nordigen"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type BankHandler struct {
	fs          *firestore.Client
	nordigen    *nordigen.Service
	redirectURL string
}

func NewBankHandler(fs *firestore.Client, svc *nordigen.Service, redirectURL string) *BankHandler {
	return &BankHandler{fs: fs, nordigen: svc, redirectURL: redirectURL}
}

// GET /bank/institutions
func (h *BankHandler) ListInstitutions(c *gin.Context) {
	if !h.nordigen.Configured() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "bank integration not configured"})
		return
	}
	institutions, err := h.nordigen.ListInstitutions(c.Request.Context(), "GR")
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, institutions)
}

// POST /bank/connect   body: { institution_id, institution_name }
func (h *BankHandler) Connect(c *gin.Context) {
	if !h.nordigen.Configured() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "bank integration not configured"})
		return
	}

	userID := c.MustGet("user_id").(string)
	var body struct {
		InstitutionID   string `json:"institution_id" binding:"required"`
		InstitutionName string `json:"institution_name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	existing, _ := h.fs.Collection("bankConnections").Where("user_id", "==", userID).Limit(1).Documents(ctx).GetAll()
	for _, doc := range existing {
		var conn models.BankConnection
		if err := doc.DataTo(&conn); err == nil && conn.RequisitionID != "" {
			_ = h.nordigen.DeleteRequisition(ctx, conn.RequisitionID)
		}
		_, _ = doc.Ref.Delete(ctx)
	}

	eua, err := h.nordigen.CreateEUA(ctx, body.InstitutionID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to create bank agreement: " + err.Error()})
		return
	}

	reference := fmt.Sprintf("bt-%s-%d", userID[:8], time.Now().Unix())
	req, err := h.nordigen.CreateRequisition(ctx, body.InstitutionID, eua.ID, h.redirectURL, reference)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to create requisition: " + err.Error()})
		return
	}

	conn := models.BankConnection{
		UserID:          userID,
		RequisitionID:   req.ID,
		InstitutionID:   body.InstitutionID,
		InstitutionName: body.InstitutionName,
		Status:          req.Status,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	_, _, err = h.fs.Collection("bankConnections").Add(ctx, conn)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save connection"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"link": req.Link, "requisition_id": req.ID})
}

// GET /bank/status
func (h *BankHandler) Status(c *gin.Context) {
	if !h.nordigen.Configured() {
		c.JSON(http.StatusOK, gin.H{"connected": false})
		return
	}

	userID := c.MustGet("user_id").(string)
	ctx := c.Request.Context()

	docs, err := h.fs.Collection("bankConnections").Where("user_id", "==", userID).Limit(1).Documents(ctx).GetAll()
	if err != nil || len(docs) == 0 {
		c.JSON(http.StatusOK, gin.H{"connected": false})
		return
	}

	var conn models.BankConnection
	if err := docs[0].DataTo(&conn); err != nil {
		c.JSON(http.StatusOK, gin.H{"connected": false})
		return
	}

	req, err := h.nordigen.GetRequisition(ctx, conn.RequisitionID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"connected":        conn.Status == "LN",
			"status":           conn.Status,
			"institution_name": conn.InstitutionName,
			"accounts":         len(conn.AccountIDs),
		})
		return
	}

	if req.Status != conn.Status || len(req.Accounts) != len(conn.AccountIDs) {
		_, _ = docs[0].Ref.Set(ctx, map[string]any{
			"status":      req.Status,
			"account_ids": req.Accounts,
			"updated_at":  time.Now(),
		}, firestore.MergeAll)
	}

	c.JSON(http.StatusOK, gin.H{
		"connected":        req.Status == "LN",
		"status":           req.Status,
		"institution_name": conn.InstitutionName,
		"accounts":         len(req.Accounts),
		"requisition_id":   req.ID,
	})
}

// POST /bank/sync — fetch transactions and try to auto-mark bills as paid
func (h *BankHandler) Sync(c *gin.Context) {
	if !h.nordigen.Configured() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "bank integration not configured"})
		return
	}

	userID := c.MustGet("user_id").(string)
	ctx := c.Request.Context()

	docs, err := h.fs.Collection("bankConnections").Where("user_id", "==", userID).Limit(1).Documents(ctx).GetAll()
	if err != nil || len(docs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no bank connection found"})
		return
	}

	var conn models.BankConnection
	if err := docs[0].DataTo(&conn); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read connection"})
		return
	}

	req, err := h.nordigen.GetRequisition(ctx, conn.RequisitionID)
	if err != nil || req.Status != "LN" || len(req.Accounts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bank not fully connected yet"})
		return
	}

	dateFrom := time.Now().AddDate(0, -3, 0).Format("2006-01-02")
	var allTx []nordigen.Transaction
	for _, accID := range req.Accounts {
		txs, err := h.nordigen.GetTransactions(ctx, accID, dateFrom)
		if err != nil {
			continue
		}
		allTx = append(allTx, txs...)
	}

	billDocs, err := h.fs.Collection("bills").
		Where("user_id", "==", userID).
		Where("status", "in", []string{"pending", "overdue"}).
		Documents(ctx).GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch bills"})
		return
	}

	type billEntry struct {
		ref    *firestore.DocumentRef
		bill   models.Bill
		amount float64
	}
	var pendingBills []billEntry
	for _, d := range billDocs {
		var bill models.Bill
		if err := d.DataTo(&bill); err != nil {
			continue
		}
		pendingBills = append(pendingBills, billEntry{ref: d.Ref, bill: bill, amount: bill.Amount})
	}

	matched := 0
	totalTx := len(allTx)
	now := time.Now()

	for _, tx := range allTx {
		amtStr := strings.TrimSpace(tx.Amount.Amount)
		if amtStr == "" {
			continue
		}
		amt, err := strconv.ParseFloat(amtStr, 64)
		if err != nil || amt >= 0 {
			continue
		}
		txAmt := math.Abs(amt)
		creditor := strings.ToLower(tx.CreditorName + " " + tx.RemittanceInfo)

		for i, entry := range pendingBills {
			providerName := strings.ToLower(entry.bill.UserProvider.Provider.Name)
			amountMatch := math.Abs(txAmt-entry.amount) < 1.0
			nameMatch := providerName != "" && strings.Contains(creditor, providerName)
			codeMatch := entry.bill.PaymentCode != "" && strings.Contains(creditor, strings.ToLower(entry.bill.PaymentCode))

			if amountMatch && (nameMatch || codeMatch) {
				_, err := entry.ref.Set(ctx, map[string]any{
					"status":     "paid",
					"paid_at":    now,
					"updated_at": now,
				}, firestore.MergeAll)
				if err == nil {
					matched++
					pendingBills = append(pendingBills[:i], pendingBills[i+1:]...)
					break
				}
			}
		}
	}

	_, _ = docs[0].Ref.Set(ctx, map[string]any{
		"last_synced_at": now,
		"updated_at":     now,
	}, firestore.MergeAll)

	c.JSON(http.StatusOK, gin.H{
		"transactions_found": totalTx,
		"bills_matched":      matched,
	})
}

// DELETE /bank/disconnect
func (h *BankHandler) Disconnect(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	ctx := c.Request.Context()

	docs, err := h.fs.Collection("bankConnections").Where("user_id", "==", userID).Documents(ctx).GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find connection"})
		return
	}

	for _, doc := range docs {
		var conn models.BankConnection
		if err := doc.DataTo(&conn); err == nil && conn.RequisitionID != "" && h.nordigen.Configured() {
			_ = h.nordigen.DeleteRequisition(ctx, conn.RequisitionID)
		}
		if _, err := doc.Ref.Delete(ctx); err != nil {
			if status.Code(err) == codes.NotFound {
				continue
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disconnect bank"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"disconnected": true})
}
