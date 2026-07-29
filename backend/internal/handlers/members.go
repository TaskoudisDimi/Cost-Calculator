package handlers

import (
	"context"
	"net/http"
	"sort"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/dimitris-taskou/cost-calculator/internal/models"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MembersHandler struct {
	fs *firestore.Client
}

func NewMembersHandler(fs *firestore.Client) *MembersHandler {
	return &MembersHandler{fs: fs}
}

func (h *MembersHandler) List(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	ctx := context.Background()

	docs, err := h.fs.Collection("members").
		Where("user_id", "==", userID).
		Documents(ctx).GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	members := make([]models.Member, 0, len(docs))
	for _, d := range docs {
		var m models.Member
		if err := d.DataTo(&m); err == nil {
			m.ID = d.Ref.ID
			members = append(members, m)
		}
	}
	sort.Slice(members, func(i, j int) bool {
		return members[i].CreatedAt.Before(members[j].CreatedAt)
	})
	c.JSON(http.StatusOK, members)
}

func (h *MembersHandler) Create(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	ctx := context.Background()

	var req struct {
		Name  string `json:"name" binding:"required,min=1,max=50"`
		Color string `json:"color"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Color == "" {
		req.Color = "#6366f1"
	}

	now := time.Now()
	m := models.Member{
		UserID:    userID,
		Name:      req.Name,
		Color:     req.Color,
		CreatedAt: now,
		UpdatedAt: now,
	}
	ref, _, err := h.fs.Collection("members").Add(ctx, m)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create member"})
		return
	}
	m.ID = ref.ID
	c.JSON(http.StatusCreated, m)
}

func (h *MembersHandler) Update(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	id := c.Param("id")
	ctx := context.Background()

	doc, err := h.fs.Collection("members").Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	var m models.Member
	doc.DataTo(&m)
	if m.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	var req struct {
		Name  *string `json:"name"`
		Color *string `json:"color"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := []firestore.Update{{Path: "updated_at", Value: time.Now()}}
	if req.Name != nil {
		m.Name = *req.Name
		updates = append(updates, firestore.Update{Path: "name", Value: *req.Name})
	}
	if req.Color != nil {
		m.Color = *req.Color
		updates = append(updates, firestore.Update{Path: "color", Value: *req.Color})
	}

	if _, err := h.fs.Collection("members").Doc(id).Update(ctx, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	m.ID = id
	c.JSON(http.StatusOK, m)
}

func (h *MembersHandler) Delete(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	id := c.Param("id")
	ctx := context.Background()

	doc, err := h.fs.Collection("members").Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	var m models.Member
	doc.DataTo(&m)
	if m.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	if _, err := h.fs.Collection("members").Doc(id).Delete(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}
