# 🛍 DigiStore — Platform Penjualan Produk Digital

**Stack:** Go (Gin + GORM) · SvelteKit · SQLite · Docker

---

## Alur Kerja Sistem

### 🟦 Produk Tipe: STOK

Digunakan untuk produk yang bisa langsung dikirim otomatis (ebook, template, source code, plugin, dll).

```
Pembeli klik Beli
       │
       ▼
Isi nama + email + jumlah
       │
       ▼
Pilih metode pembayaran
       │
       ▼
Klik "Konfirmasi Pesanan"
       │
       ├──► Stok dikurangi otomatis
       │
       ├──► Email invoice dikirim ke pembeli
       │    berisi:
       │    ┌─────────────────────────────────┐
       │    │  📦 Link Download Produk        │
       │    │  ┌──────────────────────────┐   │
       │    │  │ File Utama      [Download]│   │
       │    │  │ Dokumentasi PDF [Download]│   │
       │    │  │ Bonus           [Download]│   │
       │    │  └──────────────────────────┘   │
       │    └─────────────────────────────────┘
       │
       └──► Order tersimpan di database
            Status: pending
```

**Link download** diisi oleh admin di panel produk. Bisa berupa:
- Google Drive (set akses "Anyone with link")
- Dropbox shared link
- AWS S3 / Cloudflare R2 presigned URL
- GitHub release zip
- Link langsung ke file apapun

---

### 🟪 Produk Tipe: SCRIPT

Digunakan untuk produk berbasis jasa (desain, setup VPS, konsultasi, dll) yang TIDAK punya stok fisik. Saat order masuk, sistem mengeksekusi serangkaian *actions* otomatis.

```
Pembeli klik Beli
       │
       ▼
Isi nama + email + konfirmasi
       │
       ▼
Order tersimpan
       │
       ├──► Email invoice ke pembeli
       │    (memberitahu tim akan hubungi dalam 1x24 jam)
       │
       └──► Script Actions dieksekusi OTOMATIS (background):
            │
            ├──[action: email]────► Kirim email notifikasi ke tim internal
            │                       mis: designer@example.com, ops@example.com
            │
            ├──[action: webhook]──► POST ke URL eksternal
            │                       mis: Slack webhook, Notion API, custom backend
            │
            └──[action: log]──────► Catat pesan ke Script Logs (admin panel)
                                    mis: "Tiket #001 dibuat untuk John Doe"
```

**Semua output** setiap action tersimpan di halaman **Script Logs** di admin panel, lengkap dengan waktu eksekusi, status (berhasil/gagal), dan pesan output.

---

## Format Script Actions (JSON Array)

Script disimpan sebagai JSON array di field `script` produk.

### Action: `email`
Mengirim email ke alamat yang ditentukan (biasanya tim internal).

```json
{
  "type": "email",
  "to": "designer@example.com",
  "subject": "Order Baru: {{product_name}} - {{invoice_no}}",
  "body": "Ada order baru masuk!\n\nInvoice : {{invoice_no}}\nPembeli : {{buyer_name}}\nEmail   : {{buyer_email}}\nTotal   : {{total}}\n\nSegera hubungi klien."
}
```

### Action: `webhook`
Mengirim HTTP POST ke URL eksternal dengan data order.

```json
{
  "type": "webhook",
  "url": "https://hooks.slack.com/services/XXX/YYY/ZZZ",
  "method": "POST",
  "headers": {
    "Content-Type": "application/json",
    "X-Api-Key": "secret123"
  }
}
```

Payload yang dikirim otomatis:
```json
{
  "invoice_no": "INV-20260410-123456",
  "product_name": "Layanan Desain Logo",
  "buyer_name": "Andi Pratama",
  "buyer_email": "andi@gmail.com",
  "total": "Rp 299.000",
  "timestamp": "2026-04-10T10:30:00Z"
}
```

### Action: `log`
Menyimpan pesan ke Script Logs di admin panel.

```json
{
  "type": "log",
  "message": "Tiket desain dibuat untuk {{buyer_name}} ({{buyer_email}}) - invoice {{invoice_no}}"
}
```

### Contoh Script Lengkap (Layanan Jasa)
```json
[
  {
    "type": "email",
    "to": "tim@example.com",
    "subject": "🎨 Order Jasa Baru: {{invoice_no}}",
    "body": "Klien  : {{buyer_name}}\nEmail  : {{buyer_email}}\nProduk : {{product_name}}\nTotal  : {{total}}"
  },
  {
    "type": "webhook",
    "url": "https://hooks.slack.com/services/XXX",
    "method": "POST"
  },
  {
    "type": "log",
    "message": "Order {{invoice_no}} diterima dari {{buyer_name}}"
  }
]
```

### Variabel Template

| Variabel | Nilai |
|---|---|
| `{{invoice_no}}` | Nomor invoice unik (mis: INV-20260410-123456) |
| `{{product_name}}` | Nama produk yang dibeli |
| `{{buyer_name}}` | Nama lengkap pembeli |
| `{{buyer_email}}` | Alamat email pembeli |
| `{{total}}` | Total pembayaran (mis: Rp 299.000) |
| `{{qty}}` | Jumlah item yang dibeli |

---

## Format Download Links (JSON Array)

Download links disimpan di field `download_links` produk tipe stok.

```json
[
  {
    "name": "File Utama ZIP",
    "url": "https://drive.google.com/file/d/XXXXXXXX/view"
  },
  {
    "name": "Dokumentasi PDF",
    "url": "https://drive.google.com/file/d/YYYYYYYY/view"
  },
  {
    "name": "Bonus — Video Tutorial",
    "url": "https://drive.google.com/file/d/ZZZZZZZZ/view"
  }
]
```

