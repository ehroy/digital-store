package handlers

import (
	"digistore/config"
	"digistore/database"
	"digistore/email"
	"digistore/models"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func CreateOrder(c *gin.Context) {
	var req struct {
		ProductID  uint   `json:"product_id"  binding:"required"`
		BuyerName  string `json:"buyer_name"  binding:"required"`
		BuyerEmail string `json:"buyer_email" binding:"required"`
		Qty        int    `json:"qty"         binding:"required,min=1"`
		PayMethod  string `json:"pay_method"  binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}

	// ── Validasi metode pembayaran di backend ────────────────────────────────
	var cfg models.PaymentConfig
	database.DB.First(&cfg, 1)
	if err := ValidatePayMethod(req.PayMethod, &cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "metode pembayaran tidak valid: " + err.Error()})
		return
	}

	var product models.Product
	if err := database.DB.First(&product, req.ProductID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "produk tidak ditemukan"}); return
	}
	if !product.Active {
		c.JSON(http.StatusBadRequest, gin.H{"error": "produk tidak tersedia"}); return
	}
	if product.Type == "stock" {
		var avail int64
		database.DB.Model(&models.ProductStock{}).
			Where("product_id = ? AND sold = ?", product.ID, false).Count(&avail)
		if int(avail) < req.Qty {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("stok tidak mencukupi: tersedia %d, diminta %d", avail, req.Qty),
			}); return
		}
	}

	order := models.Order{
		InvoiceNo:   generateInvoice(),
		ProductID:   product.ID,
		ProductName: product.Name,
		ProductType: product.Type,
		Qty:         req.Qty,
		Price:       product.Price,
		Total:       product.Price * int64(req.Qty),
		BuyerName:   req.BuyerName,
		BuyerEmail:  req.BuyerEmail,
		PayMethod:   req.PayMethod,
		Status:      "pending",
	}
	if err := database.DB.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal membuat order"}); return
	}

	gwResult := createGatewayCharge(&order, config.App.FrontendURL)
	if gwResult.ChargeID != "" {
		database.DB.Model(&order).Updates(map[string]interface{}{
			"gateway_charge_id":  gwResult.ChargeID,
			"gateway_invoice_no": gwResult.GatewayInvNo,
			"gateway_provider":   gwResult.Provider,
			"gateway_pay_url":    gwResult.PayURL,
			"gateway_pay_code":   gwResult.PayCode,
			"expired_at":         gwResult.ExpiredAt,
		})
		order.GatewayChargeID  = gwResult.ChargeID
		order.GatewayInvoiceNo = gwResult.GatewayInvNo
		order.GatewayProvider  = gwResult.Provider
		order.GatewayPayURL    = gwResult.PayURL
		order.GatewayPayCode   = gwResult.PayCode
		order.ExpiredAt        = gwResult.ExpiredAt
		go email.SendPendingInvoice(&order, gwResult.PayURL, gwResult.PayCode)
	} else {
		// Produk tipe provider: cek stok cache, lalu order ke KoalaStore
		if product.Type == "provider" {
			// Cek stok dari cache ProviderProduct
			var pp models.ProviderProduct
			if dbErr := database.DB.Where("code = ?", product.ProviderCode).First(&pp).Error; dbErr == nil {
				if pp.Stock == "out_of_stock" {
					database.DB.Delete(&order)
					c.JSON(http.StatusBadRequest, gin.H{"error": "stok habis di provider, coba beberapa saat lagi"})
					return
				}
			}
			delivered, err := OrderFromKoalaStore(&order, &product)
			if err != nil {
				database.DB.Delete(&order)
				c.JSON(http.StatusBadGateway, gin.H{"error": "gagal order ke provider: " + err.Error()})
				return
			}
			deliveredJSON, _ := json.Marshal(delivered)
			database.DB.Model(&order).Updates(map[string]interface{}{
				"delivered_items": string(deliveredJSON), "status": "paid",
			})
			order.DeliveredItems = string(deliveredJSON)
			order.Status = "paid"
			go email.SendInvoiceWithItems(&order, delivered)
		} else

		// Mode manual
		if product.Type == "stock" {
			claimed, err := ClaimStockItems(product.ID, req.Qty, order.InvoiceNo)
			if err != nil {
				database.DB.Delete(&order)
				c.JSON(http.StatusConflict, gin.H{"error": err.Error()}); return
			}
			itemData := make([]string, len(claimed))
			for i, it := range claimed { itemData[i] = it.Data }
			deliveredJSON, _ := json.Marshal(itemData)
			database.DB.Model(&order).Updates(map[string]interface{}{
				"delivered_items": string(deliveredJSON), "status": "paid",
			})
			order.DeliveredItems = string(deliveredJSON)
			order.Status = "paid"
			go email.SendInvoiceWithItems(&order, itemData)
		} else {
			go func(o models.Order) { deliverOrder(&o) }(order)
			go email.SendInvoiceService(&order)
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"order":          order,
		"invoice_no":     order.InvoiceNo,
		"pay_url":        gwResult.PayURL,
		"pay_code":       gwResult.PayCode,
		"expired_at":     gwResult.ExpiredAt,
		"gateway_active": gwResult.ChargeID != "",
		"gateway":        gwResult.Provider,
		"redirect":       "/payment/" + order.InvoiceNo,
	})
}

// GET /api/invoice/:no — cek status invoice
// Verifikasi: wajib salah satu dari:
//   ?email=xxx  → verifikasi email langsung
//   ?token=xxx  → token HMAC yang digenerate setelah checkout (tanpa email ulang)
func GetInvoicePublic(c *gin.Context) {
	no := c.Param("no")
	emailParam := strings.TrimSpace(c.Query("email"))
	tokenParam := strings.TrimSpace(c.Query("token"))

	if emailParam == "" && tokenParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         "verifikasi wajib: sertakan ?email=xxx atau ?token=xxx",
			"require_email": true,
		})
		return
	}

	var order models.Order
	// Cari berdasarkan invoice_no internal ATAU gateway_invoice_no (mis: nomor invoice SayaBayar)
	err := database.DB.Where("invoice_no = ? OR gateway_invoice_no = ?", no, no).First(&order).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "invoice tidak ditemukan"})
		return
	}

	// Verifikasi: token ATAU email
	if tokenParam != "" {
		if !verifyViewToken(no, order.BuyerEmail, tokenParam) {
			c.JSON(http.StatusForbidden, gin.H{"error": "token tidak valid"})
			return
		}
	} else {
		if !strings.EqualFold(order.BuyerEmail, emailParam) {
			c.JSON(http.StatusForbidden, gin.H{"error": "email tidak sesuai dengan data pesanan"})
			return
		}
	}

	// Sync status dari gateway jika masih pending
	if order.Status == "pending" && order.GatewayChargeID != "" {
		CheckAndSyncGatewayStatus(&order)
		database.DB.Where("invoice_no = ? OR gateway_invoice_no = ?", no, no).First(&order)
	}

	// Cek apakah sudah expired tapi belum di-cancel
	if order.Status == "pending" && order.ExpiredAt != nil && order.ExpiredAt.Before(time.Now()) {
		database.DB.Model(&order).Update("status", "expired")
		restoreStock(&order)
		order.Status = "expired"
	}

	var items []string
	if order.Status == "paid" && order.DeliveredItems != "" {
		json.Unmarshal([]byte(order.DeliveredItems), &items)
	}

	c.JSON(http.StatusOK, gin.H{
		"invoice_no":        order.InvoiceNo,
		"gateway_invoice_no": order.GatewayInvoiceNo,
		"gateway_provider":  order.GatewayProvider,
		"product_name":      order.ProductName,
		"product_type":      order.ProductType,
		"buyer_name":        order.BuyerName,
		"qty":               order.Qty,
		"total":             order.Total,
		"pay_method":        order.PayMethod,
		"status":            order.Status,
		"delivered_items":   items,
		"gateway_pay_url":   order.GatewayPayURL,
		"gateway_pay_code":  order.GatewayPayCode,
		"expired_at":        order.ExpiredAt,
		"created_at":        order.CreatedAt,
		"updated_at":        order.UpdatedAt,
	})
}

func GetOrders(c *gin.Context) {
	var orders []models.Order
	query := database.DB.Order("created_at desc")
	if s := c.Query("status"); s != "" { query = query.Where("status = ?", s) }
	query.Find(&orders)
	c.JSON(http.StatusOK, orders)
}

func GetOrder(c *gin.Context) {
	var o models.Order
	if err := database.DB.First(&o, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order tidak ditemukan"}); return
	}
	var items []string
	if o.DeliveredItems != "" { json.Unmarshal([]byte(o.DeliveredItems), &items) }
	c.JSON(http.StatusOK, gin.H{"order": o, "delivered_items": items})
}

func UpdateOrderStatus(c *gin.Context) {
	var body struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	var o models.Order
	if err := database.DB.First(&o, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order tidak ditemukan"}); return
	}
	if body.Status == "paid" && o.Status == "pending" {
		deliverOrder(&o); return
	}
	database.DB.Model(&o).Update("status", body.Status)
	o.Status = body.Status
	c.JSON(http.StatusOK, o)
}

func ManualDeliver(c *gin.Context) {
	var o models.Order
	if err := database.DB.First(&o, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order tidak ditemukan"}); return
	}
	var body struct {
		Items     []string `json:"items"`
		RunScript bool     `json:"run_script"`
		Note      string   `json:"note"`
	}
	c.ShouldBindJSON(&body)

	if o.ProductType == "stock" && len(body.Items) > 0 {
		itemsJSON, _ := json.Marshal(body.Items)
		database.DB.Model(&o).Updates(map[string]interface{}{
			"delivered_items": string(itemsJSON),
			"status":          "paid",
			"notes":           "Manual delivery by admin: " + body.Note,
		})
		o.DeliveredItems = string(itemsJSON)
		o.Status = "paid"
		go email.SendInvoiceWithItems(&o, body.Items)
		c.JSON(http.StatusOK, gin.H{"message": "item berhasil dikirim manual", "status": "paid"})
	} else {
		deliverOrder(&o)
		database.DB.Where("id = ?", o.ID).First(&o)
		c.JSON(http.StatusOK, gin.H{"message": "berhasil", "status": o.Status})
	}
}

func generateInvoice() string {
	t := time.Now()
	return fmt.Sprintf("INV-%d%02d%02d-%06d", t.Year(), t.Month(), t.Day(), t.UnixMilli()%1000000)
}
