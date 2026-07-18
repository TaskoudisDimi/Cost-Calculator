package router

import (
	"cloud.google.com/go/firestore"
	"firebase.google.com/go/v4/auth"
	"github.com/dimitris-taskou/cost-calculator/internal/handlers"
	"github.com/dimitris-taskou/cost-calculator/internal/middleware"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func New(fs *firestore.Client, authClient *auth.Client, anthropicKey string) *gin.Engine {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:    []string{"Origin", "Content-Type", "Authorization"},
	}))

	providersH := handlers.NewProvidersHandler(fs)
	templatesH := handlers.NewProviderTemplatesHandler(fs)
	billsH := handlers.NewBillsHandler(fs)
	expensesH := handlers.NewExpensesHandler(fs)
	incomeH := handlers.NewIncomeHandler(fs)
	scanH := handlers.NewScanHandler(anthropicKey)

	authMW := middleware.Auth(authClient)

	api := r.Group("/api")
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

		protected.GET("/expenses", expensesH.List)
		protected.POST("/expenses", expensesH.Create)
		protected.PATCH("/expenses/:id/buy", expensesH.MarkBought)
		protected.DELETE("/expenses/:id", expensesH.Delete)

		protected.GET("/income", incomeH.List)
		protected.POST("/income", incomeH.Create)
		protected.DELETE("/income/:id", incomeH.Delete)
		protected.DELETE("/income", incomeH.BulkDelete)

		protected.GET("/dashboard", billsH.Dashboard)
	}

	return r
}
