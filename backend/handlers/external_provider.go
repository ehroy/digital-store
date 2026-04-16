package handlers

import (
	"digistore/database"
	"digistore/gateway"
	"digistore/models"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ── Provider CRUD ─────────────────────────────────────────────────────────────

func GetExternalProviders(c *gin.Context) {
	var providers []models.ExternalProvider
	database.DB.Order("created_at desc").Find(&providers)
	// Sensor API key sebelum return
	for i := range providers {
		if len(providers[i].APIKey) > 8 {
			providers[i].APIKey = providers[i].APIKey[:4] + "****" + providers[i].APIKey[len(providers[i].APIKey)-4:]
		}
	}
	c.JSON(http.StatusOK, providers)
}

func CreateExternalProvider(c *gin.Context) {
	var p models.ExternalProvider
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p.ID = 0
	database.DB.Create(&p)
	c.JSON(http.StatusCreated, p)
}

func UpdateExternalProvider(c *gin.Context) {
	var p models.ExternalProvider
	if err := database.DB.First(&p, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "provider tidak ditemukan"})
		return
	}
	var body map[string]interface{}
	c.ShouldBindJSON(&body)
	// Jangan overwrite API key jika dikirim dengan mask
	if apiKey, ok := body["api_key"].(string); ok && len(apiKey) > 0 && !containsAsterisks(apiKey) {
		p.APIKey = apiKey
	}
	if name, ok := body["name"].(string); ok {
		p.Name = name
	}
	if active, ok := body["active"].(bool); ok {
		p.Active = active
	}
	if ec, ok := body["extra_config"].(string); ok {
		p.ExtraConfig = ec
	}
	database.DB.Save(&p)
	c.JSON(http.StatusOK, p)
}

func DeleteExternalProvider(c *gin.Context) {
	database.DB.Delete(&models.ExternalProvider{}, c.Param("id"))
	database.DB.Where("provider_id = ?", c.Param("id")).Delete(&models.ProviderProduct{})
	c.JSON(http.StatusOK, gin.H{"message": "provider dihapus"})
}

// ── Sync produk dari KoalaStore ───────────────────────────────────────────────

// POST /api/admin/external-providers/:id/sync
func SyncProviderProducts(c *gin.Context) {
	var provider models.ExternalProvider
	if err := database.DB.First(&provider, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "provider tidak ditemukan"})
		return
	}

	ks := gateway.NewKoalaStore(provider.APIKey)
	products, err := ks.GetAllProducts()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "gagal sync: " + err.Error()})
		return
	}

	// Flatten: satu baris per variant
	added, updated := 0, 0
	for _, p := range products {
		for _, v := range p.Variants {
			var existing models.ProviderProduct
			err := database.DB.Where("provider_id = ? AND code = ?", provider.ID, v.CodeVariant).First(&existing).Error

			stock := "available"
			if v.AvailableStock == 0 {
				stock = "out_of_stock"
			}
			if v.IsManualProcess {
				stock = "manual"
			}

			pp := models.ProviderProduct{
				ProviderID:     provider.ID,
				ProviderName:   provider.Name,
				Code:           v.CodeVariant,
				Name:           normalizeKoalaStoreProductName(p.Name, v.Name),
				Category:       p.Category,
				ProviderPrice:  v.Price,
				Stock:          stock,
				AvailableStock: v.AvailableStock,
				Description:    p.Description,
				UpdatedAt:      time.Now(),
			}

			if err != nil {
				// Baru
				database.DB.Create(&pp)
				added++
			} else {
				// Update harga & stok
				database.DB.Model(&existing).Updates(map[string]interface{}{
					"provider_price":  v.Price,
					"stock":           stock,
					"available_stock": v.AvailableStock,
					"name":            pp.Name,
					"updated_at":      time.Now(),
				})
				updated++
			}
		}
	}

	now := time.Now()
	database.DB.Model(&provider).Update("last_sync_at", now)

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Sync selesai: %d produk baru, %d diperbarui", added, updated),
		"added":   added,
		"updated": updated,
		"total":   added + updated,
	})
}

// GET /api/admin/external-providers/:id/products
func GetProviderProducts(c *gin.Context) {
	var items []models.ProviderProduct
	query := database.DB.Where("provider_id = ?", c.Param("id")).Order("category, name")

	if cat := c.Query("category"); cat != "" {
		query = query.Where("category = ?", cat)
	}
	if s := c.Query("search"); s != "" {
		query = query.Where("name LIKE ?", "%"+s+"%")
	}
	if imported := c.Query("imported"); imported == "true" {
		query = query.Where("imported = ?", true)
	}
	if imported := c.Query("imported"); imported == "false" {
		query = query.Where("imported = ?", false)
	}

	// Pagination
	page, pageSize := 1, 30
	fmt.Sscanf(c.Query("page"), "%d", &page)
	if page < 1 {
		page = 1
	}

	var total int64
	query.Model(&models.ProviderProduct{}).Count(&total)
	query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&items)

	c.JSON(http.StatusOK, gin.H{
		"items":       items,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": int(math.Ceil(float64(total) / float64(pageSize))),
	})
}

