FROM node:24-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY web/ ./
RUN npm run build

FROM golang:1.26-alpine AS api
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY migrations ./migrations
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata wget \
    && adduser -D -H -u 10001 app \
    && mkdir -p /app/var \
    && chown app:app /app/var
WORKDIR /app
COPY --from=api /out/server ./server
COPY --from=web /web/dist ./web/dist
COPY data ./data
ENV APP_ENV=production \
    PORT=8080 \
    DATABASE_PATH=/app/var/energy.db \
    DATA_DIR=/app/data \
    STATIC_DIR=/app/web/dist
USER app
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s CMD wget -qO- http://127.0.0.1:${PORT}/api/health || exit 1
CMD ["./server"]
