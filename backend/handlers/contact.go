package handlers

import (
	"digistore/database"
	"digistore/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// GET /api/contact — public, untuk storefront
func GetContactConfig(c *gin.Context) {
	var cfg models.ContactConfig
	database.DB.FirstOrCreate(&cfg, models.ContactConfig{ID: 1})
	ensureContactDefaults(&cfg)
	database.DB.Save(&cfg)
	// Sensor data sensitif untuk public endpoint
	c.JSON(http.StatusOK, gin.H{
		"whatsapp":           cfg.WhatsApp,
		"whatsapp_label":     cfg.WhatsAppLabel,
		"telegram":           cfg.Telegram,
		"instagram":          cfg.Instagram,
		"email":              cfg.Email,
		"website":            cfg.Website,
		"business_name":      cfg.BusinessName,
		"business_desc":      cfg.BusinessDesc,
		"complaint_template": cfg.ComplaintTemplate,
		"operational_hours":  cfg.OperationalHours,
	})
}

// PUT /api/admin/contact — admin update
func UpdateContactConfig(c *gin.Context) {
	var existing models.ContactConfig
	database.DB.FirstOrCreate(&existing, models.ContactConfig{ID: 1})

	var input models.ContactConfig
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	mergeContactConfig(&existing, &input)
	ensureContactDefaults(&existing)
	existing.ID = 1
	database.DB.Save(&existing)
	c.JSON(http.StatusOK, existing)
}

func mergeContactConfig(dst, src *models.ContactConfig) {
	if src == nil || dst == nil {
		return
	}
	if strings.TrimSpace(src.WhatsApp) != "" {
		dst.WhatsApp = src.WhatsApp
	}
	if strings.TrimSpace(src.WhatsAppLabel) != "" {
		dst.WhatsAppLabel = src.WhatsAppLabel
	}
	if strings.TrimSpace(src.Telegram) != "" {
		dst.Telegram = src.Telegram
	}
	if strings.TrimSpace(src.Instagram) != "" {
		dst.Instagram = src.Instagram
	}
	if strings.TrimSpace(src.Email) != "" {
		dst.Email = src.Email
	}
	if strings.TrimSpace(src.Website) != "" {
		dst.Website = src.Website
	}
	if strings.TrimSpace(src.BusinessName) != "" {
		dst.BusinessName = src.BusinessName
	}
	if strings.TrimSpace(src.BusinessDesc) != "" {
		dst.BusinessDesc = src.BusinessDesc
	}
	if strings.TrimSpace(src.ComplaintTemplate) != "" {
		dst.ComplaintTemplate = src.ComplaintTemplate
	}
	if strings.TrimSpace(src.OperationalHours) != "" {
		dst.OperationalHours = src.OperationalHours
	}
}

func ensureContactDefaults(cfg *models.ContactConfig) {
	if cfg == nil {
		return
	}
	if strings.TrimSpace(cfg.BusinessName) == "" {
		cfg.BusinessName = "DigiStore"
	}
	if strings.TrimSpace(cfg.WhatsApp) == "" {
		cfg.WhatsApp = "6281234567890"
	}
	if strings.TrimSpace(cfg.WhatsAppLabel) == "" {
		cfg.WhatsAppLabel = "Hubungi CS"
	}
	if strings.TrimSpace(cfg.ComplaintTemplate) == "" {
		cfg.ComplaintTemplate = "Halo admin, saya ingin komplain order berikut:\n\nInvoice: {invoice_no}\nProduk: {product_name}\nNama: {buyer_name}\nEmail: {buyer_email}\nNomor HP: {phone}\nStatus: {status}\n\nMasalah:\n{issue}"
	}
	if strings.TrimSpace(cfg.OperationalHours) == "" {
		cfg.OperationalHours = "Senin - Sabtu, 08.00 - 21.00 WIB"
	}
}
