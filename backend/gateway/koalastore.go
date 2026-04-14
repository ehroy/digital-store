// Package gateway — KoalaStore.Digital Integration
// Base URL: https://koalastore.digital/api/v1
// Auth: Header X-API-Key
package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const KoalaStoreBaseURL = "https://koalastore.digital/api/v1"

type KoalaStore struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}

func NewKoalaStore(apiKey string) *KoalaStore {
	return &KoalaStore{
		APIKey:     apiKey,
		BaseURL:    KoalaStoreBaseURL,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// ── Structs ──────────────────────────────────────────────────────────────────

type KSVariant struct {
	CodeVariant        string  `json:"code_variant"`
	Name               string  `json:"name"`
	Price              int64   `json:"price"`
	AvailableStock     int     `json:"available_stock"`
	IsManualProcess    bool    `json:"is_manual_process"`
	TermsAndConditions string  `json:"terms_and_conditions"`
	WarrantyTerms      string  `json:"warranty_terms"`
}

type KSProduct struct {
	Code        string      `json:"code"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Category    string      `json:"category"`
	Variants    []KSVariant `json:"variants"`
}

type KSProductDetail struct {
	Code            string      `json:"code"`
	Name            string      `json:"name"`
	Description     string      `json:"description"`
	LongDescription string      `json:"long_description"`
	Category        string      `json:"category"`
	Image           string      `json:"image"`
	Variants        []KSVariant `json:"variants"`
}

type KSProductsResp struct {
	Success bool        `json:"success"`
	Data    []KSProduct `json:"data"`
	Meta    struct {
		CurrentPage int `json:"current_page"`
		LastPage    int `json:"last_page"`
		PerPage     int `json:"per_page"`
		Total       int `json:"total"`
	} `json:"meta"`
	Message string `json:"message"`
}

type KSProductDetailResp struct {
	Success bool            `json:"success"`
	Data    KSProductDetail `json:"data"`
	Message string          `json:"message"`
}

type KSBalanceData struct {
	Balance                   int64  `json:"balance"`
	FormattedBalance          string `json:"formatted_balance"`
	TotalBalance              int64  `json:"total_balance"`
	FormattedTotalBalance     string `json:"formatted_total_balance"`
	PendingWithdrawal         int64  `json:"pending_withdrawal"`
	TotalTopup                int64  `json:"total_topup"`
	TotalSpent                int64  `json:"total_spent"`
}

type KSBalanceResp struct {
	Success bool          `json:"success"`
	Message string        `json:"message"`
	Code    int           `json:"code"`
	Data    KSBalanceData `json:"data"`
}

type KSCheckoutItem struct {
	VariantCode string `json:"variant_code"`
	Quantity    int    `json:"quantity"`
}

type KSCheckoutReq struct {
	Items []KSCheckoutItem `json:"items"`
}

type KSStockData struct {
	DataStock string `json:"dataStock"`
}

type KSOrderItem struct {
	VariantCode        string        `json:"variant_code"`
	ProductName        string        `json:"product_name"`
	VariantName        string        `json:"variant_name"`
	Quantity           int           `json:"quantity"`
	UnitPrice          int64         `json:"unit_price"`
	Subtotal           int64         `json:"subtotal"`
	StockData          []KSStockData `json:"stock_data"`
	TermsAndConditions string        `json:"terms_and_conditions"`
	WarrantyTerms      string        `json:"warranty_terms"`
}

type KSCheckoutResp struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		TransactionID    string        `json:"transaction_id"`
		PIN              int           `json:"pin"`
		TotalAmount      int64         `json:"total_amount"`
		BalanceUsed      int64         `json:"balance_used"`
		BalanceRemaining int64         `json:"balance_remaining"`
		Items            []KSOrderItem `json:"items"`
	} `json:"data"`
}

// ── Methods ──────────────────────────────────────────────────────────────────

// GetBalance — GET /balance
func (k *KoalaStore) GetBalance() (*KSBalanceResp, error) {
	var resp KSBalanceResp
	if err := k.get("/balance", &resp); err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("koalastore: %s", resp.Message)
	}
	return &resp, nil
}

// GetProducts — GET /products (semua halaman atau satu halaman)
// Jika page=0 maka fetch semua halaman otomatis
func (k *KoalaStore) GetProducts(page, perPage int, search, category string) (*KSProductsResp, error) {
	url := fmt.Sprintf("/products?page=%d&per_page=%d", page, perPage)
	if search != "" { url += "&search=" + search }
	if category != "" { url += "&category=" + category }

	var resp KSProductsResp
	if err := k.get(url, &resp); err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("koalastore: %s", resp.Message)
	}
	return &resp, nil
}

// GetAllProducts — fetch semua produk dari semua halaman
func (k *KoalaStore) GetAllProducts() ([]KSProduct, error) {
	var all []KSProduct
	page := 1
	for {
		resp, err := k.GetProducts(page, 50, "", "")
		if err != nil { return nil, err }
		all = append(all, resp.Data...)
		if page >= resp.Meta.LastPage { break }
		page++
	}
	return all, nil
}

// GetProductDetail — GET /products/{code}
func (k *KoalaStore) GetProductDetail(code string) (*KSProductDetailResp, error) {
	var resp KSProductDetailResp
	if err := k.get("/products/"+code, &resp); err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("koalastore: product not found")
	}
	return &resp, nil
}

// Checkout — POST /checkout
// Membeli produk menggunakan saldo KoalaStore
// Mengembalikan stock_data yang berisi konten untuk dikirim ke pembeli
func (k *KoalaStore) Checkout(variantCode string, qty int) (*KSCheckoutResp, error) {
	reqBody := KSCheckoutReq{
		Items: []KSCheckoutItem{{VariantCode: variantCode, Quantity: qty}},
	}
	var resp KSCheckoutResp
	if err := k.post("/checkout", reqBody, &resp); err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("koalastore checkout gagal: %s", resp.Message)
	}
	return &resp, nil
}

// ── HTTP helpers ─────────────────────────────────────────────────────────────

func (k *KoalaStore) get(path string, out interface{}) error {
	req, err := http.NewRequest("GET", k.BaseURL+path, nil)
	if err != nil { return err }
	k.setHeaders(req)
	resp, err := k.HTTPClient.Do(req)
	if err != nil { return fmt.Errorf("koalastore request: %w", err) }
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("koalastore parse: %w — %s", err, string(body)[:min200(len(body))])
	}
	return nil
}

func (k *KoalaStore) post(path string, in, out interface{}) error {
	data, _ := json.Marshal(in)
	req, err := http.NewRequest("POST", k.BaseURL+path, bytes.NewBuffer(data))
	if err != nil { return err }
	k.setHeaders(req)
	resp, err := k.HTTPClient.Do(req)
	if err != nil { return fmt.Errorf("koalastore request: %w", err) }
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("koalastore parse: %w — %s", err, string(body)[:min200(len(body))])
	}
	return nil
}

func (k *KoalaStore) setHeaders(req *http.Request) {
	req.Header.Set("X-API-Key", k.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
}

func min200(n int) int { if n > 200 { return 200 }; return n }
