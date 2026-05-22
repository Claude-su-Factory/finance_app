# Quotient — Portfolio Intelligence Terminal

개인 운영 금융 SaaS. 한국·미국 자산 통합 분석 + 자연어 분석가 인터페이스.

[설계 문서 →](docs/superpowers/specs/2026-05-22-quotient-mvp-design.md)

## 로컬 개발

### 사전 요구
- Go 1.23+
- Node 20+
- Supabase CLI: `brew install supabase/tap/supabase`
- Air (Go 핫리로드): `go install github.com/air-verse/air@latest`

### 셋업
1. `.env.example`을 `.env.local`로 복사하고 값 채우기
2. `make db-up` — 로컬 Supabase 띄우기
3. `make migrate` — 마이그레이션 실행
4. 두 터미널: `make api`, `make web`
5. http://localhost:3000 접속

### 문서
- [STATUS](docs/STATUS.md), [ROADMAP](docs/ROADMAP.md), [ARCHITECTURE](docs/ARCHITECTURE.md), [AGENTS](docs/AGENTS.md)
