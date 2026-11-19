package router

import (
	"cloud.google.com/go/firestore"
	firebasesdk "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/dimitris-taskou/cost-calculator/internal/email"
	"github.com/dimitris-taskou/cost-calculator/internal/handlers"
	"github.com/dimitris-taskou/cost-calculator/internal/middleware"
	"github.com/dimitris-taskou/cost-calculator/internal/nordigen"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func New(fs *firestore.Client, authClient *auth.Client, app *firebasesdk.App, anthropicKey, schedulerSecret, nordigenSecretID, nordigenSecretKey, nordigenRedirectURL, resendAPIKey, resendFromEmail string) *gin.Engine {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:    []string{"Origin", "Content-Type", "Authorization", "X-Scheduler-Secret"},
	}))

	nordigenSvc := nordigen.New(nordigenSecretID, nordigenSecretKey)

	cardsH := handlers.NewCardsHandler(fs)
	providersH := handlers.NewProvidersHandler(fs)
	templatesH := handlers.NewProviderTemplatesHandler(fs)
	billsH := handlers.NewBillsHandler(fs)
	expensesH := handlers.NewExpensesHandler(fs)
	incomeH := handlers.NewIncomeHandler(fs)
	scanH := handlers.NewScanHandler(anthropicKey, fs)
	var emailClient *email.Client
	if resendAPIKey != "" {
		emailClient = email.NewClient(resendAPIKey, resendFromEmail)
	}
	notifyH := handlers.NewNotifyHandler(fs, app, schedulerSecret, emailClient)
	settingsH := handlers.NewSettingsHandler(fs)
	bankH := handlers.NewBankHandler(fs, nordigenSvc, nordigenRedirectURL)
	membersH := handlers.NewMembersHandler(fs)
	agentH := handlers.NewAgentHandler(fs, anthropicKey)

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
		protected.POST("/bills/copy-month", billsH.CopyMonth)
		protected.POST("/bills/scan", scanH.ScanBill)
		protected.POST("/bills/parse-email", scanH.ParseEmail)
		protected.POST("/bills/scan-emails", scanH.ScanEmails)

		protected.GET("/cards", cardsH.List)
		protected.POST("/cards", cardsH.Create)
		protected.PUT("/cards/:id", cardsH.Update)
		protected.DELETE("/cards/:id", cardsH.Delete)
		protected.POST("/cards/:id/secure", cardsH.StoreSecure)
		protected.DELETE("/cards/:id/secure", cardsH.ClearSecure)

		protected.GET("/expenses", expensesH.List)
		protected.POST("/expenses", expensesH.Create)
		protected.PUT("/expenses/:id", expensesH.Update)
			protected.PATCH("/expenses/:id/buy", expensesH.MarkBought)
		protected.DELETE("/expenses/:id", expensesH.Delete)

		protected.GET("/income", incomeH.List)
		protected.POST("/income", incomeH.Create)
		protected.PUT("/income/:id", incomeH.Update)
			protected.DELETE("/income/:id", incomeH.Delete)
		protected.DELETE("/income", incomeH.BulkDelete)

		protected.GET("/dashboard", billsH.Dashboard)
		protected.GET("/analytics", billsH.Analytics)

		protected.POST("/notify/register", notifyH.RegisterToken)

		protected.GET("/settings", settingsH.GetSettings)
		protected.PUT("/settings", settingsH.UpdateSettings)
		protected.DELETE("/account", settingsH.DeleteAccount)

		protected.GET("/bank/institutions", bankH.ListInstitutions)
		protected.POST("/bank/connect", bankH.Connect)
		protected.GET("/bank/status", bankH.Status)
		protected.POST("/bank/sync", bankH.Sync)
		protected.DELETE("/bank/disconnect", bankH.Disconnect)

		protected.GET("/members", membersH.List)
		protected.POST("/members", membersH.Create)
		protected.PUT("/members/:id", membersH.Update)
		protected.DELETE("/members/:id", membersH.Delete)

		protected.POST("/agent/pay/:billId", agentH.Pay)
		protected.POST("/agent/otp/:jobId", agentH.SubmitOTP)
		protected.GET("/agent/credentials", agentH.GetCredentials)
	}

	return r
}
