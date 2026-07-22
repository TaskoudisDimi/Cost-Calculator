package router

import (
	"cloud.google.com/go/firestore"
	firebasesdk "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/dimitris-taskou/cost-calculator/internal/handlers"
	"github.com/dimitris-taskou/cost-calculator/internal/middleware"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func New(fs *firestore.Client, authClient *auth.Client, app *firebasesdk.App, anthropicKey, schedulerSecret string) *gin.Engine {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:    []string{"Origin", "Content-Type", "Authorization", "X-Scheduler-Secret"},
	}))

	providersH := handlers.NewProvidersHandler(fs)
	templatesH := handlers.NewProviderTemplatesHandler(fs)
	billsH := handlers.NewBillsHandler(fs)
	expensesH := handlers.NewExpensesHandler(fs)
	incomeH := handlers.NewIncomeHandler(fs)
	scanH := handlers.NewScanHandler(anthropicKey)
	notifyH := handlers.NewNotifyHandler(fs, app, schedulerSecret)
	settingsH := handlers.NewSettingsHandler(fs)

	authMW := middleware.Auth(authClient)

	api := r.Group("/api")

	// Scheduler-called endpoint (authenticated via X-Scheduler-Secret header)
	api.POST("/notify/send-reminders", notifyH.SendReminders)

	protected := api.Group("/")
	protected.Use(authMW)
	{
		protected.GET("/providers", providersH.ListProviders)
		protected.GET("/user/providers", providersH.ListUserProviders)
		protected.POST("/user/providers", providersH.AddUserProvider)
		protected.DELETE("/user/providers/:id", providersH.DeleteUserProvider)

		protected.GET("/user/provider-templates", templatesH.List)
		protected.POST("/user/provider-templates", templatesH.Create)
		protected.DELETE("/user/provider-templates/:id", templatesH.Delete)

		protected.GET("/bills", billsH.ListBills)
		protected.POST("/bills", billsH.CreateBill)
		protected.PUT("/bills/:id", billsH.UpdateBill)
		protected.DELETE("/bills/:id", billsH.DeleteBill)
		protected.DELETE("/bills", billsH.BulkDeleteBills)
		protected.PATCH("/bills/:id/pay", billsH.MarkPaid)
		protected.PATCH("/bills/:id/unpay", billsH.MarkUnpaid)
		protected.POST("/bills/scan", scanH.ScanBill)
		protected.POST("/bills/parse-email", scanH.ParseEmail)

		protected.GET("/expenses", expensesH.List)
		protected.POST("/expenses", expensesH.Create)
		protected.PATCH("/expenses/:id/buy", expensesH.MarkBought)
		protected.DELETE("/expenses/:id", expensesH.Delete)

		protected.GET("/income", incomeH.List)
		protected.POST("/income", incomeH.Create)
		protected.DELETE("/income/:id", incomeH.Delete)
		protected.DELETE("/income", incomeH.BulkDelete)

		protected.GET("/dashboard", billsH.Dashboard)

		protected.POST("/notify/register", notifyH.RegisterToken)

		protected.GET("/settings", settingsH.GetSettings)
		protected.PUT("/settings", settingsH.UpdateSettings)
		protected.DELETE("/account", settingsH.DeleteAccount)
	}

	return r
}
