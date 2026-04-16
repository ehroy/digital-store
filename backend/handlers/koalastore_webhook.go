package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"digistore/config"
	"digistore/database"
	"digistore/models"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// WebhookKoalaStore menerima notifikasi manual provider dari KoalaStore.
// Secret test default: Kaserinas123@
func WebhookKoalaStore(c *gin.Context) {
	rawBody, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gagal baca body"})
		return
	}

	sig := firstNonEmpty(
		c.GetHeader("X-Signature"),
		c.GetHeader("X-Hub-Signature-256"),
		c.GetHeader("X-Koala-Signature"),
		c.GetHeader("X-Webhook-Signature"),
		c.GetHeader("Signature"),
	)

	var payload any
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload tidak valid"})
		return
	}

	if sig == "" {
		fallbackSecret := firstNonEmpty(extractWebhookString(payload, "password", "secret", "webhook_password"))
		if fallbackSecret != config.App.KoalaStoreWebhookSecret {
			log.Printf("[KOALASTORE WEBHOOK] password fallback invalid ip=%s", c.ClientIP())
			c.JSON(http.StatusUnauthorized, gin.H{"error": "password tidak valid"})
			return
		}
	} else if !verifyKoalaWebhookSignature(rawBody, sig) {
		log.Printf("[KOALASTORE WEBHOOK] signature invalid ip=%s", c.ClientIP())
		c.JSON(http.StatusUnauthorized, gin.H{"error": "signature tidak valid"})
		return
	}

	reference := firstNonEmpty(
		extractWebhookString(payload, "invoice_no", "invoiceNo", "reference", "order_no", "orderNo"),
		extractWebhookString(payload, "transaction_id", "transactionId", "provider_order_id", "providerOrderId"),
	)
	status := strings.ToLower(strings.TrimSpace(extractWebhookString(payload, "status", "event", "state")))
	items := extractKoalaWebhookItems(payload)

	order, providerOrder, err := resolveKoalaWebhookOrder(reference)
	if err != nil {
		log.Printf("[KOALASTORE WEBHOOK] order not found ref=%s err=%v", reference, err)
		c.JSON(http.StatusOK, gin.H{"message": "diabaikan"})
		return
	}

	if providerOrder != nil {
		updates := map[string]any{}
		if providerOrder.ProviderOrderID == "" {
			if tx := extractWebhookString(payload, "transaction_id", "transactionId", "provider_order_id", "providerOrderId"); tx != "" {
				updates["provider_order_id"] = tx
			}
		}
		if len(updates) > 0 {
			database.DB.Model(providerOrder).Updates(updates)
		}
	}

	if len(items) == 0 {
		items = extractKoalaWebhookItemsFromRawOrder(payload)
	}

	switch {
	case isKoalaWebhookFailed(status):
		database.DB.Model(order).Updates(map[string]any{
			"fulfillment_status": "failed",
			"notes":              "KoalaStore webhook menandai gagal",
		})
		if providerOrder != nil {
			database.DB.Model(providerOrder).Updates(map[string]any{"status": "failed", "message": "Webhook menandai gagal"})
		}
		c.JSON(http.StatusOK, gin.H{"message": "gagal dicatat"})
		return
	case len(items) > 0:
		product := &models.Product{}
		if err := database.DB.First(product, order.ProductID).Error; err == nil {
			finalizeOrderDelivery(order, product, items)
			updateProviderOrderDelivered(order, items, "Webhook KoalaStore: barang dikirim")
		} else {
			finalizeOrderDelivery(order, nil, items)
			updateProviderOrderDelivered(order, items, "Webhook KoalaStore: barang dikirim")
		}
		c.JSON(http.StatusOK, gin.H{"message": "berhasil"})
		return
	default:
		if strings.TrimSpace(status) != "" {
			database.DB.Model(order).Update("fulfillment_status", "waiting_provider")
		}
		c.JSON(http.StatusOK, gin.H{"message": "diterima"})
		return
	}
}

