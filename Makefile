.PHONY: build run dev test migrate seed tidy

# backend
build:
	go build -o store ./cmd/store

tidy:
	go mod tidy && cd web && bun install

run: build
	# dev: login-as aktif + SMTP ke Mailpit (UI http://localhost:8025)
	STORE_ADDR=:8080 STORE_DEV_LOGIN=1 STORE_SMTP_HOST=127.0.0.1 STORE_SMTP_PORT=1025 STORE_SMTP_FROM=dev@toko.local ./store serve

migrate: build
	./store migrate

seed: build
	./store seed

# frontend (dev: vite :5173 proxy ke :8080)
dev:
	cd web && bun run dev

web-build:
	cd web && bun run build

# verifikasi lengkap
test:
	go vet ./...
	go test ./...
	cd web && bun run typecheck && bun run build
