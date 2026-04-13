package handlers

import (
	"bytes"
	"digistore/database"
	"digistore/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// GET /api/admin/providers
func GetProviders(c *gin.Context) {
	var providers []models.StockProvider
	query := database.DB.Order("created_at desc")
	if pid := c.Query("product_id"); pid != "" {
		query = query.Where("product_id = ?", pid)
	}
	query.Find(&providers)
	c.JSON(http.StatusOK, providers)
}

// POST /api/admin/providers
func CreateProvider(c *gin.Context) {
	var p models.StockProvider
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p.ID = 0
	if err := database.DB.Create(&p).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan provider"})
		return
	}
	c.JSON(http.StatusCreated, p)
}

// PUT /api/admin/providers/:id
func UpdateProvider(c *gin.Context) {
	var p models.StockProvider
	if err := database.DB.First(&p, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "provider tidak ditemukan"})
		return
	}
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	database.DB.Save(&p)
	c.JSON(http.StatusOK, p)
}

// DELETE /api/admin/providers/:id
func DeleteProvider(c *gin.Context) {
	database.DB.Delete(&models.StockProvider{}, c.Param("id"))
	c.JSON(http.StatusOK, gin.H{"message": "provider dihapus"})
}

// POST /api/admin/providers/:id/pull — tarik stok dari provider sekarang
func PullFromProvider(c *gin.Context) {
	var provider models.StockProvider
	if err := database.DB.First(&provider, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "provider tidak ditemukan"})
		return
	}

	count, msg, status := executePull(&provider)

	// Simpan log
	log := models.PullLog{
		ProviderID: provider.ID,
		ProductID:  provider.ProductID,
		Status:     status,
		Count:      count,
		Message:    msg,
	}
	database.DB.Create(&log)

	// Update last pull
	now := time.Now()
	database.DB.Model(&provider).Updates(map[string]interface{}{
		"last_pull_at": now,
		"last_count":   count,
	})

	c.JSON(http.StatusOK, gin.H{
		"status":  status,
		"count":   count,
		"message": msg,
	})
}

// GET /api/admin/providers/:id/logs — riwayat pull
func GetProviderLogs(c *gin.Context) {
	var logs []models.PullLog
	database.DB.Where("provider_id = ?", c.Param("id")).
		Order("created_at desc").Limit(50).Find(&logs)
	c.JSON(http.StatusOK, logs)
}

// GET /api/admin/pull-logs — semua log pull
func GetAllPullLogs(c *gin.Context) {
	var logs []models.PullLog
	database.DB.Order("created_at desc").Limit(100).Find(&logs)
	c.JSON(http.StatusOK, logs)
}

// ── Pull execution ─────────────────────────────────────────────────────────────

func executePull(provider *models.StockProvider) (count int, msg string, status string) {
	switch provider.Type {
	case "http_api":
		return pullFromHTTP(provider)
	case "csv_url":
		return pullFromCSV(provider)
	default:
		return 0, fmt.Sprintf("tipe provider tidak dikenal: %s", provider.Type), "failed"
	}
}

// pullFromHTTP melakukan request HTTP ke API provider dan mengekstrak item stok
func pullFromHTTP(p *models.StockProvider) (int, string, string) {
	// Build request
	method := p.APIMethod
	if method == "" { method = "GET" }

	var bodyReader io.Reader
	if p.APIBody != "" {
		bodyReader = bytes.NewBufferString(p.APIBody)
	}

	req, err := http.NewRequest(method, p.APIURL, bodyReader)
	if err != nil {
		return 0, "gagal buat request: " + err.Error(), "failed"
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Parse dan set custom headers
	if p.APIHeaders != "" {
		var headers map[string]string
		if err := json.Unmarshal([]byte(p.APIHeaders), &headers); err == nil {
			for k, v := range headers {
				req.Header.Set(k, v)
			}
		}
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "request gagal: " + err.Error(), "failed"
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)[:min(200, len(body))]), "failed"
	}

	body, _ := io.ReadAll(resp.Body)

	// Parse JSON response
	var rawJSON interface{}
	if err := json.Unmarshal(body, &rawJSON); err != nil {
		return 0, "response bukan JSON valid: " + err.Error(), "failed"
	}

	// Ekstrak array item menggunakan items_path
	items, err := extractItems(rawJSON, p.ItemsPath, p.ItemField)
	if err != nil {
		return 0, "gagal ekstrak items: " + err.Error(), "failed"
	}

	return saveItems(p.ProductID, items)
}

// pullFromCSV mengambil file CSV dari URL — satu item per baris
func pullFromCSV(p *models.StockProvider) (int, string, string) {
	resp, err := http.Get(p.APIURL)
	if err != nil {
		return 0, "gagal download CSV: " + err.Error(), "failed"
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	lines := strings.Split(string(body), "\n")

	var items []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			items = append(items, line)
		}
	}

	return saveItems(p.ProductID, items)
}

// extractItems mengekstrak item dari JSON menggunakan dot-notation path
// Contoh: items_path = "data.items", item_field = "key"
func extractItems(data interface{}, path, field string) ([]string, error) {
	// Navigasi ke path
	current := data
	if path != "" {
		for _, key := range strings.Split(path, ".") {
			m, ok := current.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("path '%s' tidak ditemukan di response", path)
			}
			current = m[key]
		}
	}

	// Pastikan hasilnya array
	arr, ok := current.([]interface{})
	if !ok {
		return nil, fmt.Errorf("path '%s' bukan array", path)
	}

	var items []string
	for _, el := range arr {
		var val string
		switch v := el.(type) {
		case string:
			val = v // item adalah string langsung
		case map[string]interface{}:
			if field == "" {
				// Serialisasi seluruh object sebagai JSON
				b, _ := json.Marshal(v)
				val = string(b)
			} else if fv, ok := v[field]; ok {
				val = fmt.Sprintf("%v", fv)
			}
		default:
			val = fmt.Sprintf("%v", v)
		}
		if val != "" {
			items = append(items, val)
		}
	}
	return items, nil
}

// saveItems menyimpan item ke tabel product_stocks, skip duplikat
func saveItems(productID uint, items []string) (int, string, string) {
	if len(items) == 0 {
		return 0, "tidak ada item ditemukan di response", "failed"
	}

	// Ambil data yang sudah ada untuk skip duplikat
	var existing []models.ProductStock
	database.DB.Where("product_id = ?", productID).Find(&existing)
	existingSet := make(map[string]bool)
	for _, e := range existing {
		existingSet[e.Data] = true
	}

	added := 0
	skipped := 0
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" { continue }
		if existingSet[item] {
			skipped++
			continue
		}
		database.DB.Create(&models.ProductStock{ProductID: productID, Data: item})
		existingSet[item] = true
		added++
	}

	status := "success"
	if added == 0 && skipped > 0 { status = "partial" }
	msg := fmt.Sprintf("%d item ditambahkan, %d duplikat dilewati", added, skipped)
	return added, msg, status
}

func min(a, b int) int {
	if a < b { return a }
	return b
}
