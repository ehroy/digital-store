package main

import (
	"digistore/config"
	"digistore/database"
	"digistore/handlers"
	"digistore/middleware"
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	config.Load()
	database.Init()
	handlers.StartExpiryJob()
	handlers.StartProviderSyncJob()

	r := gin.Default()
	r.MaxMultipartMemory = 8 << 20

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{config.App.FrontendURL, "http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	r.Static("/uploads", "./uploads")

	api := r.Group("/api")
	{
		api.POST("/auth/login", handlers.Login)
		api.GET("/products", handlers.GetProducts)
		api.GET("/products/:id", handlers.GetProduct)
		api.POST("/orders", handlers.CreateOrder)
		api.GET("/payment/config", handlers.GetPaymentConfig)
		api.GET("/invoice/:no", handlers.GetInvoicePublic)
		api.POST("/invoice/:no/token", handlers.GenerateViewToken)
		api.POST("/orders/:invoice/check-payment", handlers.CheckPayment)
		api.GET("/payment/methods", handlers.GetPaymentMethods)
		api.GET("/contact", handlers.GetContactConfig)

		// Webhook dari masing-masing gateway (public, no JWT)
		api.POST("/webhook/qris", handlers.WebhookDompetX)
		api.POST("/webhook/dompetx", handlers.WebhookDompetX)
		api.POST("/webhook/sayabayar", handlers.WebhookSayaBayar)
		api.POST("/webhook/koalastore", handlers.WebhookKoalaStore)

		adm := api.Group("/admin", middleware.AdminAuth())
		{
			adm.GET("/dashboard", handlers.GetDashboard)

			adm.GET("/products", handlers.GetProducts)
			adm.POST("/products", handlers.CreateProduct)
			adm.PUT("/products/:id", handlers.UpdateProduct)
			adm.POST("/products/bulk", handlers.BulkProducts)
			adm.DELETE("/products/:id", handlers.DeleteProduct)
			adm.PATCH("/products/:id/toggle", handlers.ToggleProduct)
			adm.POST("/products/:id/image", handlers.UploadProductImage)
			adm.DELETE("/products/:id/image", handlers.DeleteProductImage)

			adm.GET("/products/:id/stock", handlers.GetProductStock)
			adm.POST("/products/:id/stock", handlers.AddProductStock)
			adm.PUT("/stock/:stockId", handlers.UpdateStockItem)
			adm.DELETE("/stock/:stockId", handlers.DeleteStockItem)
			adm.PATCH("/stock/:stockId/reset", handlers.ResetStockItem)

			adm.GET("/orders", handlers.GetOrders)
			adm.GET("/orders/:id", handlers.GetOrder)
			adm.PATCH("/orders/:id/status", handlers.UpdateOrderStatus)
			adm.POST("/orders/:id/deliver", handlers.ManualDeliver)

			adm.GET("/payment/config", handlers.GetPaymentConfig)
			adm.PUT("/payment/config", handlers.UpdatePaymentConfig)

			adm.GET("/scripts/logs", handlers.GetScriptLogs)
			adm.GET("/contact", handlers.GetContactConfig)
			adm.PUT("/contact", handlers.UpdateContactConfig)

			// External providers (KoalaStore, dll)
			adm.GET("/external-providers", handlers.GetExternalProviders)
			adm.POST("/external-providers", handlers.CreateExternalProvider)
			adm.PUT("/external-providers/:id", handlers.UpdateExternalProvider)
			adm.DELETE("/external-providers/:id", handlers.DeleteExternalProvider)
			adm.POST("/external-providers/:id/sync", handlers.SyncProviderProducts)
			adm.GET("/external-providers/:id/products", handlers.GetProviderProducts)
			adm.GET("/external-providers/:id/balance", handlers.GetProviderBalance)
			adm.POST("/external-providers/:id/import", handlers.ImportProviderProducts)
			adm.POST("/external-providers/:id/apply-default-markup", handlers.ApplyProviderDefaultMarkup)
			adm.POST("/external-providers/sync-prices", handlers.SyncProviderPrices)

			// Stock providers (pull log)
			adm.GET("/providers", handlers.GetProviders)
			adm.POST("/providers", handlers.CreateProvider)
			adm.PUT("/providers/:id", handlers.UpdateProvider)
			adm.DELETE("/providers/:id", handlers.DeleteProvider)
			adm.POST("/providers/:id/pull", handlers.PullFromProvider)
			adm.GET("/providers/:id/logs", handlers.GetProviderLogs)
			adm.GET("/pull-logs", handlers.GetAllPullLogs)
		}
	}

	log.Printf("🚀 DigiStore running on :%s", config.App.Port)
	r.Run(":" + config.App.Port)
}
