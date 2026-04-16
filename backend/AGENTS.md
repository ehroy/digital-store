# Backend Rules

Backend ini adalah sumber kebenaran untuk:

- sinkronisasi produk dari API eksternal
- normalisasi katalog produk
- checkout
- pembayaran QRIS
- verifikasi status pembayaran
- pengiriman produk digital setelah pembayaran valid

## Prioritas utama

JANGAN pernah mengirim produk hanya karena:

- callback gateway error
- timeout ke gateway
- response status tidak lengkap
- frontend mengirim status paid
- order lokal belum tervalidasi ke gateway

Produk hanya boleh dikirim jika backend sudah memastikan status pembayaran benar-benar berhasil/paid dari hasil verifikasi yang valid.

## Payment policy

Untuk sementara payment gateway hanya:

- QRIS

Jangan generate method lain.
Jangan tampilkan payment lain dari backend.
Jangan simpan fallback ke VA, e-wallet lain, atau transfer manual.

## Payment state machine

Gunakan state pembayaran yang jelas dan aman:

- pending
- waiting_payment
- verifying
- paid
- failed
- expired
- canceled

Aturan:

- order baru dibuat sebagai `pending` atau `waiting_payment`
- setelah QRIS berhasil dibuat, tetap `waiting_payment`
- saat callback masuk atau proses cek status dilakukan, ubah ke `verifying` bila perlu
- ubah ke `paid` hanya setelah verifikasi valid ke gateway / source of truth
- jika gateway error, network error, signature invalid, payload invalid, atau status ambigu: JANGAN ubah ke `paid`
- jika status tidak bisa dipastikan, tetap `waiting_payment` atau `verifying`
- produk hanya dikirim saat status final `paid`

## Verifikasi pembayaran

Wajib lakukan semua ini:

- validasi signature webhook jika gateway menyediakan signature
- validasi nominal yang dibayar sesuai order
- validasi reference / transaction id cocok dengan order
- validasi status final dari gateway
- log semua request dan response penting dari gateway
- simpan raw payload callback untuk audit
- lakukan idempotency check agar webhook berulang tidak mengirim produk dua kali

Jika webhook menyatakan sukses tapi verifikasi lanjutan gagal:

- tandai `verifying`
- jangan kirim produk
- jangan ubah menjadi `paid`

Jika gateway tidak bisa dihubungi:

- jangan asumsikan paid
- jangan kirim produk
- arahkan sistem untuk retry pengecekan status secara aman

## Pengiriman produk digital

Produk digital hanya boleh diproses setelah:

1. order valid
2. status pembayaran final `paid`
3. order belum pernah fulfilled
4. stok / data produk masih tersedia

Wajib gunakan proteksi:

- field seperti `is_fulfilled`
- `paid_at`
- `fulfilled_at`
- `payment_reference`
- locking / transaksi database saat fulfill

Jangan pernah fulfill dari frontend.
Semua fulfill harus terjadi dari backend.

## Integrasi produk dari API eksternal

Sumber produk berasal dari:

- API Koalastore

Tetapi frontend tidak boleh menampilkan produk mentah 1:1 dari API kalau sebenarnya itu hanya varian dari produk utama yang sama.

### Tujuan normalisasi katalog

Jika ada banyak item dari API eksternal yang sebenarnya milik satu brand / family yang sama, contoh:

- Netflix 1P1U
- Netflix 2P1U
- Netflix Sharing
- Netflix Private
- Netflix 1 Bulan
- Netflix 3 Bulan

maka di frontend harus menjadi:

- satu produk utama: `Netflix`
- varian-varian ditampilkan di dalam detail/opsi produk, bukan sebagai kartu produk terpisah

## Struktur data katalog yang wajib dipakai

Backend harus memisahkan:

- external product raw data
- normalized product group
- normalized variant

Contoh konsep model:

- `ProviderProductRaw`
- `CatalogProduct`
- `CatalogVariant`

Contoh field yang disarankan:

### CatalogProduct

- id
- slug
- name
- brand
- category
- description
- thumbnail
- is_active
- provider
- sort_order

### CatalogVariant

- id
- product_id
- provider_sku
- variant_name
- duration_label
- account_type
- region
- stock_status
- price
- original_price
- is_active

## Aturan grouping produk

Saat sinkronisasi dari provider:

- kelompokkan item yang brand utamanya sama menjadi satu `CatalogProduct`
- simpan perbedaan paket sebagai `CatalogVariant`
- jangan duplikasi kartu produk di frontend hanya karena beda paket
- gunakan slug yang stabil, misalnya `netflix`, `spotify`, `youtube-premium`

### Heuristik grouping minimal

Coba ekstrak dari nama provider:

- brand utama
- tipe paket
- durasi
- private/sharing
- region
- slot/profile

Contoh:

- "Netflix 1P1U 1 Bulan" -> product: Netflix, variant: 1P1U / 1 Bulan
- "Netflix Sharing 1 Bulan" -> product: Netflix, variant: Sharing / 1 Bulan

Jika mapping ambigu:

- jangan langsung merge secara agresif
- buat helper normalizer yang eksplisit dan mudah dikembangkan
- tambahkan unit test untuk nama-nama produk raw yang rawan salah group

## Arsitektur yang diinginkan

Pisahkan layer:

- handler/controller
- service
- repository
- provider client
- payment service
- fulfillment service
- catalog normalizer

Jangan taruh business logic di handler.

Disarankan module:

- `internal/catalog`
- `internal/provider/koalastore`
- `internal/payment`
- `internal/order`
- `internal/fulfillment`

## Endpoint backend yang diharapkan

Prioritaskan endpoint yang sudah dinormalisasi untuk frontend:

- GET `/api/catalog/products`
- GET `/api/catalog/products/:slug`
- POST `/api/checkout`
- GET `/api/orders/:invoice`
- POST `/api/webhook/qris`
- POST `/api/orders/:invoice/check-payment`

Frontend sebaiknya tidak konsumsi payload mentah provider jika sudah ada endpoint normalized.

## Error handling

Wajib:

- response JSON konsisten
- jangan bocorkan stack trace
- tulis log detail di server
- bedakan validation error, provider error, gateway error, dan internal error

Contoh response:

```json
{
  "success": false,
  "message": "Pembayaran masih menunggu verifikasi",
  "data": null,
  "error_code": "PAYMENT_NOT_CONFIRMED"
}
```
