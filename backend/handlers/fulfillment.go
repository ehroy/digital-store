package handlers

import (
	"digistore/database"
	"digistore/email"
	"digistore/models"
	"encoding/json"
	"strings"
	"time"
)

const manualProviderETA = 24 * time.Hour

func lookupProviderProductForOrder(order *models.Order) (*models.ProviderProduct, *models.ExternalProvider, error) {
	var product models.Product
	if err := database.DB.First(&product, order.ProductID).Error; err != nil {
		return nil, nil, err
	}

	var providerProduct models.ProviderProduct
	if err := database.DB.Where("provider_name = ? AND code = ?", product.ProviderName, product.ProviderCode).First(&providerProduct).Error; err != nil {
		return nil, nil, err
	}

	var provider models.ExternalProvider
	if err := database.DB.Where("name = ? AND active = ?", product.ProviderName, true).First(&provider).Error; err != nil {
		if err := database.DB.Where("name = ?", product.ProviderName).First(&provider).Error; err != nil {
			return &providerProduct, nil, err
		}
	}

	return &providerProduct, &provider, nil
}

func markOrderPaid(order *models.Order) {
	now := time.Now()
	updates := map[string]any{
		"status":  "paid",
		"paid_at": &now,
	}
	database.DB.Model(order).Updates(updates)
	order.Status = "paid"
	order.PaidAt = &now
}

func markOrderWaitingProvider(order *models.Order) {
	now := time.Now()
	eta := now.Add(manualProviderETA)
	updates := map[string]any{
		"status":               "paid",
		"paid_at":              &now,
		"fulfillment_status":   "waiting_provider",
		"is_fulfilled":         false,
		"expected_delivery_at": &eta,
		"fulfilled_at":         nil,
	}
	database.DB.Model(order).Updates(updates)
	order.Status = "paid"
	order.PaidAt = &now
	order.FulfillmentStatus = "waiting_provider"
	order.IsFulfilled = false
	order.ExpectedDeliveryAt = &eta
	order.FulfilledAt = nil
}

func createPendingProviderOrder(order *models.Order, provider *models.ExternalProvider, providerProduct *models.ProviderProduct) {
	if provider == nil || providerProduct == nil {
		return
	}

	var existing models.ProviderOrder
	if err := database.DB.Where("order_id = ? AND status = ?", order.ID, "pending").First(&existing).Error; err == nil {
		return
	}

	database.DB.Create(&models.ProviderOrder{
		OrderID:      order.ID,
		InvoiceNo:    order.InvoiceNo,
		ProviderID:   provider.ID,
		ProviderCode: providerProduct.Code,
		Status:       "pending",
		Message:      "Menunggu pengiriman manual provider",
		PricePaid:    providerProduct.ProviderPrice * int64(order.Qty),
	})
}

func updateProviderOrderDelivered(order *models.Order, items []string, note string) {
	if len(items) == 0 {
		return
	}
	itemsJSON, _ := json.Marshal(items)
	updates := map[string]any{
		"status":     "success",
		"serial":     string(itemsJSON),
		"message":    note,
		"updated_at": time.Now(),
	}
	res := database.DB.Model(&models.ProviderOrder{}).
		Where("order_id = ? AND status = ?", order.ID, "pending").
		Updates(updates)
	if res.RowsAffected == 0 {
		database.DB.Create(&models.ProviderOrder{
			OrderID:   order.ID,
			InvoiceNo: order.InvoiceNo,
			Status:    "success",
			Serial:    string(itemsJSON),
			Message:   note,
			PricePaid: order.Total,
		})
	}
}

func finalizeOrderDelivery(order *models.Order, product *models.Product, items []string) {
	if order == nil || len(items) == 0 {
		return
	}

	itemsJSON, _ := json.Marshal(items)
	now := time.Now()
	updates := map[string]any{
		"delivered_items":    string(itemsJSON),
		"status":             "paid",
		"fulfillment_status": "fulfilled",
		"is_fulfilled":       true,
		"fulfilled_at":       &now,
	}
	if order.PaidAt == nil {
		updates["paid_at"] = &now
	}
	database.DB.Model(order).Updates(updates)
	order.DeliveredItems = string(itemsJSON)
	order.Status = "paid"
	order.FulfillmentStatus = "fulfilled"
	order.IsFulfilled = true
	order.FulfilledAt = &now
	if order.PaidAt == nil {
		order.PaidAt = &now
	}

	if product != nil {
		runPostPaymentAutomation(order, product)
	}
	go email.SendInvoiceWithItems(order, items)
}

func queueManualProviderFulfillment(order *models.Order, product *models.Product) (bool, error) {
	if order.FulfillmentStatus == "waiting_provider" || order.FulfillmentStatus == "fulfilled" {
		return true, nil
	}

	providerProduct, provider, err := lookupProviderProductForOrder(order)
	if err != nil || providerProduct == nil {
		return false, err
	}

	if strings.ToLower(strings.TrimSpace(providerProduct.Stock)) != "manual" && !providerProduct.IsManual {
		return false, nil
	}

	markOrderWaitingProvider(order)
	createPendingProviderOrder(order, provider, providerProduct)
	go email.SendManualProviderNotice(order)
	if product != nil {
		runPostPaymentAutomation(order, product)
	}
	return true, nil
}
