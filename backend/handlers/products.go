package handlers

import (
	"digistore/database"
	"digistore/models"
	"net/http"
	"sort"
	"strings"

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

	for i := range products {
		populateProductAvailability(&products[i])
	}

	if c.Query("admin") != "1" {
		products = normalizePublicCatalog(products)
	}

	c.JSON(http.StatusOK, products)
}

func GetProduct(c *gin.Context) {
	var p models.Product
	if err := database.DB.First(&p, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "produk tidak ditemukan"})
		return
	}
	populateProductAvailability(&p)
	if p.Type == "provider" {
		attachProviderVariants(&p)
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

func populateProductAvailability(p *models.Product) {
	switch p.Type {
	case "stock":
		var avail, total int64
		database.DB.Model(&models.ProductStock{}).Where("product_id = ?", p.ID).Count(&total)
		database.DB.Model(&models.ProductStock{}).Where("product_id = ? AND sold = ?", p.ID, false).Count(&avail)
		p.AvailableStock = int(avail)
		p.TotalStock = int(total)
	case "provider":
		var pp models.ProviderProduct
		if err := database.DB.Where("provider_name = ? AND code = ?", p.ProviderName, p.ProviderCode).First(&pp).Error; err == nil {
			p.ProviderStock = pp.AvailableStock
			p.ProviderStatus = pp.Stock
			if pp.Stock == "out_of_stock" {
				p.AvailableStock = 0
			} else {
				p.AvailableStock = pp.AvailableStock
			}
		} else {
			p.ProviderStatus = "unknown"
			p.AvailableStock = 0
		}
	}
}

func normalizePublicCatalog(products []models.Product) []models.Product {
	groups := make(map[string][]models.Product)
	emitted := make(map[string]bool)
	result := make([]models.Product, 0, len(products))

	for _, p := range products {
		if p.Type != "provider" || !strings.Contains(strings.ToLower(p.ProviderName), "koala") {
			result = append(result, p)
			continue
		}
		key := providerCatalogKey(p.Name)
		groups[key] = append(groups[key], p)
	}

	for _, p := range products {
		if p.Type != "provider" || !strings.Contains(strings.ToLower(p.ProviderName), "koala") {
			continue
		}
		key := providerCatalogKey(p.Name)
		if emitted[key] {
			continue
		}
		result = append(result, buildProviderCatalogGroup(groups[key]))
		emitted[key] = true
	}
	return result
}

func attachProviderVariants(p *models.Product) {
	var products []models.Product
	database.DB.
		Where("provider_name = ? AND type = ? AND active = ?", p.ProviderName, "provider", true).
		Find(&products)
	if len(products) == 0 {
		p.Variants = nil
		return
	}
	group := make([]models.Product, 0, len(products))
	family := providerCatalogKey(p.Name)
	for _, prod := range products {
		if providerCatalogKey(prod.Name) == family {
			populateProductAvailability(&prod)
			group = append(group, prod)
		}
	}
	grouped := buildProviderCatalogGroup(group)
	p.Variants = grouped.Variants
	p.Price = grouped.Price
	p.ImageURL = grouped.ImageURL
	p.ProviderStatus = grouped.ProviderStatus
	p.ProviderStock = grouped.ProviderStock
	p.AvailableStock = grouped.AvailableStock
}

func buildProviderCatalogGroup(products []models.Product) models.Product {
	if len(products) == 0 {
		return models.Product{}
	}

	best := products[0]
	for _, p := range products[1:] {
		if providerProductBetter(p, best) {
			best = p
		}
	}

	variants := make([]models.CatalogVariant, 0, len(products))
	availableCount := 0
	totalStock := 0
	minPrice := int64(0)
	hasMinPrice := false
	for _, p := range products {
		stock := p.ProviderStatus
		if stock == "" {
			stock = "unknown"
		}
		if stock != "out_of_stock" {
			availableCount++
		}
		totalStock += p.ProviderStock
		if !hasMinPrice || p.Price < minPrice {
			minPrice = p.Price
			hasMinPrice = true
		}
		variants = append(variants, models.CatalogVariant{
			ProductID:      p.ID,
			ProviderCode:   p.ProviderCode,
			VariantName:    providerVariantName(p.Name),
			DurationLabel:  providerDurationName(p.Name),
			AccountType:    providerAccountType(p.Name),
			Region:         providerRegionName(p.Name),
			StockStatus:    stock,
			AvailableStock: p.ProviderStock,
			Price:          p.Price,
			OriginalPrice:  p.ProviderPrice,
			IsActive:       p.Active,
		})
	}
	sort.SliceStable(variants, func(i, j int) bool {
		if variants[i].StockStatus != variants[j].StockStatus {
			return variants[i].StockStatus != "out_of_stock"
		}
		if variants[i].Price != variants[j].Price {
			return variants[i].Price < variants[j].Price
		}
		return variants[i].VariantName < variants[j].VariantName
	})

	best.Name = providerFamilyName(best.Name)
	best.ImageURL = providerGroupImageURL(products, best)
	best.Price = minPrice
	best.Variants = variants
	best.ProviderStatus = providerCatalogStatus(variants)
	best.ProviderStock = totalStock
	best.AvailableStock = totalStock
	return best
}

func providerGroupImageURL(products []models.Product, best models.Product) string {
	if strings.TrimSpace(best.ImageURL) != "" {
		return best.ImageURL
	}
	for _, p := range products {
		if strings.TrimSpace(p.ImageURL) != "" {
			return p.ImageURL
		}
	}
	return ""
}

func providerProductBetter(candidate, current models.Product) bool {
	candidateAvailable := candidate.ProviderStatus == "available" || candidate.ProviderStatus == "manual"
	currentAvailable := current.ProviderStatus == "available" || current.ProviderStatus == "manual"
	if candidateAvailable != currentAvailable {
		return candidateAvailable
	}
	if candidate.Price != current.Price {
		return candidate.Price < current.Price
	}
	if !candidate.UpdatedAt.Equal(current.UpdatedAt) {
		return candidate.UpdatedAt.After(current.UpdatedAt)
	}
	return candidate.ID < current.ID
}

func providerCatalogKey(name string) string {
	return strings.ToLower(strings.TrimSpace(providerFamilyName(name)))
}

func providerFamilyName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	for _, sep := range []string{" — ", " - ", " | ", " : "} {
		if parts := strings.SplitN(name, sep, 2); len(parts) == 2 {
			name = parts[0]
			break
		}
	}
	return strings.Join(strings.Fields(name), " ")
}

