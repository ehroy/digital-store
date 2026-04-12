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
	"time"
)

// ── Konfigurasi ────────────────────────────────────────────────────────────────

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

// ── Request / Response structs ─────────────────────────────────────────────────
// Sesuaikan field ini dengan docs DompetX yang sebenarnya.

type ChargeRequest struct {
	OrderID       string   `json:"order_id"`       // nomor invoice internal
	Amount        int64    `json:"amount"`         // dalam Rupiah (bukan sen)
	Currency      string   `json:"currency"`       // "IDR"
	PaymentMethod string   `json:"payment_method"` // "va_bca","qris","gopay",dll
	CustomerName  string   `json:"customer_name"`
	CustomerEmail string   `json:"customer_email"`
	Description   string   `json:"description"`
	ExpiredAt     string   `json:"expired_at"` // ISO8601, mis: "2026-04-10T12:00:00Z"
	CallbackURL   string   `json:"callback_url"`  // URL webhook kamu
	ReturnURL     string   `json:"return_url"`    // redirect setelah bayar
	Metadata      Metadata `json:"metadata"`
}

type Metadata struct {
	InvoiceNo string `json:"invoice_no"`
	Source    string `json:"source"` // "digistore"
}

type ChargeResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		ChargeID      string `json:"charge_id"`
		OrderID       string `json:"order_id"`
		Status        string `json:"status"`        // "pending","paid","expired","failed"
		Amount        int64  `json:"amount"`
		PaymentURL    string `json:"payment_url"`   // URL redirect ke halaman bayar DompetX
		PaymentCode   string `json:"payment_code"`  // nomor VA / kode bayar jika ada
		ExpiredAt     string `json:"expired_at"`
	} `json:"data"`
}

// WebhookPayload adalah struktur yang dikirim DompetX ke endpoint webhook kamu.
type WebhookPayload struct {
	Event    string `json:"event"`    // "charge.paid","charge.expired","charge.failed"
	ChargeID string `json:"charge_id"`
	OrderID  string `json:"order_id"` // sama dengan invoice_no
	Status   string `json:"status"`
	Amount   int64  `json:"amount"`
	PaidAt   string `json:"paid_at,omitempty"`
	Metadata struct {
		InvoiceNo string `json:"invoice_no"`
	} `json:"metadata"`
}

// StatusResponse untuk cek status charge secara manual
type StatusResponse struct {
	Success bool `json:"success"`
	Data    struct {
		ChargeID  string `json:"charge_id"`
		OrderID   string `json:"order_id"`
		Status    string `json:"status"`
		Amount    int64  `json:"amount"`
		PaidAt    string `json:"paid_at,omitempty"`
		ExpiredAt string `json:"expired_at"`
	} `json:"data"`
}

// ── Methods ────────────────────────────────────────────────────────────────────

// CreateCharge membuat tagihan baru di DompetX.
// Mengembalikan ChargeResponse berisi payment_url untuk redirect pembeli.
func (d *DompetX) CreateCharge(req ChargeRequest) (*ChargeResponse, error) {
	body, _ := json.Marshal(req)

	httpReq, err := http.NewRequest("POST", d.BaseURL+"/charges", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("dompetx: buat request gagal: %w", err)
	}
	d.setHeaders(httpReq)

	resp, err := d.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("dompetx: request gagal: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result ChargeResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("dompetx: parse response gagal: %w — body: %s", err, string(respBody))
	}
	if !result.Success {
		return nil, fmt.Errorf("dompetx: %s", result.Message)
	}
	return &result, nil
}

// GetChargeStatus mengambil status charge dari DompetX (untuk polling manual).
func (d *DompetX) GetChargeStatus(chargeID string) (*StatusResponse, error) {
	httpReq, err := http.NewRequest("GET", d.BaseURL+"/charges/"+chargeID, nil)
	if err != nil {
		return nil, err
	}
	d.setHeaders(httpReq)

	resp, err := d.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result StatusResponse
	json.Unmarshal(body, &result)
	return &result, nil
}

// VerifySignature memvalidasi webhook signature dari DompetX.
// DompetX mengirim header X-Signature: HMAC-SHA256(secret_key, raw_body)
func (d *DompetX) VerifySignature(rawBody []byte, signature string) bool {
	mac := hmac.New(sha256.New, []byte(d.SecretKey))
	mac.Write(rawBody)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

func (d *DompetX) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+d.APIKey)
	req.Header.Set("Accept", "application/json")
}

// MapPayMethod memetakan metode bayar internal (bank, gopay, dll) ke kode DompetX.
// Sesuaikan mapping ini dengan payment_method yang didukung DompetX.
func MapPayMethod(internal string) string {
	m := map[string]string{
		"bank":   "va_bca",     // default VA BCA; bisa diganti sesuai pilihan
		"dana":   "ewallet_dana",
		"gopay":  "ewallet_gopay",
		"ovo":    "ewallet_ovo",
		"qris":   "qris",
		"crypto": "crypto_btc", // jika didukung
	}
	if v, ok := m[internal]; ok {
		return v
	}
	return "qris" // fallback
}