func verifyKoalaWebhookSignature(rawBody []byte, signature string) bool {
	signature = strings.TrimSpace(signature)
	if signature == "" {
		return false
	}
	signature = strings.TrimPrefix(signature, "sha256=")
	secret := []byte(config.App.KoalaStoreWebhookSecret)
	if len(secret) == 0 {
		return false
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write(rawBody)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

func resolveKoalaWebhookOrder(reference string) (*models.Order, *models.ProviderOrder, error) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return nil, nil, fmt.Errorf("reference kosong")
	}

	var order models.Order
	if err := database.DB.Where("invoice_no = ? OR gateway_invoice_no = ?", reference, reference).First(&order).Error; err == nil {
		var providerOrder models.ProviderOrder
		if err := database.DB.Where("order_id = ?", order.ID).Order("created_at desc").First(&providerOrder).Error; err == nil {
			return &order, &providerOrder, nil
		}
		return &order, nil, nil
	}

	var providerOrder models.ProviderOrder
	if err := database.DB.Where("provider_order_id = ? OR invoice_no = ?", reference, reference).Order("created_at desc").First(&providerOrder).Error; err == nil {
		if err := database.DB.First(&order, providerOrder.OrderID).Error; err == nil {
			return &order, &providerOrder, nil
		}
	}

	return nil, nil, fmt.Errorf("order tidak ditemukan")
}

func extractWebhookString(root any, keys ...string) string {
	if v, ok := findWebhookValue(root, keys...); ok {
		switch val := v.(type) {
		case string:
			return strings.TrimSpace(val)
		case fmt.Stringer:
			return strings.TrimSpace(val.String())
		case float64:
			return strings.TrimSpace(fmt.Sprintf("%v", val))
		default:
			return strings.TrimSpace(fmt.Sprint(val))
		}
	}
	return ""
}

func findWebhookValue(root any, keys ...string) (any, bool) {
	current := root
	for _, key := range keys {
		segments := strings.Split(key, ".")
		current = root
		found := true
		for _, seg := range segments {
			switch node := current.(type) {
			case map[string]any:
				v, ok := node[seg]
				if !ok {
					found = false
					break
				}
				current = v
			case []any:
				idx, err := strconv.Atoi(seg)
				if err != nil || idx < 0 || idx >= len(node) {
					found = false
					break
				}
				current = node[idx]
			default:
				found = false
				break
			}
		}
		if found {
			return current, true
		}
	}
	return nil, false
}

func extractKoalaWebhookItems(root any) []string {
	paths := []string{"items", "stock_data", "data.items", "data.stock_data", "data.items.0.stock_data"}
	for _, path := range paths {
		if v, ok := findWebhookValue(root, path); ok {
			if items := normalizeWebhookItems(v); len(items) > 0 {
				return items
			}
		}
	}
	return nil
}

func extractKoalaWebhookItemsFromRawOrder(root any) []string {
	if v, ok := findWebhookValue(root, "data.items.0.stock_data"); ok {
		if items := normalizeWebhookItems(v); len(items) > 0 {
			return items
		}
	}
	return nil
}

func normalizeWebhookItems(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	items := make([]string, 0, len(arr))
	for _, el := range arr {
		switch val := el.(type) {
		case string:
			if strings.TrimSpace(val) != "" {
				items = append(items, strings.TrimSpace(val))
			}
		case map[string]any:
			for _, key := range []string{"data_stock", "dataStock", "serial", "value", "content"} {
				if raw, ok := val[key]; ok {
					text := strings.TrimSpace(fmt.Sprint(raw))
					if text != "" {
						items = append(items, text)
						break
					}
				}
			}
		}
	}
	return items
}

func isKoalaWebhookFailed(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "failed", "cancelled", "canceled", "expired":
		return true
	default:
		return false
	}
}
