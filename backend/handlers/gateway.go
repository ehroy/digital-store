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
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ── GatewayResult ─────────────────────────────────────────────────────────────

type GatewayResult struct {
	ChargeID     string // ID invoice/charge di gateway (untuk GetStatus)
	GatewayInvNo string // Nomor invoice dari gateway (mis: INV-20240327-0042 dari SayaBayar)
	PayURL       string
	RedirectURL  string
	PayCode      string
	QRISString   string
	QRISImageURL string
	ExpiredAt    *time.Time
	Provider     string // "sayabayar" | "dompetx"
}

// ── ValidatePayMethod — cek apakah method boleh dipakai ───────────────────────

// ValidatePayMethod memastikan pay_method yang dikirim user sesuai dengan yang aktif di config.
// Mencegah user bypass frontend dan kirim metode yang tidak tersedia.
func ValidatePayMethod(method string, cfg *models.PaymentConfig) error {
	if method != "qris" {
		return fmt.Errorf("sementara hanya QRIS yang didukung")
	}
	if !(cfg.SayaBayarEnabled && cfg.SayaBayarAPIKey != "") && !(cfg.DompetXEnabled && cfg.DompetXAPIKey != "") {
		return fmt.Errorf("QRIS belum diaktifkan")
	}
	return nil
}

const gatewayExpireMinutes = 30

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

	channel := cfg.SayaBayarChannel
	if channel == "" {
		channel = "platform"
	}

	// Sertakan invoice_no internal di description agar bisa dilacak
	resp, err := sb.CreateInvoice(gateway.SBCreateRequest{
		CustomerName:      order.BuyerName,
		CustomerEmail:     order.BuyerEmail,
		Amount:            order.Total,
		Description:       fmt.Sprintf("[%s] %s", order.InvoiceNo, order.ProductName),
		ChannelPreference: channel,
		Redirect:          appBaseURL + "/payment/" + order.InvoiceNo,
		ExpiredMinutes:    gatewayExpireMinutes,
	})
	if err != nil {
		log.Printf("[SAYABAYAR] Gagal buat invoice %s: %v", order.InvoiceNo, err)
		return GatewayResult{}
	}

	exp := time.Now().Add(time.Duration(gatewayExpireMinutes) * time.Minute)
	paymentRef := gateway.ExtractPaymentRef(resp.Data.PaymentURL)
	if paymentRef == "" {
		paymentRef = resp.Data.ID
	}
	log.Printf("[SAYABAYAR] Invoice dibuat: internal=%s gateway_id=%s gateway_inv=%s",
		order.InvoiceNo, resp.Data.ID, resp.Data.InvoiceNumber)

	if cfg.SayaBayarAutoQRIS {
		if err := sb.SelectChannel(paymentRef, "qris"); err != nil {
			log.Printf("[SAYABAYAR] select-channel gagal invoice=%s ref=%s: %v", order.InvoiceNo, paymentRef, err)
		} else if cfg.SayaBayarAutoConfirm {
			if err := sb.ConfirmPayment(paymentRef); err != nil {
				log.Printf("[SAYABAYAR] confirm gagal invoice=%s ref=%s: %v", order.InvoiceNo, paymentRef, err)
			}
		}
	}

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
	exp := time.Now().Add(time.Duration(gatewayExpireMinutes) * time.Minute)

	resp, err := gw.CreateCharge(gateway.ChargeRequest{
		Method:          gateway.MapPayMethod(order.PayMethod),
		Amount:          order.Total,
		Currency:        "IDR",
		Reference:       order.InvoiceNo,
		ReturnURL:       appBaseURL + "/payment/" + order.InvoiceNo,
		CallbackURL:     appBaseURL + "/api/webhook/dompetx",
		SettlementSpeed: "standard",
		Metadata: map[string]any{
			"invoice_no": order.InvoiceNo,
			"source":     "digistore",
			"product":    order.ProductName,
		},
	})
	if err != nil {
		log.Printf("[DOMPETX] Gagal buat charge %s: %v", order.InvoiceNo, err)
		return GatewayResult{}
	}
	txID := firstNonEmpty(resp.TransactionID, resp.Reference)
	if txID != "" {
		if detail, err := gw.GetChargeStatus(txID); err == nil && detail != nil {
			resp = mergeDompetXCharge(resp, detail)
		}
	}

	log.Printf("[DOMPETX] Charge dibuat: internal=%s gateway_id=%s", order.InvoiceNo, resp.TransactionID)
	redirectURL := firstNonEmpty(resp.RedirectURL, resp.PaymentURL, appBaseURL+"/payment/"+order.InvoiceNo)
	return GatewayResult{
		ChargeID:     txID,
		GatewayInvNo: firstNonEmpty(resp.Reference, order.InvoiceNo),
		PayURL:       firstNonEmpty(resp.PaymentURL, redirectURL),
		RedirectURL:  redirectURL,
		PayCode:      resp.QRISString,
		QRISString:   resp.QRISString,
		QRISImageURL: resp.QRISImageURL,
		ExpiredAt:    &exp,
		Provider:     "dompetx",
	}
}

