# ── Build stage ────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o digistore ./main.go

# ── Run stage ───────────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata
RUN cp /usr/share/zoneinfo/Asia/Jakarta /etc/localtime

WORKDIR /app
COPY --from=builder /build/digistore .

RUN mkdir -p /app/data
EXPOSE 8080

CMD ["./digistore"]
