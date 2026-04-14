package handlers

import (
	"digistore/database"
	"digistore/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GET /api/contact — public, untuk storefront
func GetContactConfig(c *gin.Context) {
	var cfg models.ContactConfig
	database.DB.FirstOrCreate(&cfg, models.ContactConfig{ID: 1})
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
	var cfg models.ContactConfig
	database.DB.FirstOrCreate(&cfg, models.ContactConfig{ID: 1})
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cfg.ID = 1
	database.DB.Save(&cfg)
	c.JSON(http.StatusOK, cfg)
}
