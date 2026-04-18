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
	var body struct {
		Name               string  `json:"name"`
		Type               string  `json:"type"`
		BaseURL            string  `json:"base_url"`
		APIKey             string  `json:"api_key"`
		Username           string  `json:"username"`
		DefaultMarkupType  string  `json:"default_markup_type"`
		DefaultMarkupValue float64 `json:"default_markup_value"`
		ExtraConfig        string  `json:"extra_config"`
		Active             bool    `json:"active"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.DefaultMarkupType == "" {
		body.DefaultMarkupType = "percent"
	}
	if body.DefaultMarkupValue < 0 {
		body.DefaultMarkupValue = 15
	}
	p := models.ExternalProvider{
		Name:               body.Name,
		Type:               body.Type,
		BaseURL:            body.BaseURL,
		APIKey:             body.APIKey,
		Username:           body.Username,
		DefaultMarkupType:  body.DefaultMarkupType,
		DefaultMarkupValue: body.DefaultMarkupValue,
		ExtraConfig:        body.ExtraConfig,
		Active:             body.Active,
	}
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
	if markupType, ok := body["default_markup_type"].(string); ok && strings.TrimSpace(markupType) != "" {
		p.DefaultMarkupType = markupType
	}
	if markupValue, ok := body["default_markup_value"].(float64); ok {
		p.DefaultMarkupValue = markupValue
	}
	if ec, ok := body["extra_config"].(string); ok {
		p.ExtraConfig = ec
	}
	database.DB.Save(&p)
	if _, hasMarkupType := body["default_markup_type"]; hasMarkupType {
		applyProviderDefaultMarkupToProducts(&p)
	} else if _, hasMarkupValue := body["default_markup_value"]; hasMarkupValue {
		applyProviderDefaultMarkupToProducts(&p)
	}
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
				ProviderID:         provider.ID,
				ProviderName:       provider.Name,
				Code:               v.CodeVariant,
				Name:               normalizeKoalaStoreProductName(p.Name, v.Name),
				Category:           p.Category,
				ProviderPrice:      v.Price,
				Stock:              stock,
				WarrantyTerms:      v.WarrantyTerms,
				TermsAndConditions: v.TermsAndConditions,
				AvailableStock:     v.AvailableStock,
				IsManual:           v.IsManualProcess,
				Description:        p.Description,
				UpdatedAt:          time.Now(),
			}

			if err != nil {
				// Baru
				database.DB.Create(&pp)
				added++
			} else {
				// Update harga & stok
				database.DB.Model(&existing).Updates(map[string]interface{}{
					"provider_price":       v.Price,
					"stock":                stock,
					"warranty_terms":       v.WarrantyTerms,
					"terms_and_conditions": v.TermsAndConditions,
					"available_stock":      v.AvailableStock,
					"is_manual":            v.IsManualProcess,
					"name":                 pp.Name,
					"updated_at":           time.Now(),
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
			Name:                     pp.Name,
			Description:              pp.Description,
			Price:                    sellPrice,
			Category:                 pp.Category,
			Type:                     "provider",
			Icon:                     "🎮",
			Active:                   true,
			ProviderName:             provider.Name,
			ProviderCode:             pp.Code,
			ProviderPrice:            pp.ProviderPrice,
			WarrantyTerms:            pp.WarrantyTerms,
			TermsAndConditions:       pp.TermsAndConditions,
			MarkupType:               body.MarkupType,
			MarkupValue:              body.MarkupValue,
			UseProviderDefaultMarkup: true,
			AutoSync:                 body.AutoSync,
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
	// Manual sync: update semua produk provider, tanpa tergantung auto_sync.
	var products []models.Product
	database.DB.Where("type = ?", "provider").Find(&products)

	updated := 0
	for _, p := range products {
		// Cari harga terbaru dari cache ProviderProduct
		var pp models.ProviderProduct
		if err := database.DB.Where("provider_name = ? AND code = ?", p.ProviderName, p.ProviderCode).First(&pp).Error; err != nil {
			continue
		}
		var provider models.ExternalProvider
		if err := database.DB.Where("name = ?", p.ProviderName).First(&provider).Error; err != nil {
			continue
		}
		markupType := provider.DefaultMarkupType
		if strings.TrimSpace(markupType) == "" {
			markupType = "percent"
		}
		markupValue := provider.DefaultMarkupValue
		if markupValue < 0 {
			markupValue = 15
		}
		newPrice := calcSellPrice(pp.ProviderPrice, markupType, markupValue)
		database.DB.Model(&p).Updates(map[string]interface{}{
			"price":                       newPrice,
			"provider_price":              pp.ProviderPrice,
			"warranty_terms":              pp.WarrantyTerms,
			"terms_and_conditions":        pp.TermsAndConditions,
			"markup_type":                 markupType,
			"markup_value":                markupValue,
			"use_provider_default_markup": true,
		})
		if newPrice != p.Price || p.MarkupType != markupType || p.MarkupValue != markupValue || !p.UseProviderDefaultMarkup {
			updated++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("%d harga produk diperbarui", updated),
		"updated": updated,
	})
}

// POST /api/admin/external-providers/:id/apply-default-markup
func ApplyProviderDefaultMarkup(c *gin.Context) {
	var provider models.ExternalProvider
	if err := database.DB.First(&provider, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "provider tidak ditemukan"})
		return
	}
	updated := applyProviderDefaultMarkupToProducts(&provider)

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("%d produk mengikuti default markup provider", updated),
		"updated": updated,
	})
}

func applyProviderDefaultMarkupToProducts(provider *models.ExternalProvider) int {
	markupType := provider.DefaultMarkupType
	if strings.TrimSpace(markupType) == "" {
		markupType = "percent"
	}
	markupValue := provider.DefaultMarkupValue
	if markupValue < 0 {
		markupValue = 15
	}

	var products []models.Product
	database.DB.Where("type = ? AND provider_name = ? AND use_provider_default_markup = ?", "provider", provider.Name, true).Find(&products)

	updated := 0
	for _, p := range products {
		var pp models.ProviderProduct
		if err := database.DB.Where("provider_name = ? AND code = ?", p.ProviderName, p.ProviderCode).First(&pp).Error; err != nil {
			continue
		}
		newPrice := calcSellPrice(pp.ProviderPrice, markupType, markupValue)
		database.DB.Model(&p).Updates(map[string]interface{}{
			"markup_type":          markupType,
			"markup_value":         markupValue,
			"price":                newPrice,
			"provider_price":       pp.ProviderPrice,
			"warranty_terms":       pp.WarrantyTerms,
			"terms_and_conditions": pp.TermsAndConditions,
		})
		updated++
	}
	return updated
}

// ── Order ke KoalaStore saat ada pembelian ────────────────────────────────────

// OrderFromKoalaStore dipanggil dari orders.go saat order produk tipe "provider"
// Langsung checkout ke KoalaStore dan kembalikan stock_data sebagai delivered items
func OrderFromKoalaStore(order *models.Order, product *models.Product) ([]string, error) {
	// Cari provider
	var provider models.ExternalProvider
	if err := database.DB.Where("name = ? AND active = ?", product.ProviderName, true).First(&provider).Error; err != nil {
		if err := database.DB.Where("name = ?", product.ProviderName).First(&provider).Error; err != nil {
			return nil, fmt.Errorf("provider '%s' tidak ditemukan", product.ProviderName)
		}
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
	if markupValue < 0 {
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

func effectiveMarkupForProduct(product models.Product, provider *models.ExternalProvider) (string, float64) {
	if product.UseProviderDefaultMarkup && provider != nil {
		markupType := strings.TrimSpace(provider.DefaultMarkupType)
		if markupType == "" {
			markupType = "percent"
		}
		markupValue := provider.DefaultMarkupValue
		if markupValue < 0 {
			markupValue = 15
		}
		return markupType, markupValue
	}

	markupType := strings.TrimSpace(product.MarkupType)
	if markupType == "" {
		markupType = "percent"
	}
	markupValue := product.MarkupValue
	if markupValue < 0 {
		markupValue = 15
	}
	return markupType, markupValue
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
// Berjalan di background setiap 30 menit — sync stok semua provider aktif
// Mencegah pembeli order produk yang stoknya habis di KoalaStore

func StartProviderSyncJob() {
	go func() {
		// Delay 30 detik setelah startup baru mulai
		time.Sleep(30 * time.Second)
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		log.Println("🔄 Provider stock sync job dimulai — interval 30 menit")
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
						"provider_price":       v.Price,
						"stock":                stock,
						"warranty_terms":       v.WarrantyTerms,
						"terms_and_conditions": v.TermsAndConditions,
						"available_stock":      v.AvailableStock,
						"is_manual":            v.IsManualProcess,
						"updated_at":           time.Now(),
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
			markupType, markupValue := effectiveMarkupForProduct(prod, &p)
			newPrice := calcSellPrice(pp.ProviderPrice, markupType, markupValue)
			if newPrice != prod.Price {
				database.DB.Model(&prod).Updates(map[string]interface{}{
					"price":                newPrice,
					"provider_price":       pp.ProviderPrice,
					"warranty_terms":       pp.WarrantyTerms,
					"terms_and_conditions": pp.TermsAndConditions,
				})
			}
		}

		now := time.Now()
		database.DB.Model(&p).Update("last_sync_at", now)
		log.Printf("[SYNC JOB] Provider %s: %d items diperbarui", p.Name, updated)
	}
}
