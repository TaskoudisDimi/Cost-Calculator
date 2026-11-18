package handlers

import (
	"context"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/dimitris-taskou/cost-calculator/internal/models"
	"github.com/gin-gonic/gin"
)

type CardsHandler struct {
	fs *firestore.Client
}

func NewCardsHandler(fs *firestore.Client) *CardsHandler {
	return &CardsHandler{fs: fs}
}

func (h *CardsHandler) col() *firestore.CollectionRef {
	return h.fs.Collection("cards")
}

func (h *CardsHandler) List(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	ctx := context.Background()

	docs, err := h.col().Where("user_id", "==", userID).OrderBy("created_at", firestore.Asc).Documents(ctx).GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	cards := make([]models.Card, 0, len(docs))
	for _, d := range docs {
		var card models.Card
		if err := d.DataTo(&card); err != nil {
			continue
		}
		card.ID = d.Ref.ID
		cards = append(cards, card)
	}
	c.JSON(http.StatusOK, cards)
}

func (h *CardsHandler) Create(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	ctx := context.Background()

	var req struct {
		Nickname  string `json:"nickname" binding:"required"`
		LastFour  string `json:"last_four" binding:"required,len=4"`
		ExpiryMM  int    `json:"expiry_mm" binding:"required,min=1,max=12"`
		ExpiryYY  int    `json:"expiry_yy" binding:"required,min=2024"`
		CardType  string `json:"card_type"`
		IsDefault bool   `json:"is_default"`
		Color     string `json:"color"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()

	// If new card is default, clear default from all existing cards
	if req.IsDefault {
		h.clearDefaultCards(ctx, userID)
	}

	cardType := req.CardType
	if cardType == "" {
		cardType = "other"
	}
	color := req.Color
	if color == "" {
		color = "#6366f1"
	}

	card := models.Card{
		UserID:    userID,
		Nickname:  req.Nickname,
		LastFour:  req.LastFour,
		ExpiryMM:  req.ExpiryMM,
		ExpiryYY:  req.ExpiryYY,
		CardType:  cardType,
		IsDefault: req.IsDefault,
		Color:     color,
		CreatedAt: now,
		UpdatedAt: now,
	}

	ref, _, err := h.col().Add(ctx, card)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create card"})
		return
	}
	card.ID = ref.ID
	c.JSON(http.StatusCreated, card)
}

func (h *CardsHandler) Update(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	id := c.Param("id")
	ctx := context.Background()

	doc, err := h.col().Doc(id).Get(ctx)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var card models.Card
	doc.DataTo(&card)
	if card.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	var req struct {
		Nickname  *string `json:"nickname"`
		ExpiryMM  *int    `json:"expiry_mm"`
		ExpiryYY  *int    `json:"expiry_yy"`
		CardType  *string `json:"card_type"`
		IsDefault *bool   `json:"is_default"`
		Color     *string `json:"color"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := []firestore.Update{{Path: "updated_at", Value: time.Now()}}
	if req.Nickname != nil {
		card.Nickname = *req.Nickname
		updates = append(updates, firestore.Update{Path: "nickname", Value: *req.Nickname})
	}
	if req.ExpiryMM != nil {
		card.ExpiryMM = *req.ExpiryMM
		updates = append(updates, firestore.Update{Path: "expiry_mm", Value: *req.ExpiryMM})
	}
	if req.ExpiryYY != nil {
		card.ExpiryYY = *req.ExpiryYY
		updates = append(updates, firestore.Update{Path: "expiry_yy", Value: *req.ExpiryYY})
	}
	if req.CardType != nil {
		card.CardType = *req.CardType
		updates = append(updates, firestore.Update{Path: "card_type", Value: *req.CardType})
	}
	if req.Color != nil {
		card.Color = *req.Color
		updates = append(updates, firestore.Update{Path: "color", Value: *req.Color})
	}
	if req.IsDefault != nil {
		if *req.IsDefault {
			h.clearDefaultCards(ctx, userID)
		}
		card.IsDefault = *req.IsDefault
		updates = append(updates, firestore.Update{Path: "is_default", Value: *req.IsDefault})
	}

	if _, err := h.col().Doc(id).Update(ctx, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update card"})
		return
	}
	card.ID = id
	c.JSON(http.StatusOK, card)
}

func (h *CardsHandler) Delete(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	id := c.Param("id")
	ctx := context.Background()

	doc, err := h.col().Doc(id).Get(ctx)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var card models.Card
	doc.DataTo(&card)
	if card.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	if _, err := h.col().Doc(id).Delete(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not delete card"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *CardsHandler) clearDefaultCards(ctx context.Context, userID string) {
	docs, err := h.col().Where("user_id", "==", userID).Where("is_default", "==", true).Documents(ctx).GetAll()
	if err != nil {
		return
	}
	batch := h.fs.Batch()
	for _, d := range docs {
		batch.Update(d.Ref, []firestore.Update{{Path: "is_default", Value: false}})
	}
	if len(docs) > 0 {
		_, _ = batch.Commit(ctx)
	}
}