func mergeDompetXCharge(primary, secondary *gateway.ChargeResponse) *gateway.ChargeResponse {
	if primary == nil {
		return secondary
	}
	if secondary == nil {
		return primary
	}
	if primary.TransactionID == "" {
		primary.TransactionID = secondary.TransactionID
	}
	if primary.Reference == "" {
		primary.Reference = secondary.Reference
	}
	if primary.Status == "" {
		primary.Status = secondary.Status
	}
	if primary.Amount == 0 {
		primary.Amount = secondary.Amount
	}
	if primary.Currency == "" {
		primary.Currency = secondary.Currency
	}
	if primary.PaymentURL == "" {
		primary.PaymentURL = secondary.PaymentURL
	}
	if primary.RedirectURL == "" {
		primary.RedirectURL = secondary.RedirectURL
	}
	if primary.QRISString == "" {
		primary.QRISString = secondary.QRISString
	}
	if primary.QRISImageURL == "" {
		primary.QRISImageURL = secondary.QRISImageURL
	}
	if len(primary.RawBody) == 0 {
		primary.RawBody = secondary.RawBody
	}
	return primary
}

// ── Webhook DompetX ───────────────────────────────────────────────────────────

func WebhookDompetX(c *gin.Context) {
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gagal baca body"})
		return
	}

	var cfg models.PaymentConfig
	database.DB.First(&cfg, 1)
	if cfg.DompetXEnabled {
		sig := c.GetHeader("X-DOMPAY-Signature")
		if sig == "" {
			sig = c.GetHeader("X-Signature")
		}
		if sig == "" {
			sig = c.GetHeader("X-DompetX-Signature")
		}
		timestamp := c.GetHeader("X-DOMPAY-Timestamp")
		gw := gateway.NewDompetX(cfg.DompetXAPIKey, cfg.DompetXSecretKey, cfg.DompetXSandbox)
		if !gw.VerifyRequestSignature(rawBody, timestamp, sig) {
			log.Printf("[DOMPETX WEBHOOK] Invalid signature dari %s", c.ClientIP())
			c.JSON(http.StatusUnauthorized, gin.H{"error": "signature tidak valid"})
			return
		}
	}

	var payload any
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload tidak valid"})
		return
	}

	event := extractString(payload, "event", "type")
	status := extractString(payload, "status")
	reference := firstNonEmpty(
		extractString(payload, "reference", "order_id", "orderId"),
		extractString(payload, "invoice_no", "invoiceNo"),
	)
	transactionID := firstNonEmpty(
		extractString(payload, "id", "transaction_id", "transactionId", "charge_id"),
		reference,
	)
	var amount *int64
	if v := extractInt64(payload, "amount", "paid_amount", "gross_amount"); v > 0 {
		amount = &v
	}
	log.Printf("[DOMPETX WEBHOOK] event=%s ref=%s tx=%s status=%s", event, reference, transactionID, status)
	handleWebhookEvent("dompetx", reference, transactionID, event, status, amount, c)
}

// ── Webhook SayaBayar ─────────────────────────────────────────────────────────

func WebhookSayaBayar(c *gin.Context) {
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gagal baca body"})
		return
	}

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

	amount := payload.Invoice.Amount
	handleWebhookEvent("sayabayar", payload.Invoice.InvoiceNumber, payload.Invoice.ID, payload.Event, payload.Invoice.Status, &amount, c)
}

// ── handleWebhookEvent — logik bersama ───────────────────────────────────────