func providerVariantName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	for _, sep := range []string{" — ", " - ", " | ", " : "} {
		if parts := strings.SplitN(name, sep, 2); len(parts) == 2 {
			return strings.Join(strings.Fields(parts[1]), " ")
		}
	}
	return ""
}

func providerDurationName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	for _, token := range []string{"1 Bulan", "3 Bulan", "6 Bulan", "12 Bulan"} {
		if strings.Contains(strings.ToLower(name), strings.ToLower(token)) {
			return token
		}
	}
	return ""
}

func providerAccountType(name string) string {
	name = strings.ToLower(name)
	switch {
	case strings.Contains(name, "private"):
		return "private"
	case strings.Contains(name, "sharing"):
		return "sharing"
	case strings.Contains(name, "family"):
		return "family"
	default:
		return ""
	}
}

func providerRegionName(name string) string {
	name = strings.ToLower(name)
	for _, token := range []string{"id", "indo", "us", "sg", "my", "hk"} {
		if strings.Contains(name, token) {
			return strings.ToUpper(token)
		}
	}
	return ""
}

func providerCatalogStatus(variants []models.CatalogVariant) string {
	for _, v := range variants {
		if v.StockStatus == "available" || v.StockStatus == "manual" {
			return "available"
		}
	}
	return "out_of_stock"
}
