# Build stage — Go 1.25 (pgx v5 요구). 빌드 컨텍스트 = 레포 루트.
FROM golang:1.25-alpine AS build

WORKDIR /src
# 캐시 효율: go.mod·sum만 먼저 복사
COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download

COPY apps/api/ ./

# 정적 바이너리 3개 (CGO 비활성).
# - server:   HTTP API + cron 워커 (+ 부팅 시 SeedIfEmpty)
# - backfill: 일회성 CLI (수동 전체 재백필) — `flyctl ssh console -C "/app/backfill"`
# - migrate:  release_command (배포 전 마이그레이션 적용)
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server   ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/backfill ./cmd/backfill
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/migrate  ./cmd/migrate

# Runtime stage — distroless (rootless, minimal)
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=build /out/server   /app/server
COPY --from=build /out/backfill /app/backfill
COPY --from=build /out/migrate  /app/migrate
# 마이그레이션 SQL을 이미지에 포함 (레포 루트 컨텍스트라 접근 가능). migrate가 --dir=/app/migrations로 읽음.
COPY supabase/migrations /app/migrations

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/server"]
