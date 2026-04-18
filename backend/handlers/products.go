package handlers

import (
	"digistore/database"
	"digistore/models"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetProducts(c *gin.Context) {
	isAdmin := c.Query("admin") == "1"
	var products []models.Product
	query := database.DB
	if !isAdmin {
		query = query.Where("active = ?", true)
	}
	if cat := c.Query("category"); cat != "" {
		query = query.Where("category = ?", cat)
	}
	if s := strings.TrimSpace(c.Query("search")); s != "" {
		like := "%" + s + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", like, like)
	}
	if typ := strings.TrimSpace(c.Query("type")); typ != "" && typ != "all" {
		query = query.Where("type = ?", typ)
	}
	if active := strings.TrimSpace(c.Query("active")); active != "" && active != "all" {
		switch active {
		case "true", "1", "active":
			query = query.Where("active = ?", true)
		case "false", "0", "inactive":
			query = query.Where("active = ?", false)
		}
	}
	if popular := strings.TrimSpace(c.Query("popular")); popular != "" && popular != "all" {
		switch popular {
		case "true", "1", "popular":
			query = query.Where("is_popular = ?", true)
		case "false", "0", "regular":
			query = query.Where("is_popular = ?", false)
		}
	}

	if isAdmin {
		page := parsePositiveQueryInt(c.Query("page"), 1)
		perPage := parsePositiveQueryInt(c.Query("per_page"), 10)
		if perPage > 100 {
			perPage = 100
		}
		sortMode := strings.ToLower(strings.TrimSpace(c.Query("sort")))
		if sortMode == "" {
			sortMode = "terbaru"
		}

		var total int64
		if err := query.Model(&models.Product{}).Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal memuat produk"})
			return
		}

		query = applyProductSort(query, sortMode)
		if err := query.Offset((page - 1) * perPage).Limit(perPage).Find(&products).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal memuat produk"})
			return
		}

		for i := range products {
			populateProductAvailability(&products[i])
			applyProductWarrantyDefaults(&products[i])
		}

		pages := int(total) / perPage
		if int(total)%perPage != 0 {
			pages++
		}
		if pages == 0 {
			pages = 1
		}

		c.JSON(http.StatusOK, gin.H{
			"items":       products,
			"total":       total,
			"page":        page,
			"per_page":    perPage,
			"total_pages": pages,
			"sort":        sortMode,
		})
		return
	}

	query.Find(&products)

	for i := range products {
		populateProductAvailability(&products[i])
	}
	productCounts, familyCounts := buildProductSalesIndex(products)

	if !isAdmin {
		products = normalizePublicCatalog(products)
		products = sortPublicCatalog(products, c.Query("sort"), productCounts, familyCounts)
	} else {
		products = sortPublicCatalog(products, c.Query("sort"), productCounts, familyCounts)
	}
	if !isAdmin {
		products = sanitizePublicCatalog(products)
	}

	for i := range products {
		applyProductWarrantyDefaults(&products[i])
	}

	var publicCategories []string
	if !isAdmin {
		_ = database.DB.Model(&models.Product{}).
			Where("active = ?", true).
			Distinct("category").
			Order("category ASC").
			Pluck("category", &publicCategories).Error
	}

	if page := parsePositiveQueryInt(c.Query("page"), 0); page > 0 {
		perPage := parsePositiveQueryInt(c.Query("per_page"), 10)
		if perPage > 100 {
			perPage = 100
		}

		var categories []string
		seenCategories := make(map[string]struct{})
		for _, p := range products {
			cat := strings.TrimSpace(p.Category)
			if cat == "" {
				continue
			}
			if _, ok := seenCategories[cat]; ok {
				continue
			}
			seenCategories[cat] = struct{}{}
			categories = append(categories, cat)
		}

		total := len(products)
		start := (page - 1) * perPage
		if start > total {
			start = total
		}
		end := start + perPage
		if end > total {
			end = total
		}

		c.JSON(http.StatusOK, gin.H{
			"items":    products[start:end],
			"total":    total,
			"page":     page,
			"per_page": perPage,
			"total_pages": func() int {
				pages := total / perPage
				if total%perPage != 0 {
					pages++
				}
				if pages == 0 {
					return 1
				}
				return pages
			}(),
			"categories": publicCategories,
		})
		return
	}

	if !isAdmin {
		c.JSON(http.StatusOK, gin.H{"items": products, "categories": publicCategories})
		return
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
	applyProductWarrantyDefaults(&p)
	sanitizePublicProduct(&p)
	c.JSON(http.StatusOK, p)
}

