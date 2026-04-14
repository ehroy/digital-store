package email

import (
	"bytes"
	"digistore/config"
	"digistore/models"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// =========================
// SEND INVOICE WITH ITEMS
// =========================
func SendInvoiceWithItems(order *models.Order, items []string) {
	var deliveryHTML strings.Builder

	if len(items) > 0 {
		deliveryHTML.WriteString(`
		<div style="margin:20px 0">
		  <h3 style="font-size:15px;font-weight:600;color:#1a1a1a;margin:0 0 10px">📦 Produk Anda</h3>
		  <table style="width:100%;border-collapse:collapse;border-radius:8px;overflow:hidden;border:1px solid #e0e0e0">`)

		for i, item := range items {
			bg := "#fff"
			if i%2 == 1 {
				bg = "#f9f9f9"
			}

			isURL := strings.HasPrefix(item, "http://") || strings.HasPrefix(item, "https://")

			var cell string
			if isURL {
				cell = fmt.Sprintf(`<a href="%s" style="color:#0d5fa8;font-weight:500">Download →</a><br><span style="font-size:11px;color:#999;font-family:monospace">%s</span>`, item, item)
			} else {
				cell = fmt.Sprintf(`<span style="font-family:monospace;font-size:13px;background:#f0f4ff;padding:4px 8px;border-radius:4px;color:#1a1a1a">%s</span>`, item)
			}

			deliveryHTML.WriteString(fmt.Sprintf(`
			<tr style="background:%s">
			  <td style="padding:4px 10px;font-size:12px;color:#999;width:32px">%d</td>
			  <td style="padding:10px 14px">%s</td>
			</tr>`, bg, i+1, cell))
		}

		deliveryHTML.WriteString(`</table>
		<p style="font-size:12px;color:#888;margin-top:8px">⚠️ Simpan email ini. Link/key berlaku permanen.</p>
		</div>`)
	}

	body := buildBase(order, deliveryHTML.String())
	Send(order.BuyerEmail, fmt.Sprintf("Invoice %s — DigiStore ✅", order.InvoiceNo), body)
}

// =========================
// SERVICE INVOICE
// =========================
func SendInvoiceService(order *models.Order) {
	notice := `
	<div style="margin:20px 0;background:#FFF8E1;border-left:4px solid #FFC107;padding:14px 18px;border-radius:0 8px 8px 0">
	  <strong style="font-size:14px">⏳ Pesanan Jasa Diterima</strong>
	  <p style="font-size:13px;color:#555;margin:6px 0 0">
	    Tim kami akan menghubungi Anda melalui email dalam <strong>1×24 jam kerja</strong>.
	  </p>
	</div>`

	body := buildBase(order, notice)
	Send(order.BuyerEmail, fmt.Sprintf("Invoice %s — DigiStore", order.InvoiceNo), body)
}

// =========================
// RESEND SEND FUNCTION
// =========================
type resendPayload struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Html    string   `json:"html"`
}

