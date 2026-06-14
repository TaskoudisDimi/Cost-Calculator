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

type IncomeHandler struct {
	fs *firestore.Client
}

func NewIncomeHandler(fs *firestore.Client) *IncomeHandler {
	return &IncomeHandler{fs: fs}
}

func (h *IncomeHandler) List(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	ctx := context.Background()

	q := h.fs.Collection("income").Where("user_id", "==", userID)
	if month := c.Query("month"); month != "" {
		q = q.Where("month", "==", month)
	}

	docs, err := q.OrderBy("created_at", firestore.Desc).Documents(ctx).GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	incomes := make([]models.Income, 0, len(docs))
	for _, d := range docs {
		var i models.Income
		if err := d.DataTo(&i); err == nil {
			i.ID = d.Ref.ID
			incomes = append(incomes, i)
		}
	}
	c.JSON(http.StatusOK, incomes)
}

func (h *IncomeHandler) Create(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	ctx := context.Background()

	var req struct {
		Description string  `json:"description" binding:"required,min=1"`
		Amount      float64 `json:"amount" binding:"required,gt=0"`
		Month       string  `json:"month" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	income := models.Income{
		UserID:      userID,
		Description: req.Description,
		Amount:      req.Amount,
		Month:       req.Month,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	ref, _, err := h.fs.Collection("income").Add(ctx, income)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create income"})
		return
	}
	income.ID = ref.ID
	c.JSON(http.StatusCreated, income)
}

func (h *IncomeHandler) Delete(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	id := c.Param("id")
	ctx := context.Background()

	doc, err := h.fs.Collection("income").Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	var income models.Income
	doc.DataTo(&income)
	if income.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	h.fs.Collection("income").Doc(id).Delete(ctx)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *IncomeHandler) BulkDelete(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	ctx := context.Background()

	var req struct {
		IDs []string `json:"ids" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	batch := h.fs.Batch()
	deleted := 0
	for _, id := range req.IDs {
		doc, err := h.fs.Collection("income").Doc(id).Get(ctx)
		if err != nil {
			continue
		}
		var inc models.Income
		doc.DataTo(&inc)
		if inc.UserID != userID {
			continue
		}
		batch.Delete(h.fs.Collection("income").Doc(id))
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
