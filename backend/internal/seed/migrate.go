package seed

import (
	"context"
	"log"

	"cloud.google.com/go/firestore"
	"github.com/dimitris-taskou/cost-calculator/internal/models"
)

// MigrateExpenseMonths backfills the month field on any expense that lacks it.
// Safe to run on every startup (idempotent).
func MigrateExpenseMonths(ctx context.Context, fs *firestore.Client) {
	docs, err := fs.Collection("expenses").Documents(ctx).GetAll()
	if err != nil {
		log.Printf("migrate expenses: %v", err)
		return
	}

	var count int
	for _, doc := range docs {
		raw := doc.Data()
		if m, ok := raw["month"]; ok {
			if s, ok := m.(string); ok && s != "" {
				continue
			}
		}

		var e models.Expense
		if err := doc.DataTo(&e); err != nil || e.CreatedAt.IsZero() {
			continue
		}

		month := e.CreatedAt.Format("2006-01")
		if _, err := doc.Ref.Update(ctx, []firestore.Update{
			{Path: "month", Value: month},
		}); err != nil {
			log.Printf("migrate expenses: update %s: %v", doc.Ref.ID, err)
			continue
		}
		count++
	}

	if count > 0 {
		log.Printf("migrate: backfilled month on %d expenses", count)
	}
}