Semua link ini muncul sebagai tombol **Download** di email invoice yang diterima pembeli.

---

## Cara Menjalankan

### Development

```bash
cd digistore

# 1. Salin dan edit konfigurasi
cp backend/.env.example backend/.env
# Edit: ADMIN_PASSWORD, JWT_SECRET, SMTP_*

# 2. Install dependencies
cd backend && go mod tidy && cd ..
cd frontend && npm install && cd ..

# 3. Jalankan (dua terminal terpisah)
# Terminal 1:
cd backend && go run main.go

# Terminal 2:
cd frontend && npm run dev
```

Akses:
- Toko: http://localhost:5173
- Admin: http://localhost:5173/login  (default: admin / admin123)
- API: http://localhost:8080/api

### Production (Docker)

```bash
# 1. Setup SSL
mkdir ssl
certbot certonly --standalone -d yourdomain.com
cp /etc/letsencrypt/live/yourdomain.com/fullchain.pem ssl/
cp /etc/letsencrypt/live/yourdomain.com/privkey.pem ssl/

# 2. Edit konfigurasi
nano backend/.env
# Ubah: ADMIN_PASSWORD, JWT_SECRET, SMTP_*, FRONTEND_URL=https://yourdomain.com

# 3. Update domain di nginx.conf (ganti "yourdomain.com")

# 4. Build dan jalankan
docker compose up -d --build

# Cek log:
docker compose logs -f
```

---

## Konfigurasi SMTP (Email)

Edit `backend/.env`:

```env
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=emailkamu@gmail.com
SMTP_PASS=xxxx_xxxx_xxxx_xxxx   # Gmail: pakai App Password, bukan password biasa
SMTP_FROM=DigiStore <noreply@yourdomain.com>
```

**Cara buat Gmail App Password:**
1. Buka https://myaccount.google.com/apppasswords
2. Pilih Mail → Other (Custom name)
3. Copy 16 karakter yang muncul → paste ke SMTP_PASS

Jika SMTP kosong, email otomatis di-skip tapi semua detail tetap muncul di terminal log — berguna saat development.

---

## API Endpoints

### Public (tanpa auth)

| Method | Endpoint | Fungsi |
|--------|----------|--------|
| POST | `/api/auth/login` | Login admin, dapat JWT token |
| GET | `/api/products` | Daftar produk aktif (tipe stok saja) |
| GET | `/api/products/:id` | Detail satu produk |
| POST | `/api/orders` | Buat pesanan baru |
| GET | `/api/payment/config` | Info pembayaran (bank, ewallet) |

Body POST `/api/orders`:
```json
{
  "product_id": 1,
  "buyer_name": "Andi Pratama",
  "buyer_email": "andi@gmail.com",
  "qty": 1,
  "pay_method": "bank"
}
```

### Admin (header: Authorization: Bearer <token>)

| Method | Endpoint | Fungsi |
|--------|----------|--------|
| GET | `/api/admin/dashboard` | Statistik & pesanan terbaru |
| GET | `/api/admin/products?admin=1` | Semua produk (termasuk nonaktif) |
| POST | `/api/admin/products` | Tambah produk baru |
| PUT | `/api/admin/products/:id` | Edit produk |
| DELETE | `/api/admin/products/:id` | Hapus produk |
| PATCH | `/api/admin/products/:id/toggle` | Toggle aktif/nonaktif |
| GET | `/api/admin/orders` | Semua pesanan |
| PATCH | `/api/admin/orders/:id/status` | Update status pesanan |
| GET | `/api/admin/payment/config` | Lihat konfigurasi pembayaran |
| PUT | `/api/admin/payment/config` | Update konfigurasi pembayaran |
| GET | `/api/admin/scripts/logs` | Riwayat eksekusi script |

---

## Struktur File

```
digistore/
├── backend/
│   ├── config/config.go          Baca .env ke struct Config
│   ├── database/db.go            Init SQLite + GORM + seed data
│   ├── models/models.go          Definisi tabel: Product, Order, ScriptLog, dll
│   ├── handlers/
│   │   ├── auth.go               Login → JWT token
│   │   ├── products.go           CRUD produk
│   │   ├── orders.go             Buat order, kurangi stok, trigger script/email
│   │   ├── payment.go            Baca/update konfigurasi pembayaran
│   │   └── dashboard.go          Statistik + recent orders
│   ├── middleware/auth.go        Validasi JWT untuk route admin
│   ├── email/invoice.go          Kirim email invoice HTML (dengan link download)
│   ├── scripts/executor.go       Parse & eksekusi JSON script actions
│   ├── main.go                   Entry point + definisi semua route
│   └── .env.example
│
├── frontend/src/
│   ├── lib/
│   │   ├── api.js                HTTP client + auth token store (Svelte writable)
│   │   └── utils.js              IDR formatter, tanggal, label status
│   └── routes/
│       ├── +page.svelte          Halaman toko (grid produk + filter)
│       ├── BuyModal.svelte       Form nama, email, jumlah
│       ├── CheckoutModal.svelte  Pilih metode bayar + konfirmasi
│       ├── InvoiceModal.svelte   Tampilkan invoice setelah order
│       ├── login/+page.svelte    Form login admin
│       └── admin/
│           ├── +layout.svelte    Sidebar navigasi admin
│           ├── +page.svelte      Dashboard statistik
│           ├── products/         CRUD produk + editor link download & script
│           ├── orders/           Tabel pesanan + update status
│           ├── payment/          Konfigurasi metode pembayaran
│           └── scripts/          Riwayat eksekusi script actions
│
├── docker-compose.yml            Orkestrasi backend + frontend + nginx
├── nginx.conf                    Reverse proxy + SSL
└── Makefile                      Shortcut perintah dev & deploy
```
