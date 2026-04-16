# DigiStore Copilot Instructions

## Architecture Overview
DigiStore is a digital product marketplace with three product types:
- **STOK**: Digital downloads with automatic stock management and email delivery
- **SCRIPT**: Service-based products that execute automated actions (email, webhook, log)
- **PROVIDER**: Reselling from external APIs (e.g., KoalaStore) with markup pricing

Backend: Go/Gin/GORM/SQLite | Frontend: SvelteKit | Deployment: Docker + Nginx

## Key Patterns & Conventions

### Product Types & Delivery Logic
- **STOK products**: Store download links in `ProductStock.Data` as JSON array. On order, claim unsold stock items and email links.
- **SCRIPT products**: Store actions in `Product.Script` as JSON array. Execute via `scripts.Execute()` with template variables like `{{buyer_email}}`.
- **PROVIDER products**: Sync stock/prices from external APIs. Orders placed directly to provider on purchase.

Example script action (from `backend/scripts/executor.go`):
```json
{"type": "email", "to": "team@example.com", "subject": "New order {{invoice_no}}", "body": "Client: {{buyer_name}}"}
```

### Order Flow
Orders start as "pending". For manual payments: deliver immediately and mark "paid". For gateways: wait for webhook, then deliver.

### Database Models
- `Product`: Core product with type-specific fields
- `ProductStock`: Individual stock items for STOK products
- `Order`: Purchase records with gateway integration fields
- `ScriptLog`: Execution results for SCRIPT actions

### API Structure
- Public routes: `/api/products`, `/api/orders`
- Admin routes: `/api/admin/*` with JWT middleware
- Webhooks: `/api/webhook/{gateway}` for payment confirmations

### Development Workflow
- **Start dev**: `make dev` (runs backend on :8080, frontend on :5173)
- **Build**: `make build` (Go binary + Svelte build)
- **Deploy**: `docker compose up -d --build`
- **Seed config**: `cp backend/.env.example backend/.env` and configure SMTP/gateways

### Frontend Patterns
- Auth store in `frontend/src/lib/api.js` manages JWT tokens
- Modals for buy/checkout/invoice flow
- Admin panel uses SvelteKit routes under `/admin`

### Common Tasks
- **Add product type**: Update `models.Product`, handlers, and frontend forms
- **Integrate gateway**: Add webhook handler in `handlers/` and config in `PaymentConfig`
- **Modify delivery**: Edit `deliverOrder()` in `handlers/orders.go` based on product type

Reference: `README.md` for full setup, `backend/models/models.go` for schema, `backend/handlers/orders.go` for order logic.