# Global Rules

Project ini menggunakan konfigurasi AI agent berbasis folder.

JANGAN membuat rule baru di root ini.

## Instruksi utama

Selalu gunakan konfigurasi yang sudah didefinisikan di:

- backend/AGENTS.md → untuk semua logic backend (Golang, API, payment, produk, fulfillment)
- frontend/AGENTS.md → untuk semua logic frontend (Svelte, UI, checkout, UX)

## Cara kerja

Jika task berkaitan dengan:

### Backend

- API
- Payment QRIS
- Webhook
- Validasi pembayaran
- Sinkronisasi produk dari provider
- Struktur database
- Fulfillment produk

WAJIB mengikuti:
→ backend/AGENTS.md

---

### Frontend

- UI/UX
- Tampilan produk
- Variant selector
- Checkout
- Status pembayaran
- Notifikasi ke user

WAJIB mengikuti:
→ frontend/AGENTS.md

---

## Aturan penting

- Jangan override rules dari backend/AGENTS.md atau frontend/AGENTS.md
- Jangan mencampur logic backend dan frontend dalam satu implementasi
- Selalu pisahkan concern sesuai folder
- Jika terjadi konflik rule, prioritaskan rule yang paling dekat dengan file yang sedang dikerjakan

## Tujuan

Menjaga:

- konsistensi arsitektur
- keamanan flow pembayaran
- normalisasi produk
- pemisahan backend dan frontend dengan jelas
