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

// ── GatewayResult ─────────────────────────────────────────────────────────────

type GatewayResult struct {
	ChargeID      string     // ID invoice/charge di gateway (untuk GetStatus)
	GatewayInvNo  string     // Nomor invoice dari gateway (mis: INV-20240327-0042 dari SayaBayar)
	PayURL        string
	PayCode       string
	ExpiredAt     *time.Time
	Provider      string     // "sayabayar" | "dompetx"
}

// ── ValidatePayMethod — cek apakah method boleh dipakai ───────────────────────

// ValidatePayMethod memastikan pay_method yang dikirim user sesuai dengan yang aktif di config.
// Mencegah user bypass frontend dan kirim metode yang tidak tersedia.
func ValidatePayMethod(method string, cfg *models.PaymentConfig) error {
	// Jika gateway aktif, satu-satunya method yang valid adalah "gateway"
	if cfg.SayaBayarEnabled && cfg.SayaBayarAPIKey != "" {
		if method != "gateway" {
			return fmt.Errorf("gateway SayaBayar aktif — gunakan metode 'gateway'")
		}
		return nil
	}
	if cfg.DompetXEnabled && cfg.DompetXAPIKey != "" {
		if method != "gateway" {
			return fmt.Errorf("gateway DompetX aktif — gunakan metode 'gateway'")
		}
		return nil
	}

	// Mode manual: validasi setiap metode yang dikonfigurasi
	switch method {
	case "bank":
		if cfg.BankNo == "" {
			return fmt.Errorf("transfer bank tidak dikonfigurasi")
		}
	case "dana":
		if cfg.Dana == "" {
			return fmt.Errorf("DANA tidak dikonfigurasi")
		}
	case "gopay":
		if cfg.Gopay == "" {
			return fmt.Errorf("GoPay tidak dikonfigurasi")
		}
	case "ovo":
		if cfg.Ovo == "" {
			return fmt.Errorf("OVO tidak dikonfigurasi")
		}
	case "qris":
		if !cfg.QRIS {
			return fmt.Errorf("QRIS tidak diaktifkan")
		}
	case "crypto":
		if !cfg.Crypto || cfg.CryptoAddr == "" {
			return fmt.Errorf("cryptocurrency tidak diaktifkan")
		}
	default:
		return fmt.Errorf("metode pembayaran '%s' tidak dikenal", method)
	}
	return nil
}

// ── resolveGateway ────────────────────────────────────────────────────────────

func resolveGateway(cfg *models.PaymentConfig) string {
	if cfg.SayaBayarEnabled && cfg.SayaBayarAPIKey != "" {
		return "sayabayar"
	}
	if cfg.DompetXEnabled && cfg.DompetXAPIKey != "" {
		return "dompetx"
	}
	return ""
}

// ── createGatewayCharge ───────────────────────────────────────────────────────

func createGatewayCharge(order *models.Order, appBaseURL string) GatewayResult {
	var cfg models.PaymentConfig
	database.DB.First(&cfg, 1)

	switch resolveGateway(&cfg) {
	case "sayabayar":
		return createSayaBayarCharge(order, &cfg, appBaseURL)
	case "dompetx":
		return createDompetXCharge(order, &cfg, appBaseURL)
	}
	return GatewayResult{}
}

// ── SayaBayar charge ──────────────────────────────────────────────────────────

