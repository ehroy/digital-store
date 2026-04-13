// Package gateway — SayaBayar Payment Gateway
// Docs: https://api.sayabayar.com/v1
// Auth: header X-API-Key
package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const SayaBayarBaseURL = "https://api.sayabayar.com/v1"

type SayaBayar struct {
	APIKey     string
	HTTPClient *http.Client
}

func NewSayaBayar(apiKey string) *SayaBayar {
	return &SayaBayar{
		APIKey:     apiKey,
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
	}
}

type SBCreateRequest struct {
	CustomerName      string `json:"customer_name"`
	CustomerEmail     string `json:"customer_email"`
	Amount            int64  `json:"amount"`
	Description       string `json:"description"`
	ChannelPreference string `json:"channel_preference"` // "platform" | "client"
	ExpiredMinutes    int    `json:"expired_minutes,omitempty"`
}

type SBInvoiceData struct {
	ID            string `json:"id"`
	InvoiceNumber string `json:"invoice_number"` // nomor invoice dari SayaBayar, mis: INV-20240327-0042
	Amount        int64  `json:"amount"`
	AmountUnique  int64  `json:"amount_unique"`
	UniqueCode    int    `json:"unique_code"`
	PaymentURL    string `json:"payment_url"`
	Status        string `json:"status"` // pending | paid | expired
	ExpiredAt     string `json:"expired_at"`
	CreatedAt     string `json:"created_at"`
}

type SBResponse struct {
	Success bool          `json:"success"`
	Data    SBInvoiceData `json:"data"`
	Meta    struct {
		RequestID string `json:"request_id"`
		Timestamp string `json:"timestamp"`
	} `json:"meta"`
	Message string `json:"message,omitempty"`
	Errors  any    `json:"errors,omitempty"`
}

// SBWebhookPayload — payload yang dikirim SayaBayar ke webhook endpoint kamu
type SBWebhookPayload struct {
	Event     string        `json:"event"`   // "invoice.paid" | "invoice.expired"
	Invoice   SBInvoiceData `json:"invoice"`
	Timestamp string        `json:"timestamp"`
}

func (s *SayaBayar) CreateInvoice(req SBCreateRequest) (*SBResponse, error) {
	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequest("POST", SayaBayarBaseURL+"/invoices", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("sayabayar: %w", err)
	}
	s.setHeaders(httpReq)

	resp, err := s.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sayabayar: request gagal: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	var result SBResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("sayabayar: parse gagal: %w — %s", err, string(respBody))
	}
	if !result.Success {
		return nil, fmt.Errorf("sayabayar: %s", result.Message)
	}
	return &result, nil
}

func (s *SayaBayar) GetInvoice(id string) (*SBResponse, error) {
	httpReq, err := http.NewRequest("GET", SayaBayarBaseURL+"/invoices/"+id, nil)
	if err != nil {
		return nil, err
	}
	s.setHeaders(httpReq)
	resp, err := s.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result SBResponse
	json.Unmarshal(body, &result)
	return &result, nil
}

func (s *SayaBayar) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", s.APIKey)
	req.Header.Set("Accept", "application/json")
}
