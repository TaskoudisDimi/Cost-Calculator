package handlers

import (
	"context"
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/dimitris-taskou/cost-calculator/internal/models"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ProvidersHandler struct {
	fs *firestore.Client
}

func NewProvidersHandler(fs *firestore.Client) *ProvidersHandler {
	return &ProvidersHandler{fs: fs}
}

func (h *ProvidersHandler) ListProviders(c *gin.Context) {
	ctx := context.Background()
	docs, err := h.fs.Collection("providers").OrderBy("name", firestore.Asc).Documents(ctx).GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	providers := make([]models.Provider, 0, len(docs))
	for _, d := range docs {
		var p models.Provider
		if err := d.DataTo(&p); err == nil {
			p.ID = d.Ref.ID
			providers = append(providers, p)
		}
	}
	c.JSON(http.StatusOK, providers)
}

func (h *ProvidersHandler) ListUserProviders(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	ctx := context.Background()

	docs, err := h.fs.Collection("userProviders").Where("user_id", "==", userID).Documents(ctx).GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ups := make([]models.UserProvider, 0, len(docs))
	for _, d := range docs {
		var up models.UserProvider
		if err := d.DataTo(&up); err == nil {
			up.ID = d.Ref.ID
			// Back-fill embedded provider for old documents that lack it
			if up.Provider.Name == "" && up.ProviderID != "" {
				if pDoc, err := h.fs.Collection("providers").Doc(up.ProviderID).Get(ctx); err == nil {
					pDoc.DataTo(&up.Provider)
					up.Provider.ID = pDoc.Ref.ID
					_, _ = d.Ref.Update(ctx, []firestore.Update{{Path: "provider", Value: up.Provider}})
				} else if tDoc, err2 := h.fs.Collection("userProviderTemplates").Doc(up.ProviderID).Get(ctx); err2 == nil {
					var tmpl models.UserProviderTemplate
					tDoc.DataTo(&tmpl)
					up.Provider = models.Provider{
						ID: tDoc.Ref.ID, Name: tmpl.Name,
						Category: tmpl.Category, Color: tmpl.Color, PaymentURL: tmpl.PaymentURL,
					}
					_, _ = d.Ref.Update(ctx, []firestore.Update{{Path: "provider", Value: up.Provider}})
				}
			}
			ups = append(ups, up)
		}
	}
	c.JSON(http.StatusOK, ups)
}

type addUserProviderRequest struct {
	// Seeded provider — leave empty for a custom entry
	ProviderID string `json:"provider_id"`
	// Custom provider fields (required when provider_id is empty)
	CustomName       string `json:"custom_name"`
	CustomCategory   string `json:"custom_category"`
	CustomColor      string `json:"custom_color"`
	CustomPaymentURL string `json:"custom_payment_url"`
	// Common
	Nickname   string `json:"nickname"`
	AccountNum string `json:"account_num"`
}

func (h *ProvidersHandler) AddUserProvider(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	ctx := context.Background()

	var req addUserProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var provider models.Provider

	if req.ProviderID != "" {
		// Try seeded providers first
		pDoc, err := h.fs.Collection("providers").Doc(req.ProviderID).Get(ctx)
		if err != nil {
			// Fallback: check user provider templates
			tDoc, err2 := h.fs.Collection("userProviderTemplates").Doc(req.ProviderID).Get(ctx)
			if err2 != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
				return
			}
			var tmpl models.UserProviderTemplate
			if err := tDoc.DataTo(&tmpl); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if tmpl.UserID != userID {
				c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
				return
			}
			provider = models.Provider{
				ID:         tDoc.Ref.ID,
				Name:       tmpl.Name,
				Category:   tmpl.Category,
				Color:      tmpl.Color,
				PaymentURL: tmpl.PaymentURL,
			}
		} else {
			if err := pDoc.DataTo(&provider); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			provider.ID = pDoc.Ref.ID
		}
	} else {
		// Custom provider
		if req.CustomName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "custom_name is required"})
			return
		}
		cat := req.CustomCategory
		if cat == "" {
			cat = "other"
		}
		color := req.CustomColor
		if color == "" {
			color = "#6B7280"
		}
		provider = models.Provider{
			Name:       req.CustomName,
			Category:   cat,
			Color:      color,
			PaymentURL: req.CustomPaymentURL,
		}
	}

	up := models.UserProvider{
		UserID:     userID,
		ProviderID: req.ProviderID,
		Provider:   provider,
		Nickname:   req.Nickname,
		AccountNum: req.AccountNum,
	}

	ref, _, err := h.fs.Collection("userProviders").Add(ctx, up)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not add provider"})
		return
	}
	up.ID = ref.ID
	c.JSON(http.StatusCreated, up)
}

func (h *ProvidersHandler) DeleteUserProvider(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	id := c.Param("id")
	ctx := context.Background()

	doc, err := h.fs.Collection("userProviders").Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	var up models.UserProvider
	doc.DataTo(&up)
	if up.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	if _, err := h.fs.Collection("userProviders").Doc(id).Delete(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not delete provider"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
