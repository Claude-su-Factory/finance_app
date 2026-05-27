# Quotient

**개인 운영 금융 분석 SaaS.** 한국·미국 자산을 한 화면에 모아 시세를 보고, 자연어로 분석가에게 물어보듯 질문할 수 있습니다.

블룸버그 터미널의 정보 밀도·미감을 개인 가격대로.

## 무엇

- **포트폴리오 통합 관리** — 한국(KOSPI/KOSDAQ) + 미국(NASDAQ/NYSE) 자산을 한 곳에서 KRW 환산·손익·비중까지
- **AI 분석가** — Claude 기반 자연어 인터페이스. "내 포트폴리오 통화 분산 어때?" 같은 질문에 도구 호출로 실데이터 분석
- **마켓 모니터** — KOSPI·SPX·환율·금리 등 주요 지표 실시간(15분 지연) + 5년 일봉 차트
- **일일 브리핑** — 매일 아침 보유 자산·관심 종목·시장 변화 요약

## 누구를 위한

개발자·파워유저. 키보드 우선·정보 밀도 중시, 핀테크 UI(토스·뱅크샐러드)와 다른 결을 원하는 사람.

## 안 하는 것

- ❌ 마이데이터 연동, 자금 보관·이체 (개인 사업자 운영 → 규제 회피)
- ❌ 직접적인 매수/매도 추천 (분석 관점 제공만)
- ❌ 결제 기능 (Phase 2 사업자 등록 시 활성)

## 시작하기

로컬 구동·운영 배포 모두 [`docs/DEPLOY.md`](docs/DEPLOY.md) 한 문서에 정리.

```bash
git clone <repo>
cd finance
# 이후는 docs/DEPLOY.md Part A 참조
```

## 기술 스택

Go 1.25 (chi v5 + pgx v5) · Next.js 16 (Tailwind v4) · Supabase Postgres + RLS · Anthropic Claude · Fly.io + Vercel

## 문서

| 문서 | 용도 |
|---|---|
| [`docs/DEPLOY.md`](docs/DEPLOY.md) | 로컬 셋업 + 운영 배포 + 트러블슈팅 |
| [`docs/STATUS.md`](docs/STATUS.md) | 현재 어디까지 구현됐나 |
| [`docs/ROADMAP.md`](docs/ROADMAP.md) | 다음 작업 |
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | 핵심 설계 결정 (Why) |
| [`docs/E2E_SMOKE.md`](docs/E2E_SMOKE.md) | 배포 후 골든패스 검증 |

## 라이선스

학습·포트폴리오 용도로 공개. 운영 인스턴스의 사용자 데이터와는 별개.
