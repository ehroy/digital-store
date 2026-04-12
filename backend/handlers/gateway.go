// gateway.go — Multi-gateway handler: DompetX & SayaBayar
// Tambah gateway baru: implementasi interface GatewayResult, daftarkan di resolveGateway()
package handlers

import (
	"digistore/database"
	"digistore/email"
	"digistore/gateway"
	"digistore/models"
	"digistore/scripts"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ── GatewayResult — hasil membuat charge/invoice di gateway ──────────────────

type GatewayResult struct {
	ChargeID  string     // ID di sisi gateway (untuk cek status)
	PayURL    string     // URL halaman bayar untuk pembeli
	PayCode   string     // Nomor VA / kode bayar (opsional)
	ExpiredAt *time.Time // Waktu kadaluarsa
	Provider  string     // "dompetx" | "sayabayar"
}

// ── resolveGateway — pilih gateway yang aktif ─────────────────────────────────
// Prioritas: SayaBayar → DompetX → nil (manual).
// Ubah urutan di sini untuk mengubah prioritas default.

func resolveGateway(cfg *models.PaymentConfig) string {
	if cfg.SayaBayarEnabled && cfg.SayaBayarAPIKey != "" {
		return "sayabayar"
	}
	if cfg.DompetXEnabled && cfg.DompetXAPIKey != "" {
		return "dompetx"
	}
	return "" // tidak ada gateway aktif — mode manual
}

// ── createGatewayCharge — buat charge di gateway yang aktif ──────────────────

func createGatewayCharge(order *models.Order, appBaseURL string) GatewayResult {
	var cfg models.PaymentConfig
	database.DB.First(&cfg, 1)

	provider := resolveGateway(&cfg)
	log.Printf("[GATEWAY] Provider: %q untuk order %s", provider, order.InvoiceNo)

	switch provider {
	case "sayabayar":
		return createSayaBayarCharge(order, &cfg, appBaseURL)
	case "dompetx":
		return createDompetXCharge(order, &cfg, appBaseURL)
	}
	return GatewayResult{} // manual
}

// ── SayaBayar ─────────────────────────────────────────────────────────────────

func createSayaBayarCharge(order *models.Order, cfg *models.PaymentConfig, appBaseURL string) GatewayResult {
	sb := gateway.NewSayaBayar(cfg.SayaBayarAPIKey)

	expHours := cfg.PaymentExpireHours
	if expHours <= 0 { expHours = 24 }
	expMinutes := expHours * 60

	channel := cfg.SayaBayarChannel
	if channel == "" { channel = "platform" }

	req := gateway.SBCreateRequest{
		CustomerName:      order.BuyerName,
		CustomerEmail:     order.BuyerEmail,
		Amount:            order.Total,
		Description:       "Pembayaran " + order.ProductName + " — " + order.InvoiceNo,
		ChannelPreference: channel,
		ExpiredMinutes:    expMinutes,
	}

	resp, err := sb.CreateInvoice(req)
	if err != nil {
		log.Printf("[SAYABAYAR] Gagal buat invoice %s: %v", order.InvoiceNo, err)
		return GatewayResult{}
	}

	exp := time.Now().Add(time.Duration(expHours) * time.Hour)
	log.Printf("[SAYABAYAR] Invoice dibuat: %s → %s", order.InvoiceNo, resp.Data.ID)
	return GatewayResult{
		ChargeID:  resp.Data.ID,
		PayURL:    resp.Data.PaymentURL,
		ExpiredAt: &exp,
		Provider:  "sayabayar",
	}
}

// ── DompetX ───────────────────────────────────────────────────────────────────

func createDompetXCharge(order *models.Order, cfg *models.PaymentConfig, appBaseURL string) GatewayResult {
	gw := gateway.NewDompetX(cfg.DompetXAPIKey, cfg.DompetXSecretKey, cfg.DompetXSandbox)

	expHours := cfg.PaymentExpireHours
	if expHours <= 0 { expHours = 24 }
	exp := time.Now().Add(time.Duration(expHours) * time.Hour)

	req := gateway.ChargeRequest{
		OrderID:       order.InvoiceNo,
		Amount:        order.Total,
		Currency:      "IDR",
		PaymentMethod: gateway.MapPayMethod(order.PayMethod),
		CustomerName:  order.BuyerName,
		CustomerEmail: order.BuyerEmail,
		Description:   "Pembayaran " + order.ProductName,
		ExpiredAt:     exp.UTC().Format(time.RFC3339),
		CallbackURL:   appBaseURL + "/api/webhook/dompetx",
		ReturnURL:     appBaseURL + "/payment/" + order.InvoiceNo,
		Metadata:      gateway.Metadata{InvoiceNo: order.InvoiceNo, Source: "digistore"},
	}

	resp, err := gw.CreateCharge(req)
	if err != nil {
		log.Printf("[DOMPETX] Gagal buat charge %s: %v", order.InvoiceNo, err)
		return GatewayResult{}
	}

	log.Printf("[DOMPETX] Charge dibuat: %s → %s", order.InvoiceNo, resp.Data.ChargeID)
	return GatewayResult{
		ChargeID:  resp.Data.ChargeID,
		PayURL:    resp.Data.PaymentURL,
		PayCode:   resp.Data.PaymentCode,
		ExpiredAt: &exp,
		Provider:  "dompetx",
	}
}

// ── Webhook DompetX ───────────────────────────────────────────────────────────

func WebhookDompetX(c *gin.Context) {
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "gagal baca body"}); return }

	var cfg models.PaymentConfig
	database.DB.First(&cfg, 1)
	if cfg.DompetXEnabled {
		sig := c.GetHeader("X-Signature")
		if sig == "" { sig = c.GetHeader("X-DompetX-Signature") }
		gw := gateway.NewDompetX(cfg.DompetXAPIKey, cfg.DompetXSecretKey, cfg.DompetXSandbox)
		if !gw.VerifySignature(rawBody, sig) {
			log.Printf("[DOMPETX WEBHOOK] Signature tidak valid dari %s", c.ClientIP())
			c.JSON(http.StatusUnauthorized, gin.H{"error": "signature tidak valid"})
			return
		}
	}

	var payload gateway.WebhookPayload
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload tidak valid"})
		return
	}
	log.Printf("[DOMPETX WEBHOOK] Event: %s | Order: %s | Status: %s", payload.Event, payload.OrderID, payload.Status)

	invoiceNo := payload.OrderID
	if payload.Metadata.InvoiceNo != "" { invoiceNo = payload.Metadata.InvoiceNo }
	handleWebhookEvent(invoiceNo, payload.Event, payload.Status, c)
}

