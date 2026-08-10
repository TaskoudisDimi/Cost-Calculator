package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	firebasesdk "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/dimitris-taskou/cost-calculator/internal/email"
	"github.com/dimitris-taskou/cost-calculator/internal/models"
	"github.com/gin-gonic/gin"
)

type NotifyHandler struct {
	fs              *firestore.Client
	app             *firebasesdk.App
	schedulerSecret string
	emailClient     *email.Client
}

func NewNotifyHandler(fs *firestore.Client, app *firebasesdk.App, schedulerSecret string, emailClient *email.Client) *NotifyHandler {
	return &NotifyHandler{fs: fs, app: app, schedulerSecret: schedulerSecret, emailClient: emailClient}
}

// POST /api/notify/register — save FCM token for current user
func (h *NotifyHandler) RegisterToken(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()
	_, err := h.fs.Collection("fcmTokens").Doc(userID).Set(ctx, map[string]interface{}{
		"token":      req.Token,
		"user_id":    userID,
		"updated_at": time.Now(),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// POST /api/notify/send-reminders — called by Cloud Scheduler
func (h *NotifyHandler) SendReminders(c *gin.Context) {
	got := c.GetHeader("X-Scheduler-Secret")
	if h.schedulerSecret == "" || got != h.schedulerSecret {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if h.app == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "firebase app not available"})
		return
	}

	ctx := context.Background()
	msgClient, err := h.app.Messaging(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "messaging not available: " + err.Error()})
		return
	}

	tokenDocs, err := h.fs.Collection("fcmTokens").Documents(ctx).GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	threeDaysLater := now.Add(3 * 24 * time.Hour)
	sent := 0
	errors := 0

	for _, tokenDoc := range tokenDocs {
		var td struct {
			Token  string `firestore:"token"`
			UserID string `firestore:"user_id"`
		}
		if err := tokenDoc.DataTo(&td); err != nil || td.Token == "" {
			continue
		}

		billDocs, err := h.fs.Collection("bills").
			Where("user_id", "==", td.UserID).
			Documents(ctx).GetAll()
		if err != nil {
			continue
		}

		for _, billDoc := range billDocs {
			var bill models.Bill
			if err := billDoc.DataTo(&bill); err != nil {
				continue
			}
			if bill.Status == "paid" {
				continue
			}
			if bill.DueDate.Before(now) || bill.DueDate.After(threeDaysLater) {
				continue
			}

			daysLeft := int(bill.DueDate.Sub(now).Hours()/24) + 1
			providerName := bill.UserProvider.Nickname
			if providerName == "" {
				providerName = bill.UserProvider.Provider.Name
			}
			if providerName == "" {
				providerName = "Λογαριασμός"
			}

			body := fmt.Sprintf("%s λήγει σε %d μέρα/ες (€%.2f)", providerName, daysLeft, bill.Amount)
			if bill.Status == "overdue" {
				body = fmt.Sprintf("%s είναι ληξιπρόθεσμος (€%.2f)", providerName, bill.Amount)
			}

			msg := &messaging.Message{
				Token: td.Token,
				Notification: &messaging.Notification{
					Title: "Υπενθύμιση πληρωμής",
					Body:  body,
				},
				Data: map[string]string{
					"bill_id": billDoc.Ref.ID,
					"type":    "bill_reminder",
				},
				Webpush: &messaging.WebpushConfig{
					Notification: &messaging.WebpushNotification{
						Icon: "/favicon.png",
					},
				},
			}

			if _, err := msgClient.Send(ctx, msg); err != nil {
				errors++
			} else {
				sent++
			}
		}
	}

	// Email reminders via Resend
	emailSent := 0
	if h.emailClient != nil {
		authClient, authErr := h.app.Auth(ctx)
		if authErr == nil {
			// group upcoming bills by user_id
			allBillDocs, err := h.fs.Collection("bills").Documents(ctx).GetAll()
			if err == nil {
				byUser := map[string][]email.BillItem{}
				for _, billDoc := range allBillDocs {
					var bill models.Bill
					if err := billDoc.DataTo(&bill); err != nil || bill.Status == "paid" {
						continue
					}
					if bill.DueDate.Before(now) || bill.DueDate.After(threeDaysLater) {
						continue
					}
					name := bill.UserProvider.Nickname
					if name == "" {
						name = bill.UserProvider.Provider.Name
					}
					if name == "" {
						name = "Λογαριασμός"
					}
					byUser[bill.UserID] = append(byUser[bill.UserID], email.BillItem{
						Name:    name,
						Amount:  bill.Amount,
						DueDate: bill.DueDate.Format("02/01/2006"),
					})
				}
				for uid, items := range byUser {
					userRecord, err := authClient.GetUser(ctx, uid)
					if err != nil || userRecord.Email == "" {
						continue
					}
					html := email.ReminderHTML(items)
					if err := h.emailClient.Send(userRecord.Email, "Υπενθύμιση πληρωμής λογαριασμών", html); err == nil {
						emailSent++
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"sent": sent, "errors": errors, "email_sent": emailSent})
}
