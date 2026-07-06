package handlers

import (
	"context"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/dimitris-taskou/cost-calculator/internal/models"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type BillsHandler struct {
	fs *firestore.Client
}

func NewBillsHandler(fs *firestore.Client) *BillsHandler {
	return &BillsHandler{fs: fs}
}

func (h *BillsHandler) bills(ctx context.Context) *firestore.CollectionRef {
	return h.fs.Collection("bills")
}

func (h *BillsHandler) ListBills(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	ctx := context.Background()

	docs, err := h.bills(ctx).Where("user_id", "==", userID).OrderBy("due_date", firestore.Asc).Documents(ctx).GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	bills := make([]models.Bill, 0, len(docs))
	now := time.Now()
	batch := h.fs.Batch()
	hasBatch := false

	for _, d := range docs {
		var b models.Bill
		if err := d.DataTo(&b); err != nil {
			continue
		}
		b.ID = d.Ref.ID
		// Back-fill embedded provider for old bills that lack it
		if b.UserProvider.Provider.Name == "" && b.UserProviderID != "" {
			if upDoc, err := h.fs.Collection("userProviders").Doc(b.UserProviderID).Get(ctx); err == nil {
				var up models.UserProvider
				upDoc.DataTo(&up)
				up.ID = upDoc.Ref.ID
				b.UserProvider = up
			}
		}
		if b.Status == "pending" && now.After(b.DueDate) {
			b.Status = "overdue"
			batch.Update(d.Ref, []firestore.Update{{Path: "status", Value: "overdue"}})
			hasBatch = true
		}
		bills = append(bills, b)
	}

	if hasBatch {
		batch.Commit(ctx)
	}
	c.JSON(http.StatusOK, bills)
}

type createBillRequest struct {
	UserProviderID string     `json:"user_provider_id" binding:"required"`
	Amount         float64    `json:"amount" binding:"required,gt=0"`
	DueDate        time.Time  `json:"due_date" binding:"required"`
	IssuedDate     *time.Time `json:"issued_date"`
	Notes          string     `json:"notes"`
}

func (h *BillsHandler) CreateBill(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	ctx := context.Background()

	var req createBillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify user owns this userProvider and embed it
	upDoc, err := h.fs.Collection("userProviders").Doc(req.UserProviderID).Get(ctx)
	if err != nil || upDoc.Data()["user_id"] != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "provider not found"})
		return
	}
	var up models.UserProvider
	upDoc.DataTo(&up)
	up.ID = upDoc.Ref.ID

	now := time.Now()
	billStatus := "pending"
	if now.After(req.DueDate) {
		billStatus = "overdue"
	}

	bill := models.Bill{
		UserID:         userID,
		UserProviderID: req.UserProviderID,
		UserProvider:   up,
		Amount:         req.Amount,
		DueDate:        req.DueDate,
		IssuedDate:     req.IssuedDate,
		Status:         billStatus,
		Notes:          req.Notes,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	ref, _, err := h.bills(ctx).Add(ctx, bill)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create bill"})
		return
	}
	bill.ID = ref.ID
	c.JSON(http.StatusCreated, bill)
}

