package handlers

import (
	"digistore/database"
	"digistore/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetScriptLogs(c *gin.Context) {
	var logs []models.ScriptLog
	database.DB.Order("created_at desc").Find(&logs)
	c.JSON(http.StatusOK, logs)
}

func GetDashboard(c *gin.Context) {
	var totalOrders, paidOrders, pendingOrders int64
	var totalRevenue struct{ Sum int64 }
	var lowStockCount, activeProducts int64
	var recentOrders []models.Order
	var categories []struct {
		Category string
		Revenue  int64
	}

	database.DB.Model(&models.Order{}).Count(&totalOrders)
	database.DB.Model(&models.Order{}).Where("status = ?", "paid").Count(&paidOrders)
	database.DB.Model(&models.Order{}).Where("status = ?", "pending").Count(&pendingOrders)
	database.DB.Model(&models.Order{}).
		Select("COALESCE(SUM(total),0) as sum").Where("status = ?", "paid").Scan(&totalRevenue)
	database.DB.Model(&models.Product{}).Where("active = ?", true).Count(&activeProducts)
	database.DB.Model(&models.Product{}).
		Where("type = ? AND id IN (SELECT product_id FROM product_stocks WHERE sold = false GROUP BY product_id HAVING COUNT(*) < 5)", "stock").
		Count(&lowStockCount)
	database.DB.Model(&models.Order{}).Order("created_at desc").Limit(8).Find(&recentOrders)
	database.DB.Model(&models.Order{}).
		Select("p.category as category, COALESCE(SUM(o.total),0) as revenue").
		Joins("JOIN products p ON p.id = o.product_id").
		Where("o.status = ?", "paid").
		Group("p.category").Order("revenue desc").
		Table("orders o").Scan(&categories)

	c.JSON(http.StatusOK, gin.H{
		"total_orders":        totalOrders,
		"paid_orders":         paidOrders,
		"pending_orders":      pendingOrders,
		"total_revenue":       totalRevenue.Sum,
		"active_products":     activeProducts,
		"low_stock":           lowStockCount,
		"recent_orders":       recentOrders,
		"revenue_by_category": categories,
	})
}