// ── Webhook SayaBayar ─────────────────────────────────────────────────────────

func WebhookSayaBayar(c *gin.Context) {
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "gagal baca body"}); return }

	var payload gateway.SBWebhookPayload
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload tidak valid"})
		return
	}
	log.Printf("[SAYABAYAR WEBHOOK] Event: %s | Invoice: %s | Status: %s",
		payload.Event, payload.Invoice.ID, payload.Invoice.Status)

	// Cari order via gateway_charge_id (= ID invoice SayaBayar)
	var order models.Order
	if err := database.DB.Where("gateway_charge_id = ?", payload.Invoice.ID).First(&order).Error; err != nil {
		// Fallback: cari lewat deskripsi (invoice_number bisa jadi diisi invoice_no kita)
		log.Printf("[SAYABAYAR WEBHOOK] Charge ID %s tidak ditemukan, coba InvoiceNumber", payload.Invoice.ID)
		c.JSON(http.StatusOK, gin.H{"message": "diabaikan"})
		return
	}

	handleWebhookEvent(order.InvoiceNo, payload.Event, payload.Invoice.Status, c)
}

// handleWebhookEvent — logik bersama setelah webhook diterima dari gateway manapun
func handleWebhookEvent(invoiceNo, event, status string, c *gin.Context) {
	var order models.Order
	if err := database.DB.Where("invoice_no = ?", invoiceNo).First(&order).Error; err != nil {
		log.Printf("[WEBHOOK] Order %s tidak ditemukan", invoiceNo)
		c.JSON(http.StatusOK, gin.H{"message": "order tidak ditemukan, diabaikan"})
		return
	}
	if order.Status == "paid" || order.Status == "cancelled" {
		c.JSON(http.StatusOK, gin.H{"message": "sudah final"})
		return
	}

	isPaid := event == "charge.paid" || event == "charge.success" ||
		event == "invoice.paid" || status == "paid"
	isExpired := event == "charge.expired" || event == "invoice.expired" || status == "expired"
	isFailed := event == "charge.failed" || status == "failed"

	switch {
	case isPaid:
		deliverOrder(&order)
	case isExpired:
		database.DB.Model(&order).Update("status", "expired")
		restoreStock(&order)
		log.Printf("[WEBHOOK] Order %s expired", order.InvoiceNo)
	case isFailed:
		database.DB.Model(&order).Update("status", "failed")
		restoreStock(&order)
		log.Printf("[WEBHOOK] Order %s failed", order.InvoiceNo)
	default:
		log.Printf("[WEBHOOK] Event tidak dikenali: %s", event)
	}
	c.JSON(http.StatusOK, gin.H{"received": true})
}

// ── deliverOrder ──────────────────────────────────────────────────────────────

