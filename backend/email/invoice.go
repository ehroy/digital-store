package email

import (
	"digistore/config"
	"digistore/models"
	"fmt"
	"log"
	"strconv"
	"strings"

	"gopkg.in/gomail.v2"
)

// SendInvoiceWithItems — untuk produk stok: kirim invoice + item yang diterima pembeli
func SendInvoiceWithItems(order *models.Order, items []string) {
	var deliveryHTML strings.Builder
	if len(items) > 0 {
		deliveryHTML.WriteString(`
		<div style="margin:20px 0">
		  <h3 style="font-size:15px;font-weight:600;color:#1a1a1a;margin:0 0 10px">📦 Produk Anda</h3>
		  <table style="width:100%;border-collapse:collapse;border-radius:8px;overflow:hidden;border:1px solid #e0e0e0">`)
		for i, item := range items {
			bg := "#fff"
			if i%2 == 1 { bg = "#f9f9f9" }
			// Deteksi apakah item adalah URL
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
	send(order.BuyerEmail, fmt.Sprintf("Invoice %s — DigiStore ✅", order.InvoiceNo), body)
}

// SendInvoiceService — untuk produk script/jasa: kirim invoice tanpa item, info tim akan hubungi
func SendInvoiceService(order *models.Order) {
	notice := `
	<div style="margin:20px 0;background:#FFF8E1;border-left:4px solid #FFC107;padding:14px 18px;border-radius:0 8px 8px 0">
	  <strong style="font-size:14px">⏳ Pesanan Jasa Diterima</strong>
	  <p style="font-size:13px;color:#555;margin:6px 0 0">
	    Tim kami akan menghubungi Anda melalui email dalam <strong>1×24 jam kerja</strong>.
	    Harap pantau inbox Anda.
	  </p>
	</div>`
	body := buildBase(order, notice)
	send(order.BuyerEmail, fmt.Sprintf("Invoice %s — DigiStore", order.InvoiceNo), body)
}

// Send — digunakan script executor untuk notifikasi ke tim internal
func Send(to, subject, body string) error {
	if config.App.SMTPUser == "" {
		log.Printf("[EMAIL SKIP] to=%s | %s", to, subject)
		return nil
	}
	m := gomail.NewMessage()
	m.SetHeader("From", config.App.SMTPFrom)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)
	port, _ := strconv.Atoi(config.App.SMTPPort)
	d := gomail.NewDialer(config.App.SMTPHost, port, config.App.SMTPUser, config.App.SMTPPass)
	return d.DialAndSend(m)
}

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
        <td style="padding:12px 14px;text-align:right;font-weight:700;font-size:19px;color:#0d5fa8">Rp %s</td>
      </tr>
    </table>
    %s
    <div style="background:#f0f0f0;border-radius:6px;padding:11px 14px;font-size:12px;color:#888;margin-top:18px">
      Nomor invoice: <strong style="font-family:monospace">%s</strong> — simpan untuk klaim garansi / refund.
      Cek status: <a href="%s/cek-invoice?no=%s" style="color:#0d5fa8">%s/cek-invoice?no=%s</a>
    </div>
  </div>
</body></html>`,
		order.BuyerName, order.InvoiceNo, order.ProductName, order.Qty,
		order.PayMethod, formatRupiah(order.Total),
		deliverySection,
		order.InvoiceNo,
		config.App.FrontendURL, order.InvoiceNo,
		config.App.FrontendURL, order.InvoiceNo,
	)
}

func send(to, subject, body string) {
	if config.App.SMTPUser == "" {
		log.Printf("[EMAIL SKIP] SMTP belum dikonfigurasi — invoice %s untuk %s", subject, to)
		return
	}
	m := gomail.NewMessage()
	m.SetHeader("From", config.App.SMTPFrom)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)
	port, _ := strconv.Atoi(config.App.SMTPPort)
	d := gomail.NewDialer(config.App.SMTPHost, port, config.App.SMTPUser, config.App.SMTPPass)
	if err := d.DialAndSend(m); err != nil {
		log.Printf("[EMAIL ERROR] %v", err)
	} else {
		log.Printf("[EMAIL OK] %s → %s", subject, to)
	}
}

func formatRupiah(n int64) string {
	s := fmt.Sprintf("%d", n)
	r := []rune(s)
	var out []rune
	for i, ch := range r {
		if i > 0 && (len(r)-i)%3 == 0 { out = append(out, '.') }
		out = append(out, ch)
	}
	return string(out)
}

// SendPendingInvoice — kirim email saat order dibuat tapi belum dibayar.
// Menyertakan payment_url dan instruksi bayar.
func SendPendingInvoice(order *models.Order, payURL, payCode string) {
	paySection := ""
	if payURL != "" {
		paySection = fmt.Sprintf(`
	<div style="margin:20px 0;text-align:center">
	  <a href="%s"
	     style="display:inline-block;background:#0d5fa8;color:#fff;padding:14px 28px;
	            border-radius:8px;font-size:15px;font-weight:600;text-decoration:none">
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
	  <div style="font-size:12px;color:#666;margin-bottom:4px">Kode Pembayaran / Nomor VA</div>
	  <div style="font-family:monospace;font-size:22px;font-weight:700;letter-spacing:2px;color:#1a1a1a">%s</div>
	</div>`, payCode)
	}

	body := buildBase(order, paySection+`
	<div style="background:#FFF8E1;border-left:4px solid #FFC107;padding:12px 16px;border-radius:0 6px 6px 0;font-size:13px">
	  ⚠️ Produk akan dikirim <strong>otomatis</strong> setelah pembayaran dikonfirmasi.
	</div>`)

	send(order.BuyerEmail, fmt.Sprintf("Selesaikan Pembayaran — Invoice %s", order.InvoiceNo), body)
}
