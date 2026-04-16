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
	"net/url"
	"strings"
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
	Redirect          string `json:"redirect_url"`
	CallbackURL       string `json:"callback_url,omitempty"`
	WebhookURL        string `json:"webhook_url,omitempty"`
	ExpiredMinutes    int    `json:"expired_minutes,omitempty"`
}

type SBInvoiceData struct {
	ID            string `json:"id"`
	InvoiceNumber string `json:"invoice_number"` // nomor invoice dari SayaBayar, mis: INV-20240327-0042
	Amount        int64  `json:"amount"`
	AmountUnique  int64  `json:"amount_unique"`
	UniqueCode    int    `json:"unique_code"`
	PayURL        string `json:"pay_url"`
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

func ExtractPaymentRef(raw string) string {
	if raw == "" {
		return ""
	}
	if parsed, err := url.Parse(raw); err == nil {
		path := strings.Trim(parsed.Path, "/")
		if path != "" {
			parts := strings.Split(path, "/")
			return strings.TrimSpace(parts[len(parts)-1])
		}
	}
	parts := strings.Split(strings.TrimRight(strings.Split(raw, "?")[0], "/"), "/")
	return strings.TrimSpace(parts[len(parts)-1])
}

// SBWebhookPayload — payload yang dikirim SayaBayar ke webhook endpoint kamu
type SBWebhookPayload struct {
	Event     string        `json:"event"` // "invoice.paid" | "invoice.expired"
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
	if len(respBody) == 0 {
		return nil, fmt.Errorf("sayabayar: empty response status=%d", resp.StatusCode)
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("sayabayar: parse gagal: %w — status=%d body=%s", err, resp.StatusCode, string(respBody))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if result.Message != "" {
			return nil, fmt.Errorf("sayabayar: http %d: %s", resp.StatusCode, result.Message)
		}
		if result.Errors != nil {
			return nil, fmt.Errorf("sayabayar: http %d: %+v", resp.StatusCode, result.Errors)
		}
		return nil, fmt.Errorf("sayabayar: http %d: %s", resp.StatusCode, string(respBody))
	}
	if !result.Success {
		if result.Message != "" {
			return nil, fmt.Errorf("sayabayar: %s", result.Message)
		}
		if result.Errors != nil {
			return nil, fmt.Errorf("sayabayar: %+v", result.Errors)
		}
		return nil, fmt.Errorf("sayabayar: response tidak sukses: %s", string(respBody))
	}
	return &result, nil
}

type SBActionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

func (s *SayaBayar) SelectChannel(invoiceID string, channelType string) error {
	if strings.TrimSpace(invoiceID) == "" {
		return fmt.Errorf("invoice id kosong")
	}
	body, _ := json.Marshal(map[string]string{"channel_type": channelType})
	req, err := http.NewRequest("POST", SayaBayarBaseURL+"/pay/"+invoiceID+"/select-channel", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("sayabayar: %w", err)
	}
	s.setHeaders(req)
	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("sayabayar: request select-channel gagal: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	var result SBActionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("sayabayar: parse select-channel gagal: %w — %s", err, string(respBody))
	}
	if !result.Success {
		return fmt.Errorf("sayabayar: select-channel gagal: %s", result.Message)
	}
	return nil
}

func (s *SayaBayar) ConfirmPayment(invoiceID string) error {
	if strings.TrimSpace(invoiceID) == "" {
		return fmt.Errorf("invoice id kosong")
	}
	req, err := http.NewRequest("POST", SayaBayarBaseURL+"/pay/"+invoiceID+"/confirm", bytes.NewBuffer([]byte(`{}`)))
	if err != nil {
		return fmt.Errorf("sayabayar: %w", err)
	}
	s.setHeaders(req)
	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("sayabayar: request confirm gagal: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	var result SBActionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("sayabayar: parse confirm gagal: %w — %s", err, string(respBody))
	}
	if !result.Success {
		return fmt.Errorf("sayabayar: confirm gagal: %s", result.Message)
	}
	return nil
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
