---

## `frontend/AGENTS.md`

```md
# Frontend Rules

Frontend menggunakan Svelte/SvelteKit.
Frontend hanya menampilkan katalog yang sudah dinormalisasi oleh backend.

## Prioritas utama

Frontend bukan sumber kebenaran untuk status pembayaran.
Frontend tidak boleh:

- menentukan order paid
- memaksa fulfill
- menganggap callback sukses
- menampilkan produk terkirim hanya dari hasil polling yang ambigu

Semua status final pembayaran harus berasal dari backend.

## Tampilan katalog produk

Jangan render produk mentah provider satu per satu jika sebenarnya itu varian dari produk utama yang sama.

### Wajib

- tampilkan satu kartu produk utama per brand/layanan
- tampilkan varian di halaman detail, modal, dropdown, tabs, atau selector
- contoh: `Netflix` tampil satu kali, lalu user memilih varian seperti sharing/private/durasi/slot

### Dilarang

- menampilkan 5 kartu Netflix hanya karena API punya 5 SKU berbeda
- mencampur produk utama dengan nama varian di grid utama
- menampilkan nama provider mentah yang berantakan tanpa normalisasi

## Struktur UI yang diinginkan

### Product list

Tampilkan ringkas:

- nama produk utama
- thumbnail
- kategori
- harga mulai dari varian termurah
- badge stok jika perlu

### Product detail

Tampilkan:

- nama produk utama
- deskripsi produk
- pilihan varian
- harga sesuai varian terpilih
- informasi penting varian
- tombol beli

### Variant selector

Wajib bisa menangani:

- durasi
- jenis akun
- sharing/private
- region
- slot/profile

## Sumber data

Frontend harus prioritaskan endpoint normalized dari backend, bukan payload mentah provider.

Gunakan data seperti:

- product
- variants
- selectedVariant
- lowestPrice
- availability

Jangan coupling UI ke field provider mentah jika backend sudah punya bentuk normalized.

## Payment UI policy

Untuk sementara payment method hanya:

- QRIS

UI checkout hanya menampilkan QRIS.
Jangan tampilkan metode lain.
Jangan siapkan fallback method di UI.

## Payment status UX

Saat user selesai membuat order QRIS:

- tampilkan QRIS
- tampilkan nominal
- tampilkan invoice/order id
- tampilkan countdown expired jika ada
- tampilkan status awal: `Menunggu pembayaran`

Setelah user scan dan bayar:

- tampilkan status seperti:
  - Menunggu pembayaran
  - Pembayaran sedang diverifikasi
  - Pembayaran berhasil
  - Pembayaran gagal
  - Pembayaran kedaluwarsa

## Pesan wajib ke user

Saat status belum final `paid`, tampilkan pesan yang jelas:

- "Pembayaran kamu belum dikonfirmasi."
- "Jika kamu sudah membayar, sistem sedang memverifikasi pembayaran."
- "Produk akan dikirim otomatis setelah pembayaran berhasil diverifikasi."
- "Jangan tutup invoice jika status masih menunggu verifikasi."
- "Jika terjadi keterlambatan update, silakan coba cek status beberapa saat lagi."

Jika gateway/backend error:

- jangan tampilkan seolah pembayaran berhasil
- tampilkan pesan aman seperti:
  - "Status pembayaran belum dapat dipastikan."
  - "Jangan lakukan pembayaran ulang sebelum memeriksa status order."
  - "Silakan refresh atau cek status pembayaran beberapa saat lagi."

## Aturan status frontend

Gunakan state yang sinkron dengan backend:

- waiting_payment
- verifying
- paid
- failed
- expired
- canceled

Mapping UI:

- `waiting_payment` -> "Menunggu pembayaran"
- `verifying` -> "Pembayaran sedang diverifikasi"
- `paid` -> "Pembayaran berhasil"
- `failed` -> "Pembayaran gagal"
- `expired` -> "QRIS kedaluwarsa"
- `canceled` -> "Pesanan dibatalkan"

Jangan ada state lokal `paid` buatan frontend tanpa response backend.

## Fulfillment UX

Produk digital hanya boleh ditampilkan ke user jika backend sudah menyatakan order paid dan fulfilled atau siap dikirim.

Jangan render data produk sensitif ketika:

- order masih waiting_payment
- order masih verifying
- gateway error
- status ambigu

Jika status belum final:

- sembunyikan data akun / voucher / credential / item digital
- tampilkan placeholder status dan instruksi ke user

## Error handling UI

Wajib ada komponen error yang ramah user untuk:

- gagal memuat QRIS
- gagal memuat status order
- pembayaran belum terkonfirmasi
- order expired
- gangguan sementara dari server

Gunakan pesan sederhana, jangan tampilkan raw error teknis provider/gateway ke user.

## Polling / refresh status

Jika ada polling:

- gunakan interval wajar
- hentikan polling saat status final
- jangan spam request
- saat request status gagal, tetap tampilkan status aman, bukan paid

## Komponen yang disarankan

- `ProductCard.svelte`
- `ProductVariantSelector.svelte`
- `CheckoutSummary.svelte`
- `QrisPaymentPanel.svelte`
- `PaymentStatusAlert.svelte`
- `OrderStatusTimeline.svelte`

## Prinsip implementasi

- UI bersih dan mudah dipahami
- mobile-first
- bahasa user-facing pakai Bahasa Indonesia
- harga dan status harus mudah dibaca
- semua aksi penting harus berdasarkan response backend terbaru

## Hal yang dilarang

JANGAN:

- menganggap sukses hanya karena user klik "Saya sudah bayar"
- mengubah state order menjadi paid dari frontend
- menampilkan data produk sebelum backend konfirmasi
- menampilkan banyak kartu produk untuk layanan yang sama hanya karena beda varian
- menampilkan metode pembayaran selain QRIS
```
