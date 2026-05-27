# Build stage — Go 1.25 (pgx v5 요구).
FROM golang:1.25-alpine AS build

WORKDIR /src
# 캐시 효율: go.mod·sum만 먼저 복사
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 단일 정적 바이너리 (CGO 비활성). server 한 개만 빌드(backfill은 CLI라 운영 컨테이너에 불필요).
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# Runtime stage — distroless (rootless, minimal)
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=build /out/server /app/server

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/server"]