func createSayaBayarCharge(order *models.Order, cfg *models.PaymentConfig, appBaseURL string) GatewayResult {
	sb := gateway.NewSayaBayar(cfg.SayaBayarAPIKey)

	expHours := cfg.PaymentExpireHours
	if expHours <= 0 { expHours = 24 }

	channel := cfg.SayaBayarChannel
	if channel == "" { channel = "platform" }

	// Sertakan invoice_no internal di description agar bisa dilacak
	resp, err := sb.CreateInvoice(gateway.SBCreateRequest{
		CustomerName:      order.BuyerName,
		CustomerEmail:     order.BuyerEmail,
		Amount:            order.Total,
		Description:       fmt.Sprintf("[%s] %s", order.InvoiceNo, order.ProductName),
		ChannelPreference: channel,
		Redirect_Url:      appBaseURL + "/payment/" + order.InvoiceNo,
		ExpiredMinutes:    expHours * 60,
	})
	if err != nil {
		log.Printf("[SAYABAYAR] Gagal buat invoice %s: %v", order.InvoiceNo, err)
		return GatewayResult{}
	}

	exp := time.Now().Add(time.Duration(expHours) * time.Hour)
	log.Printf("[SAYABAYAR] Invoice dibuat: internal=%s gateway_id=%s gateway_inv=%s",
		order.InvoiceNo, resp.Data.ID, resp.Data.InvoiceNumber)

	return GatewayResult{
		ChargeID:     resp.Data.ID,            // clx9abc123
		GatewayInvNo: resp.Data.InvoiceNumber, // INV-20240327-0042 dari SayaBayar
		PayURL:       resp.Data.PaymentURL,
		ExpiredAt:    &exp,
		Provider:     "sayabayar",
	}
}

// ── DompetX charge ────────────────────────────────────────────────────────────

func createDompetXCharge(order *models.Order, cfg *models.PaymentConfig, appBaseURL string) GatewayResult {
	gw := gateway.NewDompetX(cfg.DompetXAPIKey, cfg.DompetXSecretKey, cfg.DompetXSandbox)

	expHours := cfg.PaymentExpireHours
	if expHours <= 0 { expHours = 24 }
	exp := time.Now().Add(time.Duration(expHours) * time.Hour)

	resp, err := gw.CreateCharge(gateway.ChargeRequest{
		OrderID:       order.InvoiceNo, // pakai invoice kita sebagai order_id
		Amount:        order.Total,
		Currency:      "IDR",
		PaymentMethod: gateway.MapPayMethod(order.PayMethod),
		CustomerName:  order.BuyerName,
		CustomerEmail: order.BuyerEmail,
		Description:   fmt.Sprintf("[%s] %s", order.InvoiceNo, order.ProductName),
		ExpiredAt:     exp.UTC().Format(time.RFC3339),
		CallbackURL:   appBaseURL + "/api/webhook/dompetx",
		ReturnURL:     appBaseURL + "/payment/" + order.InvoiceNo,
		Metadata:      gateway.Metadata{InvoiceNo: order.InvoiceNo, Source: "digistore"},
	})
	if err != nil {
		log.Printf("[DOMPETX] Gagal buat charge %s: %v", order.InvoiceNo, err)
		return GatewayResult{}
	}

	log.Printf("[DOMPETX] Charge dibuat: internal=%s gateway_id=%s", order.InvoiceNo, resp.Data.ChargeID)
	return GatewayResult{
		ChargeID:     resp.Data.ChargeID,
		GatewayInvNo: resp.Data.ChargeID, // DompetX tidak punya invoice_number terpisah
		PayURL:       resp.Data.PaymentURL,
		PayCode:      resp.Data.PaymentCode,
		ExpiredAt:    &exp,
		Provider:     "dompetx",
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
			log.Printf("[DOMPETX WEBHOOK] Invalid signature dari %s", c.ClientIP())
			c.JSON(http.StatusUnauthorized, gin.H{"error": "signature tidak valid"})
			return
		}
	}

	var payload gateway.WebhookPayload
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload tidak valid"})
		return
	}

	// DompetX mengirim order_id = invoice_no kita
	invoiceNo := payload.OrderID
	if payload.Metadata.InvoiceNo != "" { invoiceNo = payload.Metadata.InvoiceNo }
	log.Printf("[DOMPETX WEBHOOK] event=%s order=%s status=%s", payload.Event, invoiceNo, payload.Status)
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
	log.Printf("[SAYABAYAR WEBHOOK] event=%s id=%s inv=%s status=%s",
		payload.Event, payload.Invoice.ID, payload.Invoice.InvoiceNumber, payload.Invoice.Status)

	// Cari order via gateway_charge_id (= ID dari SayaBayar)
	var order models.Order
	if err := database.DB.Where("gateway_charge_id = ?", payload.Invoice.ID).First(&order).Error; err != nil {
		// Fallback: cari via gateway_invoice_no
		if payload.Invoice.InvoiceNumber != "" {
			database.DB.Where("gateway_invoice_no = ?", payload.Invoice.InvoiceNumber).First(&order)
		}
	}
	if order.ID == 0 {
		log.Printf("[SAYABAYAR WEBHOOK] Order tidak ditemukan untuk charge_id=%s", payload.Invoice.ID)
		c.JSON(http.StatusOK, gin.H{"message": "diabaikan"})
		return
	}

	handleWebhookEvent(order.InvoiceNo, payload.Event, payload.Invoice.Status, c)
}

