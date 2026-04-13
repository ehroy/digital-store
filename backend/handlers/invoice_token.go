package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"digistore/config"
	"digistore/database"
	"digistore/models"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// GenerateViewToken — POST /api/invoice/:no/token
// Membuat token akses invoice tanpa perlu email ulang.
// Dipanggil segera setelah checkout berhasil, token disimpan di sessionStorage.
// Token = HMAC-SHA256(invoice_no:buyer_email, jwt_secret)[:32]
func GenerateViewToken(c *gin.Context) {
	no := c.Param("no")

	var body struct {
		Email string `json:"email" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email wajib diisi"})
		return
	}

	var order models.Order
	if err := database.DB.Where("invoice_no = ?", no).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "invoice tidak ditemukan"})
		return
	}
	if !strings.EqualFold(order.BuyerEmail, body.Email) {
		c.JSON(http.StatusForbidden, gin.H{"error": "email tidak sesuai"})
		return
	}

	token := makeViewToken(no, order.BuyerEmail)
	c.JSON(http.StatusOK, gin.H{"token": token})
}

// makeViewToken membuat HMAC token yang bisa diverifikasi server-side
func makeViewToken(invoiceNo, email string) string {
	mac := hmac.New(sha256.New, []byte(config.App.JWTSecret))
	mac.Write([]byte(invoiceNo + ":" + strings.ToLower(email)))
	return hex.EncodeToString(mac.Sum(nil))[:32]
}

// verifyViewToken memvalidasi token
func verifyViewToken(invoiceNo, email, token string) bool {
	expected := makeViewToken(invoiceNo, strings.ToLower(email))
	return hmac.Equal([]byte(expected), []byte(token))
}