func handleWebhookEvent(provider, reference, transactionID, event, status string, amount *int64, c *gin.Context) {
	order, err := resolveWebhookOrder(reference, transactionID)
	if err != nil {
		log.Printf("[WEBHOOK] Order tidak ditemukan ref=%s tx=%s provider=%s", reference, transactionID, provider)
		c.JSON(http.StatusOK, gin.H{"message": "diabaikan"})
		return
	}
	if order.Status == "paid" || order.Status == "failed" || order.Status == "expired" || order.Status == "cancelled" {
		c.JSON(http.StatusOK, gin.H{"message": "sudah final"})
		return
	}
	if amount != nil && *amount > 0 && order.Total > 0 && *amount != order.Total {
		log.Printf("[WEBHOOK] Amount mismatch invoice=%s expected=%d got=%d", order.InvoiceNo, order.Total, *amount)
		database.DB.Model(&order).Update("status", "verifying")
		c.JSON(http.StatusOK, gin.H{"received": true, "status": "verifying"})
		return
	}

	updates := map[string]any{}
	if provider != "" && order.GatewayProvider == "" {
		updates["gateway_provider"] = provider
	}
	if transactionID != "" && order.GatewayChargeID == "" {
		updates["gateway_charge_id"] = transactionID
	}
	if reference != "" && order.GatewayInvoiceNo == "" {
		updates["gateway_invoice_no"] = reference
	}
	if len(updates) > 0 {
		database.DB.Model(&order).Updates(updates)
	}

	isPaid := event == "charge.paid" || event == "charge.success" || event == "invoice.paid" || status == "paid" || status == "success"
	isExpired := event == "charge.expired" || event == "invoice.expired" || status == "expired"
	isFailed := event == "charge.failed" || status == "failed"

	switch {
	case isPaid:
		database.DB.Model(&order).Update("status", "verifying")
		order.Status = "verifying"
		deliverOrder(order)
	case isExpired:
		database.DB.Model(&order).Update("status", "expired")
		restoreStock(order)
		log.Printf("[WEBHOOK] Order %s expired, stok dikembalikan", order.InvoiceNo)
	case isFailed:
		database.DB.Model(&order).Update("status", "failed")
		restoreStock(order)
	default:
		database.DB.Model(&order).Update("status", "verifying")
	}
	c.JSON(http.StatusOK, gin.H{"received": true})
}

func resolveWebhookOrder(reference, transactionID string) (*models.Order, error) {
	conds := []string{}
	args := []any{}
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		conds = append(conds, "invoice_no = ?", "gateway_invoice_no = ?", "gateway_charge_id = ?")
		args = append(args, value, value, value)
	}
	add(reference)
	if strings.TrimSpace(transactionID) != strings.TrimSpace(reference) {
		add(transactionID)
	}
	if len(conds) == 0 {
		return nil, fmt.Errorf("reference kosong")
	}
	query := strings.Join(conds, " OR ")
	var order models.Order
	if err := database.DB.Where(query, args...).First(&order).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

// ── deliverOrder ──────────────────────────────────────────────────────────────

func deliverOrder(order *models.Order) {
	var product models.Product
	database.DB.First(&product, order.ProductID)

	// Produk tipe provider: order langsung ke KoalaStore saat payment dikonfirmasi
	if order.ProductType == "provider" {
		providerProduct, provider, err := lookupProviderProductForOrder(order)
		if err != nil {
			log.Printf("[DELIVER] provider lookup gagal %s: %v", order.InvoiceNo, err)
			database.DB.Model(order).Update("status", "failed")
			return
		}
		if providerProduct != nil && (strings.EqualFold(providerProduct.Stock, "manual") || providerProduct.IsManual) {
			queued, err := queueManualProviderFulfillment(order, &product)
			if err != nil {
				log.Printf("[DELIVER] provider manual gagal %s: %v", order.InvoiceNo, err)
				database.DB.Model(order).Update("status", "failed")
				return
			}
			if queued {
				log.Printf("[DELIVER] provider manual ditahan %s provider=%s", order.InvoiceNo, provider.Name)
				return
			}
		}
		if order.DeliveredItems != "" {
			database.DB.Model(order).Updates(map[string]any{
				"status":             "paid",
				"fulfillment_status": "fulfilled",
				"is_fulfilled":       true,
			})
			return
		}
		delivered, err := OrderFromKoalaStore(order, &product)
		if err != nil {
			log.Printf("[DELIVER] KoalaStore order gagal %s: %v", order.InvoiceNo, err)
			database.DB.Model(order).Update("status", "failed")
			return
		}
		finalizeOrderDelivery(order, &product, delivered)
		return
	}
	if order.ProductType == "stock" {
		if order.DeliveredItems != "" {
			database.DB.Model(order).Updates(map[string]any{
				"status":             "paid",
				"fulfillment_status": "fulfilled",
				"is_fulfilled":       true,
			})
			return
		}
		claimed, err := ClaimStockItems(order.ProductID, order.Qty, order.InvoiceNo)
		if err != nil {
			log.Printf("[DELIVER] Gagal klaim stok %s: %v", order.InvoiceNo, err)
			database.DB.Model(order).Update("status", "failed")
			return
		}
		itemData := make([]string, len(claimed))
		for i, it := range claimed {
			itemData[i] = it.Data
		}
		finalizeOrderDelivery(order, &product, itemData)
	} else {
		markOrderPaid(order)
		database.DB.Model(order).Updates(map[string]any{
			"fulfillment_status": "fulfilled",
			"is_fulfilled":       true,
		})
		runPostPaymentAutomation(order, &product)
		go email.SendInvoiceService(order)
	}
}

