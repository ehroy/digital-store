package handlers

import (
	"digistore/models"
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

func testTime(day int) time.Time {
	return time.Date(2026, time.April, day, 10, 0, 0, 0, time.UTC)
}
