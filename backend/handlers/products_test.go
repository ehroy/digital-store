package handlers

import (
	"digistore/models"
	"strings"
	"testing"
	"time"
)

func TestNormalizePublicCatalogGroupsProviderVariants(t *testing.T) {
	products := []models.Product{
		{ID: 1, Name: "Template A", Type: "stock"},
		{ID: 2, Name: "Netflix — Sharing 1 Bulan", Type: "provider", ProviderName: "KoalaStore", Price: 50000, ProviderStatus: "available", UpdatedAt: testTime(2)},
		{ID: 3, Name: "Netflix — Private 1 Bulan", Type: "provider", ProviderName: "KoalaStore", Price: 45000, ProviderStatus: "available", ImageURL: "/uploads/products/netflix-private.jpg", UpdatedAt: testTime(3)},
		{ID: 4, Name: "Spotify — Family 1 Bulan", Type: "provider", ProviderName: "KoalaStore", Price: 30000, ProviderStatus: "manual", UpdatedAt: testTime(1)},
	}

	result := normalizePublicCatalog(products)
	if len(result) != 3 {
		t.Fatalf("expected grouped products to total 3, got %d", len(result))
	}
	var netflix models.Product
	for _, p := range result {
		if p.Name == "Netflix" {
			netflix = p
			break
		}
	}
	if netflix.ID == 0 {
		t.Fatal("expected grouped Netflix product")
	}
	if len(netflix.Variants) != 2 {
		t.Fatalf("expected 2 Netflix variants, got %d", len(netflix.Variants))
	}
	if netflix.Price != 45000 {
		t.Fatalf("expected lowest variant price, got %d", netflix.Price)
	}
	if netflix.ImageURL != "/uploads/products/netflix-private.jpg" {
		t.Fatalf("expected grouped image_url to be preserved, got %q", netflix.ImageURL)
	}
	if netflix.ProviderStock != 0 || netflix.AvailableStock != 0 {
		t.Fatalf("expected grouped stock to reflect variant stock cache, got %+v", netflix)
	}
}

func TestSortPublicCatalogByCheapest(t *testing.T) {
	products := []models.Product{
		{ID: 1, Name: "Template A", Type: "stock", Price: 25000, CreatedAt: testTime(1), UpdatedAt: testTime(1)},
		{ID: 2, Name: "Netflix — Sharing 1 Bulan", Type: "provider", ProviderName: "KoalaStore", Price: 50000, ProviderStatus: "available", CreatedAt: testTime(2), UpdatedAt: testTime(2)},
		{ID: 3, Name: "Spotify — Family 1 Bulan", Type: "provider", ProviderName: "KoalaStore", Price: 30000, ProviderStatus: "manual", CreatedAt: testTime(3), UpdatedAt: testTime(3)},
	}
	result := sortPublicCatalog(products, "termurah", map[uint]int64{1: 0, 2: 0, 3: 0}, map[string]int64{"netflix": 0, "spotify": 0})
	if result[0].ID != 1 {
		t.Fatalf("expected cheapest product first, got %+v", result[0])
	}
	if result[1].ID != 3 {
		t.Fatalf("expected second cheapest product next, got %+v", result[1])
	}
}

func TestSortPublicCatalogByTerlaris(t *testing.T) {
	products := normalizePublicCatalog([]models.Product{
		{ID: 1, Name: "Template A", Type: "stock", Price: 25000, CreatedAt: testTime(1), UpdatedAt: testTime(1)},
		{ID: 2, Name: "Netflix — Sharing 1 Bulan", Type: "provider", ProviderName: "KoalaStore", Price: 50000, ProviderStatus: "available", CreatedAt: testTime(2), UpdatedAt: testTime(2)},
		{ID: 3, Name: "Netflix — Private 1 Bulan", Type: "provider", ProviderName: "KoalaStore", Price: 45000, ProviderStatus: "available", CreatedAt: testTime(3), UpdatedAt: testTime(3)},
		{ID: 4, Name: "Spotify — Family 1 Bulan", Type: "provider", ProviderName: "KoalaStore", Price: 30000, ProviderStatus: "manual", CreatedAt: testTime(4), UpdatedAt: testTime(4)},
	})
	productCounts := map[uint]int64{1: 1, 2: 2, 3: 5, 4: 1}
	familyCounts := map[string]int64{"netflix": 7, "spotify": 1}
	result := sortPublicCatalog(products, "terlaris", productCounts, familyCounts)
	if result[0].Name != "Netflix" {
		t.Fatalf("expected Netflix first as terlaris, got %s", result[0].Name)
	}
	if result[0].Price != 45000 {
		t.Fatalf("expected grouped Netflix lowest price preserved, got %d", result[0].Price)
	}
}

func TestSortPublicCatalogHonorsPopularFlag(t *testing.T) {
	products := normalizePublicCatalog([]models.Product{
		{ID: 1, Name: "Template A", Type: "stock", Price: 25000, CreatedAt: testTime(1), UpdatedAt: testTime(1)},
		{ID: 2, Name: "Netflix — Sharing 1 Bulan", Type: "provider", ProviderName: "KoalaStore", Price: 50000, ProviderStatus: "available", CreatedAt: testTime(2), UpdatedAt: testTime(2)},
		{ID: 3, Name: "Spotify — Family 1 Bulan", Type: "provider", ProviderName: "KoalaStore", Price: 30000, ProviderStatus: "manual", IsPopular: true, CreatedAt: testTime(3), UpdatedAt: testTime(3)},
	})
	result := sortPublicCatalog(products, "terlaris", map[uint]int64{1: 0, 2: 10, 3: 1}, map[string]int64{"netflix": 10, "spotify": 1})
	if result[0].Name != "Spotify" {
		t.Fatalf("expected popular product first, got %s", result[0].Name)
	}
	if result[0].IsPopular != true {
		t.Fatalf("expected popular flag preserved, got %+v", result[0])
	}
}

func TestApplyProductWarrantyDefaults(t *testing.T) {
	stock := models.Product{Type: "stock"}
	applyProductWarrantyDefaults(&stock)
	if stock.WarrantyTerms == "" || stock.TermsAndConditions == "" {
		t.Fatal("expected internal warranty defaults to be populated")
	}
	if !strings.HasPrefix(stock.WarrantyTerms, "🛡️") {
		t.Fatalf("expected warranty lines to start with icon, got %q", stock.WarrantyTerms)
	}

	provider := models.Product{Type: "provider", ProviderStatus: "manual"}
	applyProductWarrantyDefaults(&provider)
	if provider.WarrantyTerms == "" || provider.TermsAndConditions == "" {
		t.Fatal("expected provider warranty defaults to be populated")
	}
	if !strings.HasPrefix(provider.WarrantyTerms, "🛡️") {
		t.Fatalf("expected provider warranty lines to start with icon, got %q", provider.WarrantyTerms)
	}
}

func testTime(day int) time.Time {
	return time.Date(2026, time.April, day, 10, 0, 0, 0, time.UTC)
}