func runPostPaymentAutomation(order *models.Order, product *models.Product) {
	if product == nil || strings.TrimSpace(product.Script) == "" {
		return
	}
	go func() {
		result := scripts.Execute(product.Script, order, email.SendWrapper)
		actJSON, _ := json.Marshal(result.Actions)
		database.DB.Create(&models.ScriptLog{
			OrderID: order.ID, InvoiceNo: order.InvoiceNo, Product: order.ProductName,
			Script: product.Script, Status: result.Status, Output: string(actJSON),
		})
		if result.Status == "failed" {
			log.Printf("[POST-PAYMENT] script gagal untuk %s", order.InvoiceNo)
		}
	}()
}

// ── CheckAndSyncGatewayStatus — polling manual saat pembeli buka halaman ──────

func CheckAndSyncGatewayStatus(order *models.Order) {
	if order.GatewayChargeID == "" {
		return
	}
	var cfg models.PaymentConfig
	database.DB.First(&cfg, 1)

	var status string
	var charge *gateway.ChargeResponse
	switch resolveGateway(&cfg) {
	case "sayabayar":
		sb := gateway.NewSayaBayar(cfg.SayaBayarAPIKey)
		resp, err := sb.GetInvoice(order.GatewayChargeID)
		if err != nil || !resp.Success {
			return
		}
		status = resp.Data.Status
	case "dompetx":
		gw := gateway.NewDompetX(cfg.DompetXAPIKey, cfg.DompetXSecretKey, cfg.DompetXSandbox)
		resp, err := gw.GetChargeStatus(order.GatewayChargeID)
		if err != nil || resp == nil {
			return
		}
		charge = resp
		status = resp.Status
	default:
		return
	}

	if order.Status != "pending" && order.Status != "waiting_payment" && order.Status != "verifying" {
		return
	}
	if charge != nil {
		updates := map[string]any{}
		if charge.PaymentURL != "" {
			updates["gateway_pay_url"] = charge.PaymentURL
		}
		if charge.RedirectURL != "" {
			updates["gateway_redirect_url"] = charge.RedirectURL
		}
		if charge.QRISString != "" {
			updates["gateway_pay_code"] = charge.QRISString
			updates["gateway_qris_string"] = charge.QRISString
		}
		if charge.QRISImageURL != "" {
			updates["gateway_qris_image_url"] = charge.QRISImageURL
		}
		if len(updates) > 0 {
			database.DB.Model(order).Updates(updates)
		}
	}
	switch status {
	case "paid", "success":
		deliverOrder(order)
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
		for range ticker.C {
			cancelExpiredOrders()
		}
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
	if len(expired) > 0 {
		log.Printf("[EXPIRY] %d order dicancel", len(expired))
	}
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
		if i > 0 && (len(r)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, ch)
	}
	return "Rp " + strings.TrimSpace(string(out))
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func extractString(root any, keys ...string) string {
	if v, ok := findValue(root, keys...); ok {
		switch val := v.(type) {
		case string:
			return strings.TrimSpace(val)
		case fmt.Stringer:
			return strings.TrimSpace(val.String())
		case float64:
			return strings.TrimSpace(strconv.FormatFloat(val, 'f', -1, 64))
		default:
			return strings.TrimSpace(fmt.Sprint(val))
		}
	}
	return ""
}

func extractInt64(root any, keys ...string) int64 {
	if v, ok := findValue(root, keys...); ok {
		switch val := v.(type) {
		case float64:
			return int64(val)
		case float32:
			return int64(val)
		case int:
			return int64(val)
		case int64:
			return val
		case string:
			n, _ := strconv.ParseInt(strings.TrimSpace(val), 10, 64)
			return n
		}
	}
	return 0
}

func findValue(root any, keys ...string) (any, bool) {
	targets := map[string]struct{}{}
	for _, key := range keys {
		targets[normalizeKey(key)] = struct{}{}
	}
	var walk func(any) (any, bool)
	walk = func(node any) (any, bool) {
		switch val := node.(type) {
		case map[string]any:
			for k, item := range val {
				if _, ok := targets[normalizeKey(k)]; ok {
					return item, true
				}
			}
			for _, item := range val {
				if found, ok := walk(item); ok {
					return found, true
				}
			}
		case []any:
			for _, item := range val {
				if found, ok := walk(item); ok {
					return found, true
				}
			}
		}
		return nil, false
	}
	return walk(root)
}

func normalizeKey(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	replacer := strings.NewReplacer("_", "", "-", "", " ", "")
	return replacer.Replace(s)
}
