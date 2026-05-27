# Quotient — Portfolio Intelligence Terminal

개인 운영 금융 SaaS. 개발자·파워유저 타겟. 한국·미국 자산 통합 분석 대시보드 + 자연어 분석가 인터페이스.
블룸버그 터미널의 정보 밀도·미감을 개인 가격대로.

## 빠른 네비게이션

- [`docs/STATUS.md`](docs/STATUS.md) — 현재 어디까지 구현됐나
- [`docs/ROADMAP.md`](docs/ROADMAP.md) — 다음 작업은 무엇인가
- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — 시스템 구성 및 핵심 설계 결정 (Why 포함)
- [`docs/AGENTS.md`](docs/AGENTS.md) — 에이전트 팀 구성 및 디스패치 규칙
- [`docs/DEPLOY.md`](docs/DEPLOY.md) — Supabase·Fly·Vercel·Sentry·PostHog·GitHub Actions 배포 가이드
- [`docs/E2E_SMOKE.md`](docs/E2E_SMOKE.md) — 배포 직후 골든패스 수동 스모크 시나리오
- [`docs/superpowers/specs/`](docs/superpowers/specs/) — 기능별 상세 설계 문서
- [`docs/superpowers/plans/`](docs/superpowers/plans/) — 기능별 구현 계획

## 운영 원칙

- **언어**: 모든 문서·커밋·UI 카피는 한국어 우선 (코드 식별자·기술 용어 제외).
- **소유 모델**: 개인 사업자 1인 운영. 법인 없음. 한국 금융 규제 위반 가능성이 있는 기능 금지 (마이데이터, 자금 보관·이체, 직접 자문 추천 등).
- **의사결정 페르소나**: 의견을 묻는 모드가 아니라 "이 방향으로 판단했고 이유는 ~. 승인 부탁"이 기본. 결정은 CEO·CTO·CFO 셋 중 관련 페르소나로 명시.

## superpowers 의무

`superpowers` 플러그인이 없으면 작업을 시작하지 않는다. 즉시 사용자에게 설치를 요청한다.

| 작업 유형 | 스킬 |
|---|---|
| 새 안건·기능 구상 | `superpowers:brainstorming` |
| 코드 작성·디버깅 | `superpowers:requesting-code-review`, `superpowers:systematic-debugging` |
| 코드 개발 (구현) | `superpowers:test-driven-development`, `superpowers:subagent-driven-development` |
| UI/UX 작업 | `plan-design-review` |
| 계획 작성 | `superpowers:writing-plans` → 작성 후 **subagent로 자체 검토** |

## 스펙 작성 규칙 (MANDATORY)

`docs/superpowers/specs/*.md` 작성 직후 자체 검토 사이클:

1. Critical / Important / Minor로 이슈 분류
   - Critical: 명세대로 구현 시 동작 안 함 (race, 잘못된 API, chunk 경계)
   - Important: 리소스 누수·비효율 패턴·에러 핸들링 누락
   - Minor: 명확성·비범위 명시 부족·예시 코드 누락
2. 우선순위별로 사용자에게 보고
3. 스펙 파일 직접 패치
4. 스펙 하단 "검토 이력" 섹션 갱신

별도 요청 없어도 작성→검토→패치→보고가 한 사이클.

## 계획 작성 규칙 (MANDATORY)

`docs/superpowers/plans/*.md`는 작성 후 **subagent로 자체 검토**를 거친 다음 사용자에게 알린다. 직접 검토는 금지.

## 문서 업데이트 규칙 (MANDATORY)

기능 구현 완료 시:

1. `docs/STATUS.md` — 해당 항목 ✅로 이동, "최근 변경 이력" 맨 위에 한 줄 추가, "마지막 업데이트" 갱신
2. `docs/ROADMAP.md` — 완료 항목 제거, "현재 추천 다음 작업" 재설정
3. `docs/ARCHITECTURE.md` — 아키텍처 영향 변경에만 반영 (새 컴포넌트·중대 설계 결정, Why/How 필수)

문서 업데이트 없이는 작업이 완료된 것으로 간주하지 않는다.

## 하네스 엔지니어링 규칙 (MANDATORY)

작업 중 발견한 규칙·판단 기준·결정은 반드시 파일에 기록한다. 메모리·대화 맥락에만 두는 것은 금지.

| 성격 | 기록 위치 |
|---|---|
| 프로젝트 전반 작업 규칙 | 이 `CLAUDE.md` |
| 아키텍처 설계 결정 | `docs/ARCHITECTURE.md` "핵심 설계 결정" (Why/How 필수) |
| 작업 흐름·문서 관리 규칙 | 해당 문서 하단 "업데이트 규칙" 주석 |
| 기능별 상세 규칙·트레이드오프 | `docs/superpowers/specs/<feature>-design.md` |
| 알려진 결함·미구현 이슈 | `docs/STATUS.md` "알려진 결함" + `docs/ROADMAP.md` |

기록 흐름:

1. "이 결정은 다음 세션에도 유효한가?" 판단 → 위치 선택
2. 임시 메모가 아닌 **명시적 섹션**으로 추가
3. 같은 커밋에 코드·문서 함께
4. `CLAUDE.md`의 "빠른 네비게이션"에도 신규 문서 경로 추가

---
이 파일은 새 세션 시작 시 자동 로드되어 프로젝트 컨텍스트를 제공한다. 규칙 변경 시 같은 커밋에 사유를 명시한다.
