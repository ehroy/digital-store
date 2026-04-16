// Package gateway mengintegrasikan DompetX Payment Gateway.
//
// Sesuaikan BASE_URL dan field name jika ada perbedaan dengan docs resmi di
// https://docs.dompetx.com/api-reference/introduction
//
// Alur pembayaran:
//  1. CreateCharge()  → dapat charge_id + payment_url
//  2. Redirect pembeli ke payment_url
//  3. DompetX kirim webhook ke /api/webhook/dompetx saat status berubah
//  4. VerifySignature() untuk validasi webhook
//  5. Background job (ExpiredOrderJob) cancel order yang melebihi batas waktu
package gateway

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	BaseURLSandbox = "https://api-sandbox.dompetx.com/v1"
	BaseURLProd    = "https://api.dompetx.com/v1"
)

type DompetX struct {
	APIKey     string
	SecretKey  string // untuk verifikasi webhook signature
	BaseURL    string
	HTTPClient *http.Client
}

func NewDompetX(apiKey, secretKey string, sandbox bool) *DompetX {
	base := BaseURLProd
	if sandbox {
		base = BaseURLSandbox
	}
	return &DompetX{
		APIKey:     apiKey,
		SecretKey:  secretKey,
		BaseURL:    base,
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
	}
}

type ChargeRequest struct {
	Method          string         `json:"method"`
	Amount          int64          `json:"amount"`
	Currency        string         `json:"currency"`
	Reference       string         `json:"reference"`
	ReturnURL       string         `json:"return_url,omitempty"`
	CallbackURL     string         `json:"callback_url,omitempty"`
	SettlementSpeed string         `json:"settlementSpeed,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

type ChargeResponse struct {
	TransactionID string
	Reference     string
	Status        string
	Amount        int64
	Currency      string
	PaymentURL    string
	RedirectURL   string
	QRISString    string
	QRISImageURL  string
	RawBody       []byte
}

func (d *DompetX) CreateCharge(req ChargeRequest) (*ChargeResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("dompetx: marshal request gagal: %w", err)
	}
	httpReq, err := d.newSignedRequest(http.MethodPost, "/payments", body, req.Reference)
	if err != nil {
		return nil, err
	}
	resp, err := d.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("dompetx: request gagal: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	charge, err := parseDompetXCharge(respBody)
	if err != nil {
		return nil, err
	}
	return charge, nil
}

func (d *DompetX) GetChargeStatus(transactionID string) (*ChargeResponse, error) {
	httpReq, err := d.newSignedRequest(http.MethodGet, "/payments/check-status/"+transactionID, nil, transactionID)
	if err != nil {
		return nil, err
	}
	resp, err := d.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("dompetx: request status gagal: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return parseDompetXCharge(body)
}

func (d *DompetX) VerifySignature(rawBody []byte, signature string) bool {
	return d.VerifyRequestSignature(rawBody, "", signature)
}

func (d *DompetX) VerifyRequestSignature(rawBody []byte, timestamp, signature string) bool {
	signature = strings.TrimSpace(signature)
	if signature == "" {
		return false
	}
	if strings.TrimSpace(timestamp) != "" {
		expected := d.signPayload(timestamp, rawBody)
		if hmac.Equal([]byte(expected), []byte(signature)) {
			return true
		}
	}
	if d.SecretKey != "" {
		mac := hmac.New(sha256.New, []byte(d.SecretKey))
		mac.Write(rawBody)
		expected := hex.EncodeToString(mac.Sum(nil))
		if hmac.Equal([]byte(expected), []byte(signature)) {
			return true
		}
	}
	return false
}

func (d *DompetX) newSignedRequest(method, path string, body []byte, idempotencyKey string) (*http.Request, error) {
	if body == nil {
		body = []byte{}
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	req, err := http.NewRequest(method, strings.TrimRight(d.BaseURL, "/")+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("dompetx: buat request gagal: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-DOMPAY-API-Key", d.APIKey)
	req.Header.Set("X-DOMPAY-Timestamp", timestamp)
	req.Header.Set("X-DOMPAY-Signature", d.signPayload(timestamp, body))
	if strings.TrimSpace(idempotencyKey) == "" {
		idempotencyKey = "dompetx-" + timestamp
	}
	req.Header.Set("Idempotency-Key", idempotencyKey)
	return req, nil
}

func (d *DompetX) signPayload(timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(d.APIKey))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func parseDompetXCharge(rawBody []byte) (*ChargeResponse, error) {
	var payload any
	if len(rawBody) == 0 {
		return nil, fmt.Errorf("dompetx: response kosong")
	}
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return nil, fmt.Errorf("dompetx: parse response gagal: %w — body: %s", err, string(rawBody))
	}
	charge := &ChargeResponse{RawBody: rawBody}
	charge.TransactionID = extractString(payload, "id", "transaction_id", "transactionId", "charge_id", "payment_id")
	charge.Reference = extractString(payload, "reference", "order_id", "orderId")
	charge.Status = extractString(payload, "status")
	charge.Currency = extractString(payload, "currency")
	charge.Amount = extractInt64(payload, "amount")
	charge.PaymentURL = extractString(payload, "payment_url", "paymentUrl", "checkout_url", "checkoutUrl", "redirect_url", "redirectUrl")
	charge.RedirectURL = firstNonEmpty(
		extractString(payload, "redirect_url", "redirectUrl", "payment_url", "paymentUrl", "checkout_url", "checkoutUrl"),
		extractString(payload, "web_url", "webUrl"),
	)
	charge.QRISString = extractString(payload, "qris", "qris_string", "qrisString", "qr_string", "qrString", "qr_data", "qrData")
	charge.QRISImageURL = extractString(payload, "qris_image_url", "qrisImageUrl", "qr_image_url", "qrImageUrl", "qr_url", "qrUrl")
	if charge.Reference == "" {
		charge.Reference = charge.TransactionID
	}
	if charge.RedirectURL == "" {
		charge.RedirectURL = charge.PaymentURL
	}
	if charge.TransactionID == "" && charge.Reference == "" {
		return nil, fmt.Errorf("dompetx: transaksi tidak ditemukan di response: %s", string(rawBody))
	}
	return charge, nil
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
		case json.Number:
			return strings.TrimSpace(val.String())
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
		case json.Number:
			n, _ := val.Int64()
			return n
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

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func MapPayMethod(internal string) string {
	return "QRIS"
}
