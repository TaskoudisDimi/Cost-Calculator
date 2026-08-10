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

type ExpensesHandler struct {
	fs *firestore.Client
}

func NewExpensesHandler(fs *firestore.Client) *ExpensesHandler {
	return &ExpensesHandler{fs: fs}
}

func (h *ExpensesHandler) col() *firestore.CollectionRef {
	return h.fs.Collection("expenses")
}

func (h *ExpensesHandler) List(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	ctx := context.Background()
	month := c.Query("month")

	docs, err := h.col().Where("user_id", "==", userID).OrderBy("created_at", firestore.Desc).Documents(ctx).GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	expenses := make([]models.Expense, 0, len(docs))
	for _, d := range docs {
		var e models.Expense
		if err := d.DataTo(&e); err == nil {
			e.ID = d.Ref.ID
			if month == "" || e.Month == month {
				expenses = append(expenses, e)
			}
		}
	}
	c.JSON(http.StatusOK, expenses)
}

type createExpenseRequest struct {
	Name        string     `json:"name" binding:"required,min=1"`
	Amount      float64    `json:"amount" binding:"required,gt=0"`
	Category    string     `json:"category"`
	Month       string     `json:"month"` // YYYY-MM; defaults to current month if empty
	Store       string     `json:"store"`
	Notes       string     `json:"notes"`
	PlannedDate *time.Time `json:"planned_date"`
}

func (h *ExpensesHandler) Create(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	ctx := context.Background()

	var req createExpenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cat := req.Category
	if cat == "" {
		cat = "other"
	}

	month := req.Month
	if month == "" {
		if req.PlannedDate != nil {
			month = req.PlannedDate.Format("2006-01")
		} else {
			month = time.Now().Format("2006-01")
		}
	}

	now := time.Now()
	expense := models.Expense{
		UserID:      userID,
		Month:       month,
		Name:        req.Name,
		Amount:      req.Amount,
		Category:    cat,
		Status:      "planned",
		Store:       req.Store,
		Notes:       req.Notes,
		PlannedDate: req.PlannedDate,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	ref, _, err := h.col().Add(ctx, expense)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create expense"})
		return
	}
	expense.ID = ref.ID
	c.JSON(http.StatusCreated, expense)
}

func (h *ExpensesHandler) MarkBought(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	id := c.Param("id")
	ctx := context.Background()

	doc, err := h.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	var expense models.Expense
	doc.DataTo(&expense)
	if expense.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	now := time.Now()
	expense.Status = "bought"
	expense.BoughtAt = &now
	expense.UpdatedAt = now
	expense.ID = id

	if _, err := h.col().Doc(id).Update(ctx, []firestore.Update{
		{Path: "status", Value: "bought"},
		{Path: "bought_at", Value: now},
		{Path: "updated_at", Value: now},
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update expense"})
		return
	}
	c.JSON(http.StatusOK, expense)
}

func (h *ExpensesHandler) Delete(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	id := c.Param("id")
	ctx := context.Background()

	doc, err := h.col().Doc(id).Get(ctx)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var expense models.Expense
	doc.DataTo(&expense)
	if expense.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	h.col().Doc(id).Delete(ctx)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