func (h *BillsHandler) UpdateBill(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	id := c.Param("id")
	ctx := context.Background()

	doc, err := h.bills(ctx).Doc(id).Get(ctx)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var bill models.Bill
	doc.DataTo(&bill)
	if bill.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	var req struct {
		Amount     *float64   `json:"amount"`
		DueDate    *time.Time `json:"due_date"`
		IssuedDate *time.Time `json:"issued_date"`
		Notes      *string    `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := []firestore.Update{{Path: "updated_at", Value: time.Now()}}
	if req.Amount != nil {
		bill.Amount = *req.Amount
		updates = append(updates, firestore.Update{Path: "amount", Value: *req.Amount})
	}
	if req.DueDate != nil {
		bill.DueDate = *req.DueDate
		updates = append(updates, firestore.Update{Path: "due_date", Value: *req.DueDate})
	}
	if req.IssuedDate != nil {
		bill.IssuedDate = req.IssuedDate
		updates = append(updates, firestore.Update{Path: "issued_date", Value: req.IssuedDate})
	}
	if req.Notes != nil {
		bill.Notes = *req.Notes
		updates = append(updates, firestore.Update{Path: "notes", Value: *req.Notes})
	}

	h.bills(ctx).Doc(id).Update(ctx, updates)
	bill.ID = id
	c.JSON(http.StatusOK, bill)
}

func (h *BillsHandler) DeleteBill(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	id := c.Param("id")
	ctx := context.Background()

	doc, err := h.bills(ctx).Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	var bill models.Bill
	doc.DataTo(&bill)
	if bill.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	h.bills(ctx).Doc(id).Delete(ctx)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

type bulkDeleteBillsRequest struct {
	IDs []string `json:"ids" binding:"required,min=1"`
}

func (h *BillsHandler) BulkDeleteBills(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	ctx := context.Background()

	var req bulkDeleteBillsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	batch := h.fs.Batch()
	deleted := 0
	for _, id := range req.IDs {
		doc, err := h.bills(ctx).Doc(id).Get(ctx)
		if err != nil {
			continue
		}
		var b models.Bill
		doc.DataTo(&b)
		if b.UserID != userID {
			continue
		}
		batch.Delete(h.bills(ctx).Doc(id))
		deleted++
	}

	if deleted > 0 {
		if _, err := batch.Commit(ctx); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"deleted": deleted})
}

func (h *BillsHandler) MarkPaid(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	id := c.Param("id")
	ctx := context.Background()

	doc, err := h.bills(ctx).Doc(id).Get(ctx)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var bill models.Bill
	doc.DataTo(&bill)
	if bill.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	now := time.Now()
	bill.Status = "paid"
	bill.PaidAt = &now
	bill.UpdatedAt = now
	bill.ID = id

	h.bills(ctx).Doc(id).Update(ctx, []firestore.Update{
		{Path: "status", Value: "paid"},
		{Path: "paid_at", Value: now},
		{Path: "updated_at", Value: now},
	})
	c.JSON(http.StatusOK, bill)
}

func (h *BillsHandler) MarkUnpaid(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	id := c.Param("id")
	ctx := context.Background()

	doc, err := h.bills(ctx).Doc(id).Get(ctx)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var bill models.Bill
	doc.DataTo(&bill)
	if bill.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	now := time.Now()
	newStatus := "pending"
	if now.After(bill.DueDate) {
		newStatus = "overdue"
	}
	bill.Status = newStatus
	bill.PaidAt = nil
	bill.UpdatedAt = now
	bill.ID = id

	h.bills(ctx).Doc(id).Update(ctx, []firestore.Update{
		{Path: "status", Value: newStatus},
		{Path: "paid_at", Value: nil},
		{Path: "updated_at", Value: now},
	})
	c.JSON(http.StatusOK, bill)
}

func (h *BillsHandler) Dashboard(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	ctx := context.Background()

	month := c.Query("month")
	if month == "" {
		month = time.Now().Format("2006-01")
	}

	now := time.Now()

	// Parse month boundaries for filtering
	monthStart, _ := time.Parse("2006-01", month)
	monthEnd := monthStart.AddDate(0, 1, 0)

	// Bills — fetch all, update overdue status, then filter by due_date month
	billDocs, _ := h.bills(ctx).Where("user_id", "==", userID).Documents(ctx).GetAll()
	var totalPending, totalOverdue, totalPaid int64
	var amountBills, amountOverdue float64
	upcoming := make([]models.Bill, 0)
	batch := h.fs.Batch()
	hasBatch := false

	for _, d := range billDocs {
		var b models.Bill
		d.DataTo(&b)
		b.ID = d.Ref.ID

		// Always update overdue status regardless of selected month
		if b.Status == "pending" && now.After(b.DueDate) {
			b.Status = "overdue"
			batch.Update(d.Ref, []firestore.Update{{Path: "status", Value: "overdue"}})
			hasBatch = true
		}

		// Only count bills whose due_date falls in the selected month
		if b.DueDate.Before(monthStart) || !b.DueDate.Before(monthEnd) {
			continue
		}

		switch b.Status {
		case "pending":
			totalPending++
			amountBills += b.Amount
			upcoming = append(upcoming, b)
		case "overdue":
			totalOverdue++
			amountBills += b.Amount
			amountOverdue += b.Amount
			upcoming = append(upcoming, b)
		case "paid":
			totalPaid++
		}
	}
	if hasBatch {
		batch.Commit(ctx)
	}

	// Sort upcoming by DueDate
	for i := 0; i < len(upcoming)-1; i++ {
		for j := i + 1; j < len(upcoming); j++ {
			if upcoming[j].DueDate.Before(upcoming[i].DueDate) {
				upcoming[i], upcoming[j] = upcoming[j], upcoming[i]
			}
		}
	}
	if len(upcoming) > 5 {
		upcoming = upcoming[:5]
	}

	// Income for month
	incomeDocs, _ := h.fs.Collection("income").
		Where("user_id", "==", userID).
		Where("month", "==", month).
		Documents(ctx).GetAll()
	var totalIncome float64
	for _, d := range incomeDocs {
		var inc models.Income
		d.DataTo(&inc)
		totalIncome += inc.Amount
	}

	// Planned expenses — fetch all planned for user, filter by month in memory
	allExpDocs, _ := h.fs.Collection("expenses").
		Where("user_id", "==", userID).
		Where("status", "==", "planned").
		Documents(ctx).GetAll()
	var amountExpensesPlanned float64
	plannedExpenses := make([]models.Expense, 0)
	for _, d := range allExpDocs {
		var e models.Expense
		d.DataTo(&e)
		e.ID = d.Ref.ID
		if e.Month == month {
			amountExpensesPlanned += e.Amount
			plannedExpenses = append(plannedExpenses, e)
		}
	}

	remaining := totalIncome - amountBills - amountExpensesPlanned

	c.JSON(http.StatusOK, gin.H{
		"month":                   month,
		"total_income":            totalIncome,
		"total_pending":           totalPending,
		"total_overdue":           totalOverdue,
		"total_paid":              totalPaid,
		"amount_bills":            amountBills,
		"amount_overdue":          amountOverdue,
		"amount_expenses_planned": amountExpensesPlanned,
		"total_expenses_planned":  int64(len(plannedExpenses)),
		"remaining":               remaining,
		"upcoming_bills":          upcoming,
		"planned_expenses":        plannedExpenses,
	})
}
