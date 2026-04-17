package handlers

import (
	"digistore/models"
	"testing"
)

func TestResolveProviderFulfillmentSourcePrefersInternalStock(t *testing.T) {
	provider := &models.ProviderProduct{Stock: "available", AvailableStock: 99}
	source, err := resolveProviderFulfillmentSource(3, provider, 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if source != "stock" {
		t.Fatalf("expected stock source, got %q", source)
	}
}

func TestResolveProviderFulfillmentSourceFallsBackToProvider(t *testing.T) {
	provider := &models.ProviderProduct{Stock: "available", AvailableStock: 5}
	source, err := resolveProviderFulfillmentSource(0, provider, 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if source != "provider" {
		t.Fatalf("expected provider source, got %q", source)
	}
}

func TestResolveProviderFulfillmentSourceRejectsWhenProviderInsufficient(t *testing.T) {
	provider := &models.ProviderProduct{Stock: "available", AvailableStock: 1}
	if _, err := resolveProviderFulfillmentSource(0, provider, 2); err == nil {
		t.Fatal("expected error when provider stock is insufficient")
	}
}

func TestResolveProviderFulfillmentSourceRejectsWhenAllUnavailable(t *testing.T) {
	provider := &models.ProviderProduct{Stock: "out_of_stock", AvailableStock: 0}
	if _, err := resolveProviderFulfillmentSource(0, provider, 1); err == nil {
		t.Fatal("expected error when both internal and provider stock are unavailable")
	}
}
