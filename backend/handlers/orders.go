package handlers

import (
	"digistore/config"
	"digistore/database"
	"digistore/email"
	"digistore/models"
	"encoding/json"
	"fmt"
	"net/http"
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

	// Coba buat charge di gateway yang aktif
	gwResult := createGatewayCharge(&order, config.App.FrontendURL)
	if gwResult.ChargeID != "" {
		database.DB.Model(&order).Updates(map[string]interface{}{
			"gateway_charge_id": gwResult.ChargeID,
			"gateway_pay_url":   gwResult.PayURL,
			"gateway_pay_code":  gwResult.PayCode,
			"expired_at":        gwResult.ExpiredAt,
		})
		order.GatewayChargeID = gwResult.ChargeID
		order.GatewayPayURL = gwResult.PayURL
		order.GatewayPayCode = gwResult.PayCode
		order.ExpiredAt = gwResult.ExpiredAt
		// Kirim email "menunggu pembayaran" dengan tombol bayar
		go email.SendPendingInvoice(&order, gwResult.PayURL, gwResult.PayCode)
	} else {
		// Tidak ada gateway aktif — mode manual
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

func GetInvoicePublic(c *gin.Context) {
	var order models.Order
	if err := database.DB.Where("invoice_no = ?", c.Param("no")).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "invoice tidak ditemukan"}); return
	}
	if order.Status == "pending" && order.GatewayChargeID != "" {
		CheckAndSyncGatewayStatus(&order)
		database.DB.Where("invoice_no = ?", c.Param("no")).First(&order)
	}
	var items []string
	if order.Status == "paid" && order.DeliveredItems != "" {
		json.Unmarshal([]byte(order.DeliveredItems), &items)
	}
	c.JSON(http.StatusOK, gin.H{
		"invoice_no":       order.InvoiceNo,
		"product_name":     order.ProductName,
		"product_type":     order.ProductType,
		"buyer_name":       order.BuyerName,
		"qty":              order.Qty,
		"total":            order.Total,
		"pay_method":       order.PayMethod,
		"status":           order.Status,
		"delivered_items":  items,
		"gateway_pay_url":  order.GatewayPayURL,
		"gateway_pay_code": order.GatewayPayCode,
		"expired_at":       order.ExpiredAt,
		"created_at":       order.CreatedAt,
		"updated_at":       order.UpdatedAt,
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

func generateInvoice() string {
	t := time.Now()
	return fmt.Sprintf("INV-%d%02d%02d-%06d", t.Year(), t.Month(), t.Day(), t.UnixMilli()%1000000)
}

// POST /api/admin/orders/:id/deliver — admin kirim produk secara manual
// Untuk produk stok: admin pilih item mana yang dikirim
// Untuk produk script: admin bisa re-trigger script
func ManualDeliver(c *gin.Context) {
	var o models.Order
	if err := database.DB.First(&o, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order tidak ditemukan"})
		return
	}

	var body struct {
		// Untuk stok: admin bisa override item yang dikirim (opsional)
		// Jika kosong, sistem klaim stok tersedia otomatis
		Items []string `json:"items"`
		// Untuk script: apakah re-run script?
		RunScript bool `json:"run_script"`
		// Catatan admin
		Note string `json:"note"`
	}
	c.ShouldBindJSON(&body)

	if o.ProductType == "stock" {
		if len(body.Items) > 0 {
			// Admin override: kirim item yang dipilih manual
			itemsJSON, _ := json.Marshal(body.Items)
			database.DB.Model(&o).Updates(map[string]interface{}{
				"delivered_items": string(itemsJSON),
				"status":          "paid",
				"notes":           "Manual delivery by admin: " + body.Note,
			})
			o.DeliveredItems = string(itemsJSON)
			o.Status = "paid"
			go email.SendInvoiceWithItems(&o, body.Items)
			c.JSON(http.StatusOK, gin.H{"message": "item berhasil dikirim manual", "items": body.Items})
		} else {
			// Klaim stok otomatis lalu deliver
			deliverOrder(&o)
			database.DB.Where("id = ?", o.ID).First(&o)
			c.JSON(http.StatusOK, gin.H{"message": "item dikirim otomatis dari stok", "status": o.Status})
		}
	} else {
		// Script product: deliver (re-run script)
		deliverOrder(&o)
		c.JSON(http.StatusOK, gin.H{"message": "script dieksekusi"})
	}
}