func Send(to, subject, body string) {
	if config.App.ResendAPIKey == "" {
		log.Printf("[EMAIL SKIP] missing API key — %s", to)
		return
	}

	payload := resendPayload{
		From:    config.App.EmailFrom,
		To:      []string{to},
		Subject: subject,
		Html:    body,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[EMAIL ERROR] marshal: %v", err)
		return
	}

	req, err := http.NewRequest(
		"POST",
		"https://api.resend.com/emails",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		log.Printf("[EMAIL ERROR] request: %v", err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+config.App.ResendAPIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[EMAIL ERROR] send failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Printf("[EMAIL ERROR] status: %d", resp.StatusCode)
		return
	}

	log.Printf("[EMAIL OK] %s → %s", subject, to)
}
func SendWrapper(to, subject, body string) error {
	Send(to, subject, body)
	return nil
}
// =========================
// BASE HTML TEMPLATE
// =========================
func buildBase(order *models.Order, deliverySection string) string {
	return fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="UTF-8"></head>
<body style="font-family:sans-serif;max-width:580px;margin:0 auto;color:#222;font-size:14px">
  <div style="background:#0d5fa8;padding:22px 28px;border-radius:10px 10px 0 0">
    <h1 style="color:#fff;margin:0;font-size:21px">🛍 DigiStore</h1>
    <p style="color:#cde;margin:4px 0 0;font-size:13px">Invoice Pembelian Digital</p>
  </div>

  <div style="border:1px solid #e0e0e0;border-top:none;padding:26px 28px;border-radius:0 0 10px 10px">
    <p>Halo <strong>%s</strong>, terima kasih telah berbelanja di DigiStore!</p>

    <table style="width:100%%;border-collapse:collapse;background:#f8f8f8;border-radius:8px;overflow:hidden;margin:16px 0">
      <tr><td style="padding:9px 14px;color:#666;font-size:12px">INVOICE</td>
          <td style="padding:9px 14px;text-align:right;font-family:monospace;font-weight:700">%s</td></tr>

      <tr style="background:#fff"><td style="padding:9px 14px;color:#666;font-size:12px">PRODUK</td>
          <td style="padding:9px 14px;text-align:right">%s</td></tr>

      <tr><td style="padding:9px 14px;color:#666;font-size:12px">JUMLAH</td>
          <td style="padding:9px 14px;text-align:right">%d pcs</td></tr>

      <tr style="background:#fff"><td style="padding:9px 14px;color:#666;font-size:12px">METODE BAYAR</td>
          <td style="padding:9px 14px;text-align:right">%s</td></tr>

      <tr style="border-top:2px solid #0d5fa8">
        <td style="padding:12px 14px;font-weight:700">TOTAL</td>
        <td style="padding:12px 14px;text-align:right;font-weight:700;font-size:19px;color:#0d5fa8">
          Rp %s
        </td>
      </tr>
    </table>

    %s

    <div style="background:#f0f0f0;border-radius:6px;padding:11px 14px;font-size:12px;color:#888;margin-top:18px">
      Nomor invoice: <strong style="font-family:monospace">%s</strong><br>
      Cek status:
      <a href="%s/cek-invoice?no=%s" style="color:#0d5fa8">
        %s/cek-invoice?no=%s
      </a>
    </div>

  </div>
</body></html>`,
		order.BuyerName,
		order.InvoiceNo,
		order.ProductName,
		order.Qty,
		order.PayMethod,
		formatRupiah(order.Total),
		deliverySection,
		order.InvoiceNo,
		config.App.FrontendURL, order.InvoiceNo,
		config.App.FrontendURL, order.InvoiceNo,
	)
}

// =========================
// PENDING INVOICE
// =========================
func SendPendingInvoice(order *models.Order, payURL, payCode string) {
	paySection := ""

	if payURL != "" {
		paySection = fmt.Sprintf(`
	<div style="margin:20px 0;text-align:center">
	  <a href="%s"
	     style="display:inline-block;background:#0d5fa8;color:#fff;padding:14px 28px;border-radius:8px;font-size:15px;font-weight:600;text-decoration:none">
	    💳 Bayar Sekarang →
	  </a>
	  <p style="font-size:12px;color:#888;margin-top:8px">
	    Link berlaku sampai %v
	  </p>
	</div>`, payURL, order.ExpiredAt)
	}

	if payCode != "" {
		paySection += fmt.Sprintf(`
	<div style="margin:16px 0;padding:14px;background:#f5f5f5;border-radius:8px;text-align:center">
	  <div style="font-size:12px;color:#666">Kode VA</div>
	  <div style="font-family:monospace;font-size:22px;font-weight:700">%s</div>
	</div>`, payCode)
	}

	body := buildBase(order, paySection+`
	<div style="background:#FFF8E1;border-left:4px solid #FFC107;padding:12px 16px;border-radius:6px">
	  ⚠️ Produk akan dikirim otomatis setelah pembayaran dikonfirmasi.
	</div>`)

	Send(order.BuyerEmail, fmt.Sprintf("Selesaikan Pembayaran — Invoice %s", order.InvoiceNo), body)
}

// =========================
// FORMAT RUPIAH
// =========================
func formatRupiah(n int64) string {
	s := fmt.Sprintf("%d", n)
	r := []rune(s)
	var out []rune
	for i, ch := range r {
		if i > 0 && (len(r)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, ch)
	}
	return string(out)
}