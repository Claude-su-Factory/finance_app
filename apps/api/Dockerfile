# Build stage — Go 1.25 (pgx v5 요구).
FROM golang:1.25-alpine AS build

WORKDIR /src
# 캐시 효율: go.mod·sum만 먼저 복사
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 정적 바이너리 2개 (CGO 비활성).
# - server: HTTP API + cron 워커
# - backfill: 일회성 CLI (5년 가격 히스토리 등) — `flyctl ssh console -C "/app/backfill"`로 호출
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/backfill ./cmd/backfill

# Runtime stage — distroless (rootless, minimal)
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=build /out/server /app/server
COPY --from=build /out/backfill /app/backfill

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/server"]
