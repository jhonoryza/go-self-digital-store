# Backend Go + frontend ter-build (shell fallback + aset) dalam satu image.
# Frontend produksi utama di Cloudflare Pages — build di sana; image ini hanya
# butuh web/dist untuk mode shell-Go (dev/single-origin).
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /store ./cmd/store

FROM oven/bun:1-alpine AS webbuild
WORKDIR /web
COPY web/package.json web/bun.lock* ./
RUN bun install
COPY web ./
RUN bun run build

FROM alpine:3.21
RUN apk add --no-cache ca-certificates && adduser -D -u 1000 store
WORKDIR /app
COPY --from=build /store /app/store
COPY --from=webbuild /web/dist /app/web/dist
RUN mkdir -p /app/data && chown -R store:store /app
USER store
EXPOSE 8080
VOLUME /app/data
ENTRYPOINT ["/app/store"]
CMD ["serve"]
