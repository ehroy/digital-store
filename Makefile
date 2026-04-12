.PHONY: dev backend frontend build docker clean

# ── Development ──────────────────────────────────────────────────
dev:
	@make -j2 backend frontend

backend:
	@echo "▶  Starting Go backend on :8080"
	@cd backend && go run main.go

frontend:
	@echo "▶  Starting Svelte frontend on :5173"
	@cd frontend && npm run dev

# ── Install dependencies ─────────────────────────────────────────
install:
	@echo "▶  Installing Go dependencies…"
	@cd backend && go mod tidy
	@echo "▶  Installing Node dependencies…"
	@cd frontend && npm install

# ── Production build ─────────────────────────────────────────────
build-backend:
	@cd backend && CGO_ENABLED=1 go build -ldflags="-s -w" -o digistore ./main.go
	@echo "✅ Backend binary: backend/digistore"

build-frontend:
	@cd frontend && npm run build
	@echo "✅ Frontend build: frontend/build/"

build: build-backend build-frontend

# ── Docker ───────────────────────────────────────────────────────
docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

# ── Utilities ─────────────────────────────────────────────────────
clean:
	@rm -f backend/digistore backend/digistore.db
	@rm -rf frontend/build frontend/.svelte-kit

seed-env:
	@cp backend/.env.example backend/.env
	@echo "✅ .env dibuat dari .env.example — edit sebelum dijalankan!"
