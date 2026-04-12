package handlers

import (
	"digistore/database"
	"digistore/models"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	uploadDir    = "./uploads/products"
	maxFileSize  = 5 << 20 // 5 MB
)

var allowedTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
	"image/gif":  true,
}

// POST /api/admin/products/:id/image
// Menerima multipart form-data dengan field "image"
func UploadProductImage(c *gin.Context) {
	// Pastikan produk ada
	var product models.Product
	if err := database.DB.First(&product, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "produk tidak ditemukan"})
		return
	}

	// Baca file dari form
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "field 'image' tidak ditemukan di form"})
		return
	}
	defer file.Close()

	// Validasi ukuran
	if header.Size > maxFileSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ukuran file maksimal 5 MB"})
		return
	}

	// Deteksi MIME type dari bytes pertama
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	mimeType := http.DetectContentType(buf[:n])
	if !allowedTypes[mimeType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("tipe file tidak diizinkan: %s — gunakan JPEG, PNG, WebP, atau GIF", mimeType),
		})
		return
	}

	// Buat direktori jika belum ada
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal membuat direktori upload"})
		return
	}

	// Hapus gambar lama jika ada
	if product.ImageURL != "" {
		oldPath := "." + product.ImageURL
		os.Remove(oldPath)
	}

	// Buat nama file unik: {product_id}_{timestamp}.{ext}
	ext := extensionFromMime(mimeType)
	filename := fmt.Sprintf("%d_%d%s", product.ID, time.Now().UnixMilli(), ext)
	savePath := filepath.Join(uploadDir, filename)

	// Tulis file (mulai dari awal — karena sudah baca 512 byte untuk deteksi MIME)
	if err := c.SaveUploadedFile(header, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan file"})
		return
	}

	// Simpan URL relatif ke DB
	imageURL := "/uploads/products/" + filename
	database.DB.Model(&product).Update("image_url", imageURL)

	c.JSON(http.StatusOK, gin.H{
		"image_url": imageURL,
		"message":   "gambar berhasil diupload",
	})
}

// DELETE /api/admin/products/:id/image
func DeleteProductImage(c *gin.Context) {
	var product models.Product
	if err := database.DB.First(&product, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "produk tidak ditemukan"})
		return
	}
	if product.ImageURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "produk tidak memiliki gambar"})
		return
	}

	os.Remove("." + product.ImageURL)
	database.DB.Model(&product).Update("image_url", "")

	c.JSON(http.StatusOK, gin.H{"message": "gambar dihapus"})
}

func extensionFromMime(mime string) string {
	switch mime {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	}
	return ".jpg"
}

// ServeUploads melayani file statis dari folder uploads
// Dipanggil dari main.go: r.Static("/uploads", "./uploads")
