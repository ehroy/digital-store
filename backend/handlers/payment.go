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

	var methods []Method

	// Jika ada gateway aktif: hanya tampilkan "gateway" sebagai satu opsi
	if cfg.SayaBayarEnabled && cfg.SayaBayarAPIKey != "" {
		methods = append(methods, Method{
			ID:     "gateway",
			Label:  "Bayar Online",
			Detail: "Pilih metode pembayaran di halaman berikutnya (transfer, QRIS, e-Wallet, dll)",
			Icon:   "💳",
		})
	} else if cfg.DompetXEnabled && cfg.DompetXAPIKey != "" {
		methods = append(methods, Method{
			ID:     "gateway",
			Label:  "Bayar Online",
			Detail: "Pilih metode pembayaran di halaman berikutnya",
			Icon:   "💳",
		})
	} else {
		// Mode manual — tampilkan sesuai yang dikonfigurasi
		if cfg.BankName != "" && cfg.BankNo != "" {
			methods = append(methods, Method{
				ID:     "bank",
				Label:  "Transfer Bank " + cfg.BankName,
				Detail: cfg.BankNo + " a/n " + cfg.BankAcc,
				Icon:   "🏦",
			})
		}
		if cfg.Dana != "" {
			methods = append(methods, Method{ID: "dana", Label: "DANA", Detail: cfg.Dana, Icon: "💚"})
		}
		if cfg.Gopay != "" {
			methods = append(methods, Method{ID: "gopay", Label: "GoPay", Detail: cfg.Gopay, Icon: "💚"})
		}
		if cfg.Ovo != "" {
			methods = append(methods, Method{ID: "ovo", Label: "OVO", Detail: cfg.Ovo, Icon: "💜"})
		}
		if cfg.QRIS {
			methods = append(methods, Method{ID: "qris", Label: "QRIS", Detail: "Scan kode QR untuk pembayaran instan", Icon: "📱"})
		}
		if cfg.Crypto && cfg.CryptoAddr != "" {
			methods = append(methods, Method{ID: "crypto", Label: "Cryptocurrency", Detail: cfg.CryptoAddr, Icon: "₿"})
		}
	}

	c.JSON(200, gin.H{
		"methods":        methods,
		"gateway_active": cfg.SayaBayarEnabled || cfg.DompetXEnabled,
		"provider": func() string {
			if cfg.SayaBayarEnabled { return "sayabayar" }
			if cfg.DompetXEnabled { return "dompetx" }
			return "manual"
		}(),
	})
}