func CreateProduct(c *gin.Context) {
	var p models.Product
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	applyProductWarrantyDefaults(&p)
	applyProviderPricing(&p)
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
	applyProductWarrantyDefaults(&p)
	applyProviderPricing(&p)
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

func BulkProducts(c *gin.Context) {
	var body struct {
		IDs                      []uint  `json:"ids" binding:"required,min=1"`
		Action                   string  `json:"action" binding:"required"`
		MarkupType               string  `json:"markup_type"`
		MarkupValue              float64 `json:"markup_value"`
		UseProviderDefaultMarkup *bool   `json:"use_provider_default_markup"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ids := uniqueUintIDs(body.IDs)
	if len(ids) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tidak ada produk yang dipilih"})
		return
	}

	action := strings.ToLower(strings.TrimSpace(body.Action))
	if action == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "aksi bulk wajib diisi"})
		return
	}

	updatedCount := 0
	deletedCount := 0
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		switch action {
		case "delete":
			res := tx.Delete(&models.Product{}, ids)
			if res.Error != nil {
				return res.Error
			}
			deletedCount = int(res.RowsAffected)
		case "activate", "deactivate", "toggle_active":
			active := true
			if action == "deactivate" {
				active = false
			}
			res := tx.Model(&models.Product{}).Where("id IN ?", ids).Update("active", active)
			if res.Error != nil {
				return res.Error
			}
			updatedCount = int(res.RowsAffected)
		case "set_markup":
			markupType := normalizeMarkupType(body.MarkupType)
			markupValue := body.MarkupValue
			for _, id := range ids {
				var p models.Product
				if err := tx.First(&p, id).Error; err != nil {
					continue
				}
				if p.Type != "provider" {
					continue
				}
				p.MarkupType = markupType
				p.MarkupValue = markupValue
				p.UseProviderDefaultMarkup = false
				applyProviderPricing(&p)
				if err := tx.Save(&p).Error; err != nil {
					return err
				}
				updatedCount++
			}
		default:
			return gorm.ErrInvalidData
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gagal memproses bulk action"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Aksi bulk berhasil diproses",
		"updated": updatedCount,
		"deleted": deletedCount,
	})
}

func populateProductAvailability(p *models.Product) {
	switch p.Type {
	case "stock":
		var avail, total int64
		database.DB.Model(&models.ProductStock{}).Where("product_id = ?", p.ID).Count(&total)
		database.DB.Model(&models.ProductStock{}).Where("product_id = ? AND sold = ?", p.ID, false).Count(&avail)
		p.AvailableStock = int(avail)
		p.TotalStock = int(total)
		p.InternalStock = int(avail)
		p.StockSource = "stock"
	case "provider":
		var internalAvail int64
		database.DB.Model(&models.ProductStock{}).Where("product_id = ? AND sold = ?", p.ID, false).Count(&internalAvail)
		p.InternalStock = int(internalAvail)
		if internalAvail > 0 {
			p.StockSource = "internal"
		}
		var pp models.ProviderProduct
		if err := database.DB.Where("provider_name = ? AND code = ?", p.ProviderName, p.ProviderCode).First(&pp).Error; err == nil {
			p.ProviderStock = pp.AvailableStock
			p.ProviderStatus = pp.Stock
			p.AvailableStock = int(internalAvail) + pp.AvailableStock
			if internalAvail > 0 && pp.AvailableStock > 0 {
				p.StockSource = "combined"
			} else if internalAvail > 0 {
				p.StockSource = "internal"
			} else {
				p.StockSource = "provider"
			}
		} else {
			p.ProviderStatus = "unknown"
			p.AvailableStock = int(internalAvail)
			if internalAvail > 0 {
				p.StockSource = "internal"
			} else {
				p.StockSource = "provider"
			}
		}
	}
}

func applyProviderPricing(p *models.Product) {
	if p.Type != "provider" {
		return
	}

	p.MarkupType = normalizeMarkupType(p.MarkupType)
	if p.UseProviderDefaultMarkup {
		var provider models.ExternalProvider
		if err := database.DB.Where("name = ?", p.ProviderName).First(&provider).Error; err == nil {
			markupType := normalizeMarkupType(provider.DefaultMarkupType)
			markupValue := provider.DefaultMarkupValue
			if markupValue < 0 {
				markupValue = 15
			}
			p.MarkupType = markupType
			p.MarkupValue = markupValue
			p.Price = calcSellPrice(p.ProviderPrice, markupType, markupValue)
			return
		}
	}

	if p.MarkupValue < 0 {
		p.MarkupValue = 0
	}
	p.Price = calcSellPrice(p.ProviderPrice, p.MarkupType, p.MarkupValue)
}

func normalizeMarkupType(raw string) string {
	markupType := strings.ToLower(strings.TrimSpace(raw))
	switch markupType {
	case "fixed":
		return "fixed"
	default:
		return "percent"
	}
}

func uniqueUintIDs(ids []uint) []uint {
	seen := make(map[uint]struct{}, len(ids))
	result := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
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

func sortPublicCatalog(products []models.Product, sortMode string, productCounts map[uint]int64, familyCounts map[string]int64) []models.Product {
	sortMode = strings.ToLower(strings.TrimSpace(sortMode))
	if sortMode == "" {
		sortMode = "terbaru"
	}

	if productCounts == nil || familyCounts == nil {
		productCounts, familyCounts = buildProductSalesIndex(products)
	}

	sort.SliceStable(products, func(i, j int) bool {
		left := catalogSortValue(products[i], sortMode, productCounts, familyCounts)
		right := catalogSortValue(products[j], sortMode, productCounts, familyCounts)

		if left.primary != right.primary {
			if sortMode == "termurah" {
				return left.primary < right.primary
			}
			return left.primary > right.primary
		}
		if left.secondary != right.secondary {
			if sortMode == "termurah" {
				return left.secondary > right.secondary
			}
			return left.secondary > right.secondary
		}
		return strings.ToLower(products[i].Name) < strings.ToLower(products[j].Name)
	})

	return products
}

func applyProductSort(query *gorm.DB, sortMode string) *gorm.DB {
	switch sortMode {
	case "terlaris", "terpopuler", "popular", "best_seller", "bestseller":
		return query.Order("is_popular desc").Order("updated_at desc").Order("id desc")
	case "termurah", "cheapest", "price_asc":
		return query.Order("price asc").Order("updated_at desc").Order("id desc")
	case "terbaru", "newest", "latest":
		return query.Order("updated_at desc").Order("created_at desc").Order("id desc")
	default:
		return query.Order("created_at desc").Order("updated_at desc").Order("id desc")
	}
}

type catalogSortInfo struct {
	primary   int64
	secondary int64
}

func catalogSortValue(p models.Product, sortMode string, productCounts map[uint]int64, familyCounts map[string]int64) catalogSortInfo {
	switch sortMode {
	case "terlaris", "terpopuler", "popular", "best_seller", "bestseller":
		return catalogSortInfo{
			primary:   boolToSortValue(p.IsPopular),
			secondary: catalogSalesCount(p, productCounts, familyCounts),
		}
	case "termurah", "cheapest", "price_asc":
		return catalogSortInfo{
			primary:   p.Price,
			secondary: catalogSalesCount(p, productCounts, familyCounts),
		}
	case "terbaru", "newest", "latest":
		return catalogSortInfo{
			primary:   p.UpdatedAt.Unix(),
			secondary: p.CreatedAt.Unix(),
		}
	default:
		return catalogSortInfo{
			primary:   p.CreatedAt.Unix(),
			secondary: p.UpdatedAt.Unix(),
		}
	}
}

func boolToSortValue(v bool) int64 {
	if v {
		return 1
	}
	return 0
}

func catalogSalesCount(p models.Product, productCounts map[uint]int64, familyCounts map[string]int64) int64 {
	if p.Type == "provider" && strings.Contains(strings.ToLower(p.ProviderName), "koala") {
		return familyCounts[providerCatalogKey(p.Name)]
	}
	return productCounts[p.ID]
}

func buildProductSalesIndex(products []models.Product) (map[uint]int64, map[string]int64) {
	productCounts := map[uint]int64{}
	familyCounts := map[string]int64{}

	var orderRows []struct {
		ProductID uint
		Total     int64
	}
	database.DB.Model(&models.Order{}).
		Select("product_id, COUNT(*) as total").
		Where("status = ? OR fulfillment_status = ?", "paid", "fulfilled").
		Group("product_id").
		Scan(&orderRows)
	for _, row := range orderRows {
		productCounts[row.ProductID] = row.Total
	}

	for _, p := range products {
		if p.Type == "provider" && strings.Contains(strings.ToLower(p.ProviderName), "koala") {
			familyCounts[providerCatalogKey(p.Name)] += productCounts[p.ID]
		}
	}

	return productCounts, familyCounts
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
		if p.InternalStock > 0 {
			stock = "available"
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
			ProductID:          p.ID,
			ProviderCode:       p.ProviderCode,
			VariantName:        providerVariantName(p.Name),
			DurationLabel:      providerDurationName(p.Name),
			AccountType:        providerAccountType(p.Name),
			Region:             providerRegionName(p.Name),
			WarrantyTerms:      p.WarrantyTerms,
			TermsAndConditions: p.TermsAndConditions,
			StockStatus:        stock,
			AvailableStock:     p.AvailableStock,
			InternalStock:      p.InternalStock,
			Price:              p.Price,
			OriginalPrice:      p.ProviderPrice,
			IsActive:           p.Active,
			StockSource:        p.StockSource,
		})
		applyProviderVariantWarrantyDefaults(&variants[len(variants)-1], stock)
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
	for _, p := range products {
		if p.IsPopular {
			best.IsPopular = true
			break
		}
	}
	best.ProviderStatus = providerCatalogStatus(variants)
	best.ProviderStock = totalStock
	best.InternalStock = 0
	for _, p := range products {
		best.InternalStock += p.InternalStock
	}
	best.AvailableStock = totalStock
	best.AvailableStock += best.InternalStock
	best.StockSource = "provider"
	if best.InternalStock > 0 && best.ProviderStock > 0 {
		best.StockSource = "combined"
	} else if best.InternalStock > 0 {
		best.StockSource = "internal"
	}
	best.WarrantyTerms = pickWarrantyText(products, true)
	best.TermsAndConditions = pickTermsText(products, true)
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

func parsePositiveQueryInt(raw string, fallback int) int {
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || v < 1 {
		return fallback
	}
	return v
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

func sanitizePublicCatalog(products []models.Product) []models.Product {
	for i := range products {
		sanitizePublicProduct(&products[i])
	}
	return products
}

func sanitizePublicProduct(p *models.Product) {
	if p == nil {
		return
	}
	p.ProviderName = ""
	p.ProviderCode = ""
	p.ProviderPrice = 0
	p.MarkupType = ""
	p.MarkupValue = 0
	p.UseProviderDefaultMarkup = false
	p.AutoSync = false
	p.ProviderStock = 0
	p.InternalStock = 0
	p.ProviderStatus = ""
	p.StockSource = ""
	if len(p.Variants) == 0 {
		return
	}
	for i := range p.Variants {
		sanitizePublicVariant(&p.Variants[i])
	}
}

func sanitizePublicVariant(v *models.CatalogVariant) {
	if v == nil {
		return
	}
	v.ProviderCode = ""
	v.InternalStock = 0
	v.StockSource = ""
	v.OriginalPrice = 0
}

func pickWarrantyText(products []models.Product, isProvider bool) string {
	for _, p := range products {
		if strings.TrimSpace(p.WarrantyTerms) != "" {
			return decorateWarrantyText(p.WarrantyTerms, "🛡️")
		}
	}
	if isProvider {
		return buildProviderWarrantyTerms("available")
	}
	if len(products) > 0 {
		return buildInternalWarrantyTerms(products[0].Type)
	}
	return ""
}

func pickTermsText(products []models.Product, isProvider bool) string {
	for _, p := range products {
		if strings.TrimSpace(p.TermsAndConditions) != "" {
			return decorateWarrantyText(p.TermsAndConditions, "📌")
		}
	}
	if isProvider {
		return buildProviderTermsAndConditions("available")
	}
	if len(products) > 0 {
		return buildInternalTermsAndConditions(products[0].Type)
	}
	return ""
}