// ── handleWebhookEvent — logik bersama ───────────────────────────────────────

func handleWebhookEvent(invoiceNo, event, status string, c *gin.Context) {
	var order models.Order
	if err := database.DB.Where("invoice_no = ?", invoiceNo).First(&order).Error; err != nil {
		log.Printf("[WEBHOOK] Order %s tidak ditemukan", invoiceNo)
		c.JSON(http.StatusOK, gin.H{"message": "diabaikan"})
		return
	}
	if order.Status == "paid" || order.Status == "cancelled" {
		c.JSON(http.StatusOK, gin.H{"message": "sudah final"})
		return
	}

	isPaid    := event == "charge.paid" || event == "charge.success" || event == "invoice.paid" || status == "paid"
	isExpired := event == "charge.expired" || event == "invoice.expired" || status == "expired"
	isFailed  := event == "charge.failed" || status == "failed"

	switch {
	case isPaid:
		deliverOrder(&order)
	case isExpired:
		database.DB.Model(&order).Update("status", "expired")
		restoreStock(&order)
		log.Printf("[WEBHOOK] Order %s expired, stok dikembalikan", order.InvoiceNo)
	case isFailed:
		database.DB.Model(&order).Update("status", "failed")
		restoreStock(&order)
	}
	c.JSON(http.StatusOK, gin.H{"received": true})
}

// ── deliverOrder ──────────────────────────────────────────────────────────────

func deliverOrder(order *models.Order) {
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
			"delivered_items": string(deliveredJSON), "status": "paid",
		})
		order.DeliveredItems = string(deliveredJSON)
		order.Status = "paid"
		go email.SendInvoiceWithItems(order, itemData)
	} else {
		var product models.Product
		database.DB.First(&product, order.ProductID)
		if product.Script != "" {
			go func() {
				result := scripts.Execute(product.Script, order, email.Send)
				actJSON, _ := json.Marshal(result.Actions)
				database.DB.Create(&models.ScriptLog{
					OrderID: order.ID, InvoiceNo: order.InvoiceNo, Product: order.ProductName,
					Script: product.Script, Status: result.Status, Output: string(actJSON),
				})
				status := "script_executed"
				if result.Status == "failed" { status = "failed" }
				database.DB.Model(order).Update("status", status)
			}()
		} else {
			database.DB.Model(order).Update("status", "paid")
		}
		go email.SendInvoiceService(order)
	}
}

// ── CheckAndSyncGatewayStatus — polling manual saat pembeli buka halaman ──────

func CheckAndSyncGatewayStatus(order *models.Order) {
	if order.GatewayChargeID == "" { return }
	var cfg models.PaymentConfig
	database.DB.First(&cfg, 1)

	var status string
	switch resolveGateway(&cfg) {
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

	if order.Status != "pending" { return }
	switch status {
	case "paid", "success": deliverOrder(order)
	case "expired":
		database.DB.Model(order).Update("status", "expired")
		restoreStock(order)
	case "failed":
		database.DB.Model(order).Update("status", "failed")
		restoreStock(order)
	}
}

// ── Expiry Job ────────────────────────────────────────────────────────────────

func StartExpiryJob() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		log.Println("⏰ Expiry job dimulai — cek setiap 5 menit")
		for range ticker.C { cancelExpiredOrders() }
	}()
}

func cancelExpiredOrders() {
	var expired []models.Order
	database.DB.Where("status = ? AND expired_at IS NOT NULL AND expired_at < ?", "pending", time.Now()).Find(&expired)
	for _, o := range expired {
		log.Printf("[EXPIRY] Cancel order %s (expired: %v)", o.InvoiceNo, o.ExpiredAt)
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
