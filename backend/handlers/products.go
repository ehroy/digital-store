package handlers

import (
	"digistore/database"
	"digistore/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetProducts(c *gin.Context) {
	var products []models.Product
	query := database.DB.Order("created_at desc")
	if c.Query("admin") != "1" {
		query = query.Where("active = ?", true)
	}
	if cat := c.Query("category"); cat != "" {
		query = query.Where("category = ?", cat)
	}
	query.Find(&products)

	// Hitung stok tersedia untuk setiap produk
	for i := range products {
		if products[i].Type == "stock" {
			var avail, total int64
			database.DB.Model(&models.ProductStock{}).Where("product_id = ?", products[i].ID).Count(&total)
			database.DB.Model(&models.ProductStock{}).Where("product_id = ? AND sold = ?", products[i].ID, false).Count(&avail)
			products[i].AvailableStock = int(avail)
			products[i].TotalStock = int(total)
		}
	}

	c.JSON(http.StatusOK, products)
}

func GetProduct(c *gin.Context) {
	var p models.Product
	if err := database.DB.First(&p, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "produk tidak ditemukan"})
		return
	}
	if p.Type == "stock" {
		var avail, total int64
		database.DB.Model(&models.ProductStock{}).Where("product_id = ?", p.ID).Count(&total)
		database.DB.Model(&models.ProductStock{}).Where("product_id = ? AND sold = ?", p.ID, false).Count(&avail)
		p.AvailableStock = int(avail)
		p.TotalStock = int(total)
	}
	c.JSON(http.StatusOK, p)
}

func CreateProduct(c *gin.Context) {
	var p models.Product
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p.ID = 0
	database.DB.Create(&p)
	c.JSON(http.StatusCreated, p)
}

func UpdateProduct(c *gin.Context) {
	var p models.Product
	if err := database.DB.First(&p, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "produk tidak ditemukan"})
		return
	}
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	database.DB.Save(&p)
	c.JSON(http.StatusOK, p)
}

func DeleteProduct(c *gin.Context) {
	database.DB.Delete(&models.Product{}, c.Param("id"))
	c.JSON(http.StatusOK, gin.H{"message": "produk dihapus"})
}

func ToggleProduct(c *gin.Context) {
	var p models.Product
	if err := database.DB.First(&p, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "produk tidak ditemukan"})
		return
	}
	p.Active = !p.Active
	database.DB.Save(&p)
	c.JSON(http.StatusOK, p)
}
