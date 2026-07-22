package handlers

import (
	"context"
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SettingsHandler struct {
	fs *firestore.Client
}

func NewSettingsHandler(fs *firestore.Client) *SettingsHandler {
	return &SettingsHandler{fs: fs}
}

type UserSettings struct {
	NotifEnabled bool   `json:"notif_enabled" firestore:"notif_enabled"`
	NotifDays    int    `json:"notif_days" firestore:"notif_days"`
	Currency     string `json:"currency" firestore:"currency"`
}

func (h *SettingsHandler) GetSettings(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	ctx := context.Background()

	doc, err := h.fs.Collection("userSettings").Doc(userID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			c.JSON(http.StatusOK, UserSettings{NotifEnabled: false, NotifDays: 3, Currency: "EUR"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var s UserSettings
	doc.DataTo(&s)
	if s.Currency == "" {
		s.Currency = "EUR"
	}
	if s.NotifDays == 0 {
		s.NotifDays = 3
	}
	c.JSON(http.StatusOK, s)
}

func (h *SettingsHandler) UpdateSettings(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	ctx := context.Background()

	var req UserSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Currency == "" {
		req.Currency = "EUR"
	}
	if req.NotifDays <= 0 {
		req.NotifDays = 3
	}

	if _, err := h.fs.Collection("userSettings").Doc(userID).Set(ctx, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, req)
}

// DeleteAccount removes all user data from Firestore (Firebase Auth deletion happens on the frontend)
func (h *SettingsHandler) DeleteAccount(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	ctx := context.Background()

	collections := []string{"bills", "userProviders", "userProviderTemplates", "expenses", "income", "fcmTokens"}

	for _, col := range collections {
		docs, err := h.fs.Collection(col).Where("user_id", "==", userID).Documents(ctx).GetAll()
		if err != nil {
			continue
		}
		batch := h.fs.Batch()
		for _, d := range docs {
			batch.Delete(d.Ref)
		}
		if len(docs) > 0 {
			batch.Commit(ctx)
		}
	}

	// Delete settings doc
	h.fs.Collection("userSettings").Doc(userID).Delete(ctx)

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
