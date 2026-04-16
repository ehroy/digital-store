package handlers

import (
	"digistore/models"
	"strings"
)

func applyProductWarrantyDefaults(p *models.Product) {
	if p == nil {
		return
	}

	if strings.TrimSpace(p.WarrantyTerms) == "" {
		if p.Type == "provider" {
			p.WarrantyTerms = buildProviderWarrantyTerms(p.ProviderStatus)
		} else {
			p.WarrantyTerms = buildInternalWarrantyTerms(p.Type)
		}
	}
	if strings.TrimSpace(p.TermsAndConditions) == "" {
		if p.Type == "provider" {
			p.TermsAndConditions = buildProviderTermsAndConditions(p.ProviderStatus)
		} else {
			p.TermsAndConditions = buildInternalTermsAndConditions(p.Type)
		}
	}
	if p.Type == "provider" {
		p.WarrantyTerms = decorateWarrantyText(p.WarrantyTerms, "🛡️")
		p.TermsAndConditions = decorateWarrantyText(p.TermsAndConditions, "📌")
	}
}

func applyProviderVariantWarrantyDefaults(v *models.CatalogVariant, stockStatus string) {
	if v == nil {
		return
	}
	if strings.TrimSpace(v.WarrantyTerms) == "" {
		v.WarrantyTerms = buildProviderWarrantyTerms(stockStatus)
	}
	if strings.TrimSpace(v.TermsAndConditions) == "" {
		v.TermsAndConditions = buildProviderTermsAndConditions(stockStatus)
	}
	v.WarrantyTerms = decorateWarrantyText(v.WarrantyTerms, "🛡️")
	v.TermsAndConditions = decorateWarrantyText(v.TermsAndConditions, "📌")
}

func buildInternalWarrantyTerms(productType string) string {
	switch strings.ToLower(strings.TrimSpace(productType)) {
	case "script":
		return joinWarrantyLines([]string{
			"🛡️ Garansi revisi sesuai paket layanan.",
			"⏱️ Komplain/respons maksimal 1x24 jam kerja.",
			"📩 Sertakan invoice dan detail revisi yang diminta.",
			"⚠️ Garansi tidak berlaku untuk perubahan scope di luar paket.",
		})
	case "stock":
		return joinWarrantyLines([]string{
			"🛡️ Garansi file, akses, atau lisensi aktif 7 hari.",
			"⏱️ Klaim maksimal 1x24 jam setelah produk diterima.",
			"📩 Wajib lampirkan invoice dan bukti kendala.",
			"⚠️ Garansi hangus jika data diubah sendiri atau dibagikan ulang.",
		})
	default:
		return joinWarrantyLines([]string{
			"🛡️ Garansi penanganan sesuai jenis produk.",
			"⏱️ Komplain diproses maksimal 1x24 jam kerja.",
			"📩 Sertakan invoice saat mengajukan klaim.",
			"⚠️ Ketentuan detail mengikuti deskripsi produk.",
		})
	}
}

func buildInternalTermsAndConditions(productType string) string {
	switch strings.ToLower(strings.TrimSpace(productType)) {
	case "script":
		return joinWarrantyLines([]string{
			"📌 Revisi mengikuti ruang lingkup paket yang dibeli.",
			"📌 Permintaan di luar brief bisa dikenakan biaya tambahan.",
			"📌 Bukti komplain wajib menyertakan invoice.",
		})
	case "stock":
		return joinWarrantyLines([]string{
			"📌 Produk dikirim dalam bentuk file, lisensi, atau credential sesuai deskripsi.",
			"📌 Simpan semua data pembelian untuk kebutuhan klaim.",
			"📌 Produk yang sudah diubah/dibagikan ulang tidak mendapat garansi.",
		})
	default:
		return joinWarrantyLines([]string{
			"📌 Garansi mengikuti jenis produk dan keterangan pada halaman detail.",
			"📌 Komplain hanya diproses dengan invoice yang valid.",
		})
	}
}

func buildProviderWarrantyTerms(stockStatus string) string {
	switch strings.ToLower(strings.TrimSpace(stockStatus)) {
	case "manual":
		return joinWarrantyLines([]string{
			"🛡️ Garansi mengikuti kebijakan provider manual.",
			"⏱️ Pengiriman dapat memakan waktu sampai 1x24 jam.",
			"📩 Klaim wajib menyertakan invoice dan bukti kendala.",
			"⚠️ Status manual menunggu ketersediaan dari provider.",
		})
	case "out_of_stock":
		return joinWarrantyLines([]string{
			"🛡️ Garansi tidak tersedia sementara karena stok provider kosong.",
			"⏱️ Order akan diproses hanya jika stok tersedia kembali.",
		})
	default:
		return joinWarrantyLines([]string{
			"🛡️ Garansi mengikuti ketentuan provider resmi.",
			"⏱️ Klaim diproses maksimal 1x24 jam kerja.",
			"📩 Simpan invoice dan bukti transaksi untuk klaim.",
			"⚠️ Detail final mengikuti kebijakan provider saat checkout.",
		})
	}
}

func buildProviderTermsAndConditions(stockStatus string) string {
	switch strings.ToLower(strings.TrimSpace(stockStatus)) {
	case "manual":
		return joinWarrantyLines([]string{
			"📌 Produk manual diproses mengikuti antrian provider.",
			"📌 Estimasi pengiriman 1x24 jam atau sesuai kebijakan provider.",
			"📌 Komplain valid jika disertai invoice dan screenshot masalah.",
		})
	default:
		return joinWarrantyLines([]string{
			"📌 Ketentuan produk mengikuti info yang diberikan provider.",
			"📌 Pastikan data pembeli benar sebelum checkout.",
		})
	}
}

func decorateWarrantyText(text, icon string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	lines := strings.Split(text, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "🛡️") || strings.HasPrefix(line, "📌") || strings.HasPrefix(line, "⏱️") || strings.HasPrefix(line, "⚠️") || strings.HasPrefix(line, "📩") {
			result = append(result, line)
			continue
		}
		result = append(result, icon+" "+line)
	}
	return strings.Join(result, "\n")
}

func joinWarrantyLines(lines []string) string {
	clean := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			clean = append(clean, line)
		}
	}
	return strings.Join(clean, "\n")
}
