package models

import "time"

type Product struct {
	ID             uint      `json:"id"              gorm:"primaryKey;autoIncrement"`
	Name           string    `json:"name"            gorm:"not null"`
	Description    string    `json:"description"`
	Price          int64     `json:"price"           gorm:"not null"`
	Category       string    `json:"category"`
	Type           string    `json:"type"            gorm:"not null;default:'stock'"`
	Icon           string    `json:"icon"            gorm:"default:'📦'"`
	Active         bool      `json:"active"          gorm:"default:true"`
	ImageURL       string    `json:"image_url"       gorm:"default:''"`
	Script         string    `json:"script"          gorm:"type:text"`
	AvailableStock int       `json:"available_stock" gorm:"-"`
	TotalStock     int       `json:"total_stock"     gorm:"-"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ProductStock struct {
	ID        uint       `json:"id"         gorm:"primaryKey;autoIncrement"`
	ProductID uint       `json:"product_id" gorm:"not null;index"`
	Data      string     `json:"data"       gorm:"type:text;not null"`
	Sold      bool       `json:"sold"       gorm:"default:false"`
	InvoiceNo string     `json:"invoice_no" gorm:"default:''"`
	SoldAt    *time.Time `json:"sold_at"`
	CreatedAt time.Time  `json:"created_at"`
}

type Order struct {
	ID             uint       `json:"id"              gorm:"primaryKey;autoIncrement"`
	InvoiceNo      string     `json:"invoice_no"      gorm:"uniqueIndex;not null"`
	ProductID      uint       `json:"product_id"`
	ProductName    string     `json:"product_name"`
	ProductType    string     `json:"product_type"`
	Qty            int        `json:"qty"             gorm:"default:1"`
	Price          int64      `json:"price"`
	Total          int64      `json:"total"`
	BuyerName      string     `json:"buyer_name"`
	BuyerEmail     string     `json:"buyer_email"`
	PayMethod      string     `json:"pay_method"`
	// Status: pending | paid | expired | cancelled | script_executed | failed
	Status         string     `json:"status"          gorm:"default:'pending'"`
	DeliveredItems string     `json:"delivered_items" gorm:"type:text"`
	Notes          string     `json:"notes"`

	// Payment Gateway fields
	GatewayChargeID  string     `json:"gateway_charge_id"   gorm:"default:''"`  // ID charge/invoice di sisi gateway
	GatewayInvoiceNo string     `json:"gateway_invoice_no"  gorm:"default:''"`  // Nomor invoice dari gateway (untuk sync)
	GatewayProvider  string     `json:"gateway_provider"    gorm:"default:''"`  // "sayabayar" | "dompetx" | ""
	GatewayPayURL    string     `json:"gateway_pay_url"     gorm:"default:''"`
	GatewayPayCode   string     `json:"gateway_pay_code"    gorm:"default:''"`
	ExpiredAt        *time.Time `json:"expired_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PaymentConfig struct {
	ID         uint   `json:"id"          gorm:"primaryKey"`
	BankName   string `json:"bank_name"`
	BankNo     string `json:"bank_no"`
	BankAcc    string `json:"bank_acc"`
	Dana       string `json:"dana"`
	Gopay      string `json:"gopay"`
	Ovo        string `json:"ovo"`
	QRIS       bool   `json:"qris"`
	Crypto     bool   `json:"crypto"`
	CryptoAddr string `json:"crypto_addr"`
	// DompetX gateway settings
	DompetXEnabled   bool   `json:"dompetx_enabled"    gorm:"default:false"`
	DompetXAPIKey    string `json:"dompetx_api_key"    gorm:"default:''"`
	DompetXSecretKey string `json:"dompetx_secret_key" gorm:"default:''"`
	DompetXSandbox   bool   `json:"dompetx_sandbox"    gorm:"default:true"`
	// Berapa jam sampai order expired (default: 24)
	PaymentExpireHours int `json:"payment_expire_hours" gorm:"default:24"`

	// SayaBayar gateway settings
	SayaBayarEnabled   bool   `json:"sayabayar_enabled"    gorm:"default:false"`
	SayaBayarAPIKey    string `json:"sayabayar_api_key"    gorm:"default:''"`
	// channel_preference: "platform" (default) | "client" (dana langsung ke rekening)
	SayaBayarChannel   string `json:"sayabayar_channel"    gorm:"default:'platform'"`
}

type ScriptLog struct {
	ID        uint      `json:"id"         gorm:"primaryKey;autoIncrement"`
	OrderID   uint      `json:"order_id"`
	InvoiceNo string    `json:"invoice_no"`
	Product   string    `json:"product"`
	Script    string    `json:"script"     gorm:"type:text"`
	Status    string    `json:"status"`
	Output    string    `json:"output"     gorm:"type:text"`
	CreatedAt time.Time `json:"created_at"`
}

type ProviderAction struct {
	Provider   string            `json:"provider"`
	Enabled    bool              `json:"enabled"`
	Label      string            `json:"label,omitempty"`
	To         string            `json:"to,omitempty"`
	Subject    string            `json:"subject,omitempty"`
	Body       string            `json:"body,omitempty"`
	URL        string            `json:"url,omitempty"`
	Method     string            `json:"method,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	WebhookURL string            `json:"webhook_url,omitempty"`
	Message    string            `json:"message,omitempty"`
}

// ── StockProvider ─────────────────────────────────────────────────────────────
// Provider eksternal yang menyuplai item stok via API.
// Saat pull dilakukan, response di-parse dan item baru ditambahkan ke ProductStock.

type StockProvider struct {
	ID          uint      `json:"id"           gorm:"primaryKey;autoIncrement"`
	Name        string    `json:"name"         gorm:"not null"`       // Nama provider
	ProductID   uint      `json:"product_id"   gorm:"not null;index"` // Produk tujuan
	Type        string    `json:"type"         gorm:"not null"`       // "http_api" | "csv_url"
	// HTTP API config
	APIURL      string    `json:"api_url"      gorm:"type:text"`      // URL endpoint provider
	APIMethod   string    `json:"api_method"   gorm:"default:'GET'"` // GET | POST
	APIHeaders  string    `json:"api_headers"  gorm:"type:text"`      // JSON: {"X-Key":"val"}
	APIBody     string    `json:"api_body"     gorm:"type:text"`      // JSON body jika POST
	// Response parsing
	ItemsPath   string    `json:"items_path"`                         // JSONPath ke array, mis: "data.items"
	ItemField   string    `json:"item_field"`                         // Field tiap item, mis: "key" atau "url"
	// Status
	Active      bool      `json:"active"       gorm:"default:true"`
	LastPullAt  *time.Time `json:"last_pull_at"`
	LastCount   int       `json:"last_count"`  // Jumlah item terakhir di-pull
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PullLog — log setiap kali pull dilakukan
type PullLog struct {
	ID         uint      `json:"id"          gorm:"primaryKey;autoIncrement"`
	ProviderID uint      `json:"provider_id" gorm:"index"`
	ProductID  uint      `json:"product_id"`
	Status     string    `json:"status"`     // success | failed | partial
	Count      int       `json:"count"`      // item berhasil ditambahkan
	Message    string    `json:"message"     gorm:"type:text"`
	CreatedAt  time.Time `json:"created_at"`
}
