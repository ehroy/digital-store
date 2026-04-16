package handlers

import (
	"digistore/database"
	"digistore/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetPaymentConfig(c *gin.Context) {
	var cfg models.PaymentConfig
	database.DB.FirstOrCreate(&cfg, models.PaymentConfig{ID: 1})
	c.JSON(http.StatusOK, cfg)
}

func UpdatePaymentConfig(c *gin.Context) {
	var cfg models.PaymentConfig
	database.DB.FirstOrCreate(&cfg, models.PaymentConfig{ID: 1})
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cfg.ID = 1
	database.DB.Save(&cfg)
	c.JSON(http.StatusOK, cfg)
}

// GET /api/payment/methods — daftar metode yang tersedia untuk checkout
// Mengembalikan info berdasarkan gateway aktif & config manual
func GetPaymentMethods(c *gin.Context) {
	var cfg models.PaymentConfig
	database.DB.FirstOrCreate(&cfg, models.PaymentConfig{ID: 1})

	type Method struct {
		ID     string `json:"id"`
		Label  string `json:"label"`
		Detail string `json:"detail"`
		Icon   string `json:"icon"`
	}

	methods := []Method{}
	if cfg.SayaBayarEnabled || cfg.DompetXEnabled {
		methods = append(methods, Method{
			ID:     "qris",
			Label:  "QRIS",
			Detail: "Scan kode QR untuk pembayaran instan",
			Icon:   "📱",
		})
	}

	c.JSON(200, gin.H{
		"methods":        methods,
		"gateway_active": len(methods) > 0,
		"provider": func() string {
			if cfg.SayaBayarEnabled {
				return "sayabayar"
			}
			if cfg.DompetXEnabled {
				return "dompetx"
			}
			return "manual"
		}(),
	})
}
