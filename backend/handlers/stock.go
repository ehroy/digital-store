package handlers

import (
	"digistore/database"
	"digistore/models"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GET /api/admin/products/:id/stock — semua item stok produk
func GetProductStock(c *gin.Context) {
	var items []models.ProductStock
	database.DB.
		Where("product_id = ?", c.Param("id")).
		Order("sold asc, id asc").
		Find(&items)

	var avail, sold int
	for _, it := range items {
		if it.Sold { sold++ } else { avail++ }
	}

	c.JSON(http.StatusOK, gin.H{
		"items":     items,
		"available": avail,
		"sold":      sold,
		"total":     len(items),
	})
}

// POST /api/admin/products/:id/stock — tambah item stok (bulk)
// Body: { "items": ["key1", "key2", "https://..."] }
func AddProductStock(c *gin.Context) {
	var body struct {
		Items []string `json:"items" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "items wajib diisi (array string)"})
		return
	}

	var product models.Product
	if err := database.DB.First(&product, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "produk tidak ditemukan"})
		return
	}

	var created []models.ProductStock
	for _, data := range body.Items {
		if data == "" { continue }
		item := models.ProductStock{ProductID: product.ID, Data: data}
		database.DB.Create(&item)
		created = append(created, item)
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": fmt.Sprintf("%d item berhasil ditambahkan", len(created)),
		"items":   created,
	})
}

// PUT /api/admin/stock/:stockId — edit data item (hanya belum terjual)
func UpdateStockItem(c *gin.Context) {
	var item models.ProductStock
	if err := database.DB.First(&item, c.Param("stockId")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "item tidak ditemukan"})
		return
	}
	if item.Sold {
		c.JSON(http.StatusBadRequest, gin.H{"error": "item sudah terjual, tidak dapat diubah"})
		return
	}
	var body struct {
		Data string `json:"data" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item.Data = body.Data
	database.DB.Save(&item)
	c.JSON(http.StatusOK, item)
}

// DELETE /api/admin/stock/:stockId — hapus item (hanya belum terjual)
func DeleteStockItem(c *gin.Context) {
	var item models.ProductStock
	if err := database.DB.First(&item, c.Param("stockId")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "item tidak ditemukan"})
		return
	}
	if item.Sold {
		c.JSON(http.StatusBadRequest, gin.H{"error": "item sudah terjual, tidak dapat dihapus"})
		return
	}
	database.DB.Delete(&item)
	c.JSON(http.StatusOK, gin.H{"message": "item dihapus"})
}

// PATCH /api/admin/stock/:stockId/reset — reset item sold → available (untuk refund/re-stok)
func ResetStockItem(c *gin.Context) {
	var item models.ProductStock
	if err := database.DB.First(&item, c.Param("stockId")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "item tidak ditemukan"})
		return
	}
	item.Sold = false
	item.InvoiceNo = ""
	item.SoldAt = nil
	database.DB.Save(&item)
	c.JSON(http.StatusOK, item)
}

// ClaimStockItems mengambil sejumlah item available secara atomik (transaksi DB).
// Menandai Sold=true + mengisi InvoiceNo agar tidak tabrakan saat transaksi serentak.
func ClaimStockItems(productID uint, qty int, invoiceNo string) ([]models.ProductStock, error) {
	var claimed []models.ProductStock

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var available []models.ProductStock
		if err := tx.
			Where("product_id = ? AND sold = ?", productID, false).
			Limit(qty).
			Find(&available).Error; err != nil {
			return err
		}
		if len(available) < qty {
			return fmt.Errorf("stok tidak mencukupi: hanya %d tersedia, dibutuhkan %d", len(available), qty)
		}

		now := time.Now()
		for i := range available {
			available[i].Sold = true
			available[i].InvoiceNo = invoiceNo
			available[i].SoldAt = &now
			if err := tx.Save(&available[i]).Error; err != nil {
				return err
			}
		}
		claimed = available
		return nil
	})

	return claimed, err
}
