package models

import "time"

type Product struct {
	ID             uint             `json:"id"              gorm:"primaryKey;autoIncrement"`
	Name           string           `json:"name"            gorm:"not null"`
	Description    string           `json:"description"`
	Price          int64            `json:"price"           gorm:"not null"`
	Category       string           `json:"category"`
	Type           string           `json:"type"            gorm:"not null;default:'stock'"`
	Icon           string           `json:"icon"            gorm:"default:'📦'"`
	Active         bool             `json:"active"          gorm:"default:true"`
	ImageURL       string           `json:"image_url"       gorm:"default:''"`
	Script         string           `json:"script"          gorm:"type:text"`
	AvailableStock int              `json:"available_stock"  gorm:"-"` // untuk tipe stock: jumlah item tersedia
	TotalStock     int              `json:"total_stock"      gorm:"-"` // untuk tipe stock: total item
	ProviderStock  int              `json:"provider_stock"   gorm:"-"` // untuk tipe provider: stok real dari KoalaStore
	ProviderStatus string           `json:"provider_status"  gorm:"-"` // "available"|"out_of_stock"|"manual"
	Variants       []CatalogVariant `json:"variants,omitempty" gorm:"-"`

	// Provider product fields — diisi jika Type = "provider"
	// Produk diambil dari API provider eksternal (mis: KoalaStore)
	ProviderName  string    `json:"provider_name"   gorm:"default:''"`        // nama provider
	ProviderCode  string    `json:"provider_code"   gorm:"default:''"`        // kode produk di provider
	ProviderPrice int64     `json:"provider_price"  gorm:"default:0"`         // harga beli dari provider
	MarkupType    string    `json:"markup_type"     gorm:"default:'percent'"` // "percent" | "fixed"
	MarkupValue   float64   `json:"markup_value"    gorm:"default:0"`         // persen atau nominal
	AutoSync      bool      `json:"auto_sync"       gorm:"default:false"`     // sync harga otomatis
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CatalogVariant struct {
	ProductID      uint   `json:"product_id"`
	ProviderCode   string `json:"provider_code"`
	VariantName    string `json:"variant_name"`
	DurationLabel  string `json:"duration_label,omitempty"`
	AccountType    string `json:"account_type,omitempty"`
	Region         string `json:"region,omitempty"`
	StockStatus    string `json:"stock_status"`
	AvailableStock int    `json:"available_stock"`
	Price          int64  `json:"price"`
	OriginalPrice  int64  `json:"original_price,omitempty"`
	IsActive       bool   `json:"is_active"`
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
	ID          uint   `json:"id"              gorm:"primaryKey;autoIncrement"`
	InvoiceNo   string `json:"invoice_no"      gorm:"uniqueIndex;not null"`
	ProductID   uint   `json:"product_id"`
	ProductName string `json:"product_name"`
	ProductType string `json:"product_type"`
	Qty         int    `json:"qty"             gorm:"default:1"`
	Price       int64  `json:"price"`
	Total       int64  `json:"total"`
	BuyerName   string `json:"buyer_name"`
	BuyerEmail  string `json:"buyer_email"`
	PayMethod   string `json:"pay_method"`
	// Status: pending | paid | expired | cancelled | script_executed | failed
	Status             string     `json:"status"          gorm:"default:'pending'"`
	FulfillmentStatus  string     `json:"fulfillment_status" gorm:"default:'pending'"`
	IsFulfilled        bool       `json:"is_fulfilled"      gorm:"default:false"`
	PaidAt             *time.Time `json:"paid_at"`
	FulfilledAt        *time.Time `json:"fulfilled_at"`
	ExpectedDeliveryAt *time.Time `json:"expected_delivery_at"`
	DeliveredItems     string     `json:"delivered_items" gorm:"type:text"`
	Notes              string     `json:"notes"`

	// Payment Gateway fields
	GatewayChargeID     string     `json:"gateway_charge_id"   gorm:"default:''"` // ID charge/invoice di sisi gateway
	GatewayInvoiceNo    string     `json:"gateway_invoice_no"  gorm:"default:''"` // Nomor invoice dari gateway (untuk sync)
	GatewayProvider     string     `json:"gateway_provider"    gorm:"default:''"` // "sayabayar" | "dompetx" | ""
	GatewayPayURL       string     `json:"gateway_pay_url"     gorm:"default:''"`
	GatewayRedirectURL  string     `json:"gateway_redirect_url" gorm:"default:''"`
	GatewayPayCode      string     `json:"gateway_pay_code"    gorm:"default:''"`
	GatewayQrisString   string     `json:"gateway_qris_string" gorm:"type:text"`
	GatewayQrisImageURL string     `json:"gateway_qris_image_url" gorm:"default:''"`
	ExpiredAt           *time.Time `json:"expired_at"`

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
	SayaBayarEnabled     bool   `json:"sayabayar_enabled"      gorm:"default:false"`
	SayaBayarAPIKey      string `json:"sayabayar_api_key"      gorm:"default:''"`
	SayaBayarAutoQRIS    bool   `json:"sayabayar_auto_qris"    gorm:"default:true"`
	SayaBayarAutoConfirm bool   `json:"sayabayar_auto_confirm" gorm:"default:true"`
	// channel_preference: "platform" (default) | "client" (dana langsung ke rekening)
	SayaBayarChannel string `json:"sayabayar_channel"          gorm:"default:'platform'"`
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
	ID        uint   `json:"id"           gorm:"primaryKey;autoIncrement"`
	Name      string `json:"name"         gorm:"not null"`       // Nama provider
	ProductID uint   `json:"product_id"   gorm:"not null;index"` // Produk tujuan
	Type      string `json:"type"         gorm:"not null"`       // "http_api" | "csv_url"
	// HTTP API config
	APIURL     string `json:"api_url"      gorm:"type:text"`     // URL endpoint provider
	APIMethod  string `json:"api_method"   gorm:"default:'GET'"` // GET | POST
	APIHeaders string `json:"api_headers"  gorm:"type:text"`     // JSON: {"X-Key":"val"}
	APIBody    string `json:"api_body"     gorm:"type:text"`     // JSON body jika POST
	// Response parsing
	ItemsPath string `json:"items_path"` // JSONPath ke array, mis: "data.items"
	ItemField string `json:"item_field"` // Field tiap item, mis: "key" atau "url"
	// Status
	Active     bool       `json:"active"       gorm:"default:true"`
	LastPullAt *time.Time `json:"last_pull_at"`
	LastCount  int        `json:"last_count"` // Jumlah item terakhir di-pull
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// PullLog — log setiap kali pull dilakukan
type PullLog struct {
	ID         uint      `json:"id"          gorm:"primaryKey;autoIncrement"`
	ProviderID uint      `json:"provider_id" gorm:"index"`
	ProductID  uint      `json:"product_id"`
	Status     string    `json:"status"` // success | failed | partial
	Count      int       `json:"count"`  // item berhasil ditambahkan
	Message    string    `json:"message"     gorm:"type:text"`
	CreatedAt  time.Time `json:"created_at"`
}

// ── ExternalProvider ──────────────────────────────────────────────────────────
// Konfigurasi koneksi ke provider eksternal seperti KoalaStore.
// Satu provider bisa menyuplai banyak produk.

type ExternalProvider struct {
	ID          uint   `json:"id"           gorm:"primaryKey;autoIncrement"`
	Name        string `json:"name"         gorm:"not null"`   // "KoalaStore", "DigiFlazz", dll
	Type        string `json:"type"         gorm:"not null"`   // "koalastore" | "digiflazz" | "generic"
	BaseURL     string `json:"base_url"     gorm:"not null"`   // https://koalastore.digital/api
	APIKey      string `json:"api_key"      gorm:"type:text"`  // API key / token
	Username    string `json:"username"     gorm:"default:''"` // jika butuh username
	ExtraConfig string `json:"extra_config" gorm:"type:text"`  // JSON config tambahan
	Active      bool   `json:"active"       gorm:"default:true"`
	// Cache daftar produk dari provider
	LastSyncAt *time.Time `json:"last_sync_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// ProviderProduct — cache produk yang tersedia di provider
// Diisi saat sync, admin memilih mana yang mau dijual
type ProviderProduct struct {
	ID            uint   `json:"id"              gorm:"primaryKey;autoIncrement"`
	ProviderID    uint   `json:"provider_id"     gorm:"not null;index"`
	ProviderName  string `json:"provider_name"`
	Code          string `json:"code"            gorm:"not null"` // code_variant dari KoalaStore
	Name          string `json:"name"            gorm:"not null"`
	Category      string `json:"category"`
	ProviderPrice int64  `json:"provider_price"`
	// Stock status dari KoalaStore: "available" | "out_of_stock" | "manual"
	Stock string `json:"stock"           gorm:"default:'unknown'"`
	// Jumlah stok real dari KoalaStore (field available_stock dari response API)
	AvailableStock int       `json:"available_stock" gorm:"default:0"`
	IsManual       bool      `json:"is_manual"       gorm:"default:false"` // is_manual_process
	Description    string    `json:"description"     gorm:"type:text"`
	ProductID      *uint     `json:"product_id"`
	Imported       bool      `json:"imported"        gorm:"default:false"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ProviderOrder — log pemesanan ke provider saat ada order masuk
type ProviderOrder struct {
	ID              uint      `json:"id"               gorm:"primaryKey;autoIncrement"`
	OrderID         uint      `json:"order_id"         gorm:"index"`
	InvoiceNo       string    `json:"invoice_no"`
	ProviderID      uint      `json:"provider_id"`
	ProviderCode    string    `json:"provider_code"`                     // kode produk di provider
	ProviderOrderID string    `json:"provider_order_id"`                 // ID order dari provider
	Status          string    `json:"status"`                            // pending | success | failed
	Serial          string    `json:"serial"           gorm:"type:text"` // nomor seri / hasil dari provider
	Message         string    `json:"message"          gorm:"type:text"`
	PricePaid       int64     `json:"price_paid"` // harga yang dibayar ke provider
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ── ContactConfig ─────────────────────────────────────────────────────────────
// Konfigurasi kontak dan sosial media yang ditampilkan di toko dan halaman komplain.

type ContactConfig struct {
	ID            uint   `json:"id"               gorm:"primaryKey"`
	WhatsApp      string `json:"whatsapp"         gorm:"default:''"` // nomor WA, mis: 6281234567890
	WhatsAppLabel string `json:"whatsapp_label"   gorm:"default:'Hubungi CS'"`
	Telegram      string `json:"telegram"         gorm:"default:''"` // username atau link
	Instagram     string `json:"instagram"        gorm:"default:''"` // @username
	Email         string `json:"email"            gorm:"default:''"` // email support
	Website       string `json:"website"          gorm:"default:''"`
	BusinessName  string `json:"business_name"    gorm:"default:'DigiStore'"`
	BusinessDesc  string `json:"business_desc"    gorm:"type:text;default:''"`
	// Pesan template untuk WA/Telegram
	ComplaintTemplate string `json:"complaint_template" gorm:"type:text;default:''"`
	OperationalHours  string `json:"operational_hours"  gorm:"default:'Senin - Sabtu, 08.00 - 21.00 WIB'"`
}