func deliverOrder(order *models.Order) {
	log.Printf("[DELIVER] Order %s tipe %s", order.InvoiceNo, order.ProductType)

	if order.ProductType == "stock" {
		if order.DeliveredItems != "" {
			database.DB.Model(order).Update("status", "paid")
			return
		}
		claimed, err := ClaimStockItems(order.ProductID, order.Qty, order.InvoiceNo)
		if err != nil {
			log.Printf("[DELIVER] Gagal klaim stok %s: %v", order.InvoiceNo, err)
			database.DB.Model(order).Update("status", "failed")
			return
		}
		itemData := make([]string, len(claimed))
		for i, it := range claimed { itemData[i] = it.Data }
		deliveredJSON, _ := json.Marshal(itemData)
		database.DB.Model(order).Updates(map[string]interface{}{
			"delivered_items": string(deliveredJSON),
			"status":          "paid",
		})
		order.DeliveredItems = string(deliveredJSON)
		order.Status = "paid"
		go email.SendInvoiceWithItems(order, itemData)
		log.Printf("[DELIVER] %d item terkirim ke order %s", len(itemData), order.InvoiceNo)

	} else {
		var product models.Product
		database.DB.First(&product, order.ProductID)
		if product.Script != "" {
			go func() {
				result := scripts.Execute(product.Script, order, email.Send)
				actionsJSON, _ := json.Marshal(result.Actions)
				database.DB.Create(&models.ScriptLog{
					OrderID:   order.ID,
					InvoiceNo: order.InvoiceNo,
					Product:   order.ProductName,
					Script:    product.Script,
					Status:    result.Status,
					Output:    string(actionsJSON),
				})
				newStatus := "script_executed"
				if result.Status == "failed" { newStatus = "failed" }
				database.DB.Model(order).Update("status", newStatus)
			}()
		} else {
			database.DB.Model(order).Update("status", "paid")
		}
		go email.SendInvoiceService(order)
	}
}

// ── CheckAndSyncGatewayStatus — polling manual saat pembeli buka halaman payment

func CheckAndSyncGatewayStatus(order *models.Order) {
	if order.GatewayChargeID == "" { return }
	var cfg models.PaymentConfig
	database.DB.First(&cfg, 1)

	var status string
	provider := resolveGateway(&cfg)

	switch provider {
	case "sayabayar":
		sb := gateway.NewSayaBayar(cfg.SayaBayarAPIKey)
		resp, err := sb.GetInvoice(order.GatewayChargeID)
		if err != nil || !resp.Success { return }
		status = resp.Data.Status

	case "dompetx":
		gw := gateway.NewDompetX(cfg.DompetXAPIKey, cfg.DompetXSecretKey, cfg.DompetXSandbox)
		resp, err := gw.GetChargeStatus(order.GatewayChargeID)
		if err != nil || !resp.Success { return }
		status = resp.Data.Status
	default:
		return
	}

	switch status {
	case "paid", "success":
		if order.Status == "pending" { deliverOrder(order) }
	case "expired":
		if order.Status == "pending" {
			database.DB.Model(order).Update("status", "expired")
			restoreStock(order)
		}
	case "failed":
		if order.Status == "pending" {
			database.DB.Model(order).Update("status", "failed")
			restoreStock(order)
		}
	}
}

// ── Expiry Job ────────────────────────────────────────────────────────────────

func StartExpiryJob() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		log.Println("⏰ Expiry job dimulai")
		for range ticker.C { cancelExpiredOrders() }
	}()
}

func cancelExpiredOrders() {
	var expired []models.Order
	database.DB.Where("status = ? AND expired_at IS NOT NULL AND expired_at < ?", "pending", time.Now()).Find(&expired)
	for _, o := range expired {
		log.Printf("[EXPIRY] Cancel order %s", o.InvoiceNo)
		database.DB.Model(&o).Update("status", "cancelled")
		restoreStock(&o)
	}
	if len(expired) > 0 { log.Printf("[EXPIRY] %d order dicancel", len(expired)) }
}

func restoreStock(order *models.Order) {
	if order.ProductType == "stock" && order.DeliveredItems == "" {
		database.DB.Model(&models.ProductStock{}).
			Where("invoice_no = ? AND sold = ?", order.InvoiceNo, true).
			Updates(map[string]interface{}{"sold": false, "invoice_no": "", "sold_at": nil})
	}
}

// ── Utility ───────────────────────────────────────────────────────────────────

func FormatIDR(n int64) string {
	s := fmt.Sprintf("%d", n)
	r := []rune(s)
	var out []rune
	for i, ch := range r {
		if i > 0 && (len(r)-i)%3 == 0 { out = append(out, '.') }
		out = append(out, ch)
	}
	return "Rp " + strings.TrimSpace(string(out))
}