// GET /api/admin/external-providers/:id/balance
func GetProviderBalance(c *gin.Context) {
	var provider models.ExternalProvider
	if err := database.DB.First(&provider, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "provider tidak ditemukan"})
		return
	}
	ks := gateway.NewKoalaStore(provider.APIKey)
	bal, err := ks.GetBalance()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, bal.Data)
}

// POST /api/admin/external-providers/:id/import
// Import satu atau banyak ProviderProduct menjadi Product di DigiStore
// Body: { "codes": ["MLBB-86DM", "MLBB-172DM"], "markup_type": "percent", "markup_value": 15 }
func ImportProviderProducts(c *gin.Context) {
	var body struct {
		Codes       []string `json:"codes"        binding:"required,min=1"`
		MarkupType  string   `json:"markup_type"` // "percent" | "fixed"
		MarkupValue float64  `json:"markup_value"`
		AutoSync    bool     `json:"auto_sync"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.MarkupType == "" {
		body.MarkupType = "percent"
	}

	var provider models.ExternalProvider
	if err := database.DB.First(&provider, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "provider tidak ditemukan"})
		return
	}

	var imported []models.Product
	for _, code := range body.Codes {
		var pp models.ProviderProduct
		if err := database.DB.Where("provider_id = ? AND code = ?", provider.ID, code).First(&pp).Error; err != nil {
			continue
		}

		// Hitung harga jual
		sellPrice := calcSellPrice(pp.ProviderPrice, body.MarkupType, body.MarkupValue)

		product := models.Product{
			Name:          pp.Name,
			Description:   pp.Description,
			Price:         sellPrice,
			Category:      pp.Category,
			Type:          "provider",
			Icon:          "🎮",
			Active:        true,
			ProviderName:  provider.Name,
			ProviderCode:  pp.Code,
			ProviderPrice: pp.ProviderPrice,
			MarkupType:    body.MarkupType,
			MarkupValue:   body.MarkupValue,
			AutoSync:      body.AutoSync,
		}
		database.DB.Create(&product)
		imported = append(imported, product)

		// Tandai sebagai imported
		database.DB.Model(&pp).Updates(map[string]interface{}{
			"imported":   true,
			"product_id": product.ID,
		})
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  fmt.Sprintf("%d produk berhasil diimport", len(imported)),
		"products": imported,
	})
}

// POST /api/admin/external-providers/sync-prices — update harga semua produk provider
func SyncProviderPrices(c *gin.Context) {
	// Ambil semua produk tipe provider dengan auto_sync = true
	var products []models.Product
	database.DB.Where("type = ? AND auto_sync = ?", "provider", true).Find(&products)

	updated := 0
	for _, p := range products {
		// Cari harga terbaru dari cache ProviderProduct
		var pp models.ProviderProduct
		if err := database.DB.Where("provider_name = ? AND code = ?", p.ProviderName, p.ProviderCode).First(&pp).Error; err != nil {
			continue
		}
		newPrice := calcSellPrice(pp.ProviderPrice, p.MarkupType, p.MarkupValue)
		if newPrice != p.Price {
			database.DB.Model(&p).Updates(map[string]interface{}{
				"price":          newPrice,
				"provider_price": pp.ProviderPrice,
			})
			updated++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("%d harga produk diperbarui", updated),
		"updated": updated,
	})
}

// ── Order ke KoalaStore saat ada pembelian ────────────────────────────────────

// OrderFromKoalaStore dipanggil dari orders.go saat order produk tipe "provider"
// Langsung checkout ke KoalaStore dan kembalikan stock_data sebagai delivered items
func OrderFromKoalaStore(order *models.Order, product *models.Product) ([]string, error) {
	// Cari provider
	var provider models.ExternalProvider
	if err := database.DB.Where("name = ? AND active = ?", product.ProviderName, true).First(&provider).Error; err != nil {
		return nil, fmt.Errorf("provider '%s' tidak aktif atau tidak ditemukan", product.ProviderName)
	}

	ks := gateway.NewKoalaStore(provider.APIKey)

	// Checkout ke KoalaStore
	resp, err := ks.Checkout(product.ProviderCode, order.Qty)
	if err != nil {
		// Simpan log kegagalan
		database.DB.Create(&models.ProviderOrder{
			OrderID:      order.ID,
			InvoiceNo:    order.InvoiceNo,
			ProviderID:   provider.ID,
			ProviderCode: product.ProviderCode,
			Status:       "failed",
			Message:      err.Error(),
			PricePaid:    product.ProviderPrice * int64(order.Qty),
		})
		return nil, err
	}

	// Kumpulkan stock_data dari semua items
	var deliveredItems []string
	var totalPaid int64
	for _, item := range resp.Data.Items {
		totalPaid += item.Subtotal
		for _, sd := range item.StockData {
			if sd.DataStock != "" {
				deliveredItems = append(deliveredItems, sd.DataStock)
			}
		}
	}

	// Simpan log sukses
	stockJSON, _ := json.Marshal(deliveredItems)
	database.DB.Create(&models.ProviderOrder{
		OrderID:         order.ID,
		InvoiceNo:       order.InvoiceNo,
		ProviderID:      provider.ID,
		ProviderCode:    product.ProviderCode,
		ProviderOrderID: resp.Data.TransactionID,
		Status:          "success",
		Serial:          string(stockJSON),
		Message:         resp.Message,
		PricePaid:       totalPaid,
	})

	log.Printf("[KOALASTORE] Order berhasil: invoice=%s tx=%s items=%d",
		order.InvoiceNo, resp.Data.TransactionID, len(deliveredItems))

	return deliveredItems, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func calcSellPrice(basePrice int64, markupType string, markupValue float64) int64 {
	if markupValue <= 0 {
		return basePrice
	}
	switch markupType {
	case "percent":
		return basePrice + int64(math.Round(float64(basePrice)*markupValue/100))
	case "fixed":
		return basePrice + int64(markupValue)
	}
	return basePrice
}

func containsAsterisks(s string) bool {
	for _, c := range s {
		if c == '*' {
			return true
		}
	}
	return false
}

func normalizeKoalaStoreProductName(productName, variantName string) string {
	productName = strings.TrimSpace(productName)
	variantName = strings.TrimSpace(variantName)
	if productName == "" {
		return variantName
	}
	if variantName == "" {
		return productName
	}
	return productName + " — " + variantName
}

// ── Auto Stock Sync Job ───────────────────────────────────────────────────────
// Berjalan di background setiap 15 menit — sync stok semua provider aktif
// Mencegah pembeli order produk yang stoknya habis di KoalaStore

func StartProviderSyncJob() {
	go func() {
		// Delay 30 detik setelah startup baru mulai
		time.Sleep(30 * time.Second)
		ticker := time.NewTicker(120 * time.Minute)
		defer ticker.Stop()
		log.Println("🔄 Provider stock sync job dimulai — interval 15 menit")
		// Sync pertama kali
		autoSyncAllProviders()
		for range ticker.C {
			autoSyncAllProviders()
		}
	}()
}

func autoSyncAllProviders() {
	var providers []models.ExternalProvider
	database.DB.Where("active = ?", true).Find(&providers)
	for _, p := range providers {
		if p.Type != "koalastore" && p.Type != "" {
			continue
		}
		ks := gateway.NewKoalaStore(p.APIKey)
		products, err := ks.GetAllProducts()
		if err != nil {
			log.Printf("[SYNC JOB] Gagal sync provider %s: %v", p.Name, err)
			continue
		}

		updated := 0
		for _, prod := range products {
			for _, v := range prod.Variants {
				stock := "available"
				if v.AvailableStock == 0 {
					stock = "out_of_stock"
				}
				if v.IsManualProcess {
					stock = "manual"
				}

				res := database.DB.Model(&models.ProviderProduct{}).
					Where("provider_id = ? AND code = ?", p.ID, v.CodeVariant).
					Updates(map[string]interface{}{
						"provider_price":  v.Price,
						"stock":           stock,
						"available_stock": v.AvailableStock,
						"updated_at":      time.Now(),
					})
				if res.RowsAffected > 0 {
					updated++
				}
			}
		}

		// Update harga produk DigiStore yang auto_sync = true
		var autoSyncProducts []models.Product
		database.DB.Where("type = ? AND provider_name = ? AND auto_sync = ?", "provider", p.Name, true).
			Find(&autoSyncProducts)
		for _, prod := range autoSyncProducts {
			var pp models.ProviderProduct
			if err := database.DB.Where("provider_name = ? AND code = ?", prod.ProviderName, prod.ProviderCode).First(&pp).Error; err != nil {
				continue
			}
			newPrice := calcSellPrice(pp.ProviderPrice, prod.MarkupType, prod.MarkupValue)
			if newPrice != prod.Price {
				database.DB.Model(&prod).Updates(map[string]interface{}{
					"price":          newPrice,
					"provider_price": pp.ProviderPrice,
				})
			}
		}

		now := time.Now()
		database.DB.Model(&p).Update("last_sync_at", now)
		log.Printf("[SYNC JOB] Provider %s: %d items diperbarui", p.Name, updated)
	}
}
