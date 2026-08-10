package handlers

import (
	"context"
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/dimitris-taskou/cost-calculator/internal/models"
	"github.com/gin-gonic/gin"
)

type ProviderTemplatesHandler struct {
	fs *firestore.Client
}

func NewProviderTemplatesHandler(fs *firestore.Client) *ProviderTemplatesHandler {
	return &ProviderTemplatesHandler{fs: fs}
}

func (h *ProviderTemplatesHandler) col() *firestore.CollectionRef {
	return h.fs.Collection("userProviderTemplates")
}

func (h *ProviderTemplatesHandler) List(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	ctx := context.Background()

	docs, err := h.col().Where("user_id", "==", userID).Documents(ctx).GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	templates := make([]models.UserProviderTemplate, 0, len(docs))
	for _, d := range docs {
		var t models.UserProviderTemplate
		if err := d.DataTo(&t); err == nil {
			t.ID = d.Ref.ID
			templates = append(templates, t)
		}
	}
	c.JSON(http.StatusOK, templates)
}

type createTemplateRequest struct {
	Name       string `json:"name" binding:"required,min=1"`
	Category   string `json:"category"`
	Color      string `json:"color"`
	PaymentURL string `json:"payment_url"`
}

func (h *ProviderTemplatesHandler) Create(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	ctx := context.Background()

	var req createTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cat := req.Category
	if cat == "" {
		cat = "other"
	}
	color := req.Color
	if color == "" {
		color = "#6B7280"
	}

	t := models.UserProviderTemplate{
		UserID:     userID,
		Name:       req.Name,
		Category:   cat,
		Color:      color,
		PaymentURL: req.PaymentURL,
	}

	ref, _, err := h.col().Add(ctx, t)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create template"})
		return
	}
	t.ID = ref.ID
	c.JSON(http.StatusCreated, t)
}

func (h *ProviderTemplatesHandler) Delete(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	id := c.Param("id")
	ctx := context.Background()

	doc, err := h.col().Doc(id).Get(ctx)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	var t models.UserProviderTemplate
	doc.DataTo(&t)
	if t.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	if _, err := h.col().Doc(id).Delete(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not delete template"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
