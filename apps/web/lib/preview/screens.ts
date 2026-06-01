// apps/web/lib/preview/screens.ts
export type PreviewScreen = { label: string; path: string };
export type PreviewGroup = { group: string; screens: PreviewScreen[] };

export const PREVIEW_SCREENS: PreviewGroup[] = [
  {
    group: "앱 화면",
    screens: [
      { label: "홈 대시보드", path: "/preview/home" },
      { label: "포트폴리오", path: "/preview/portfolio" },
      { label: "Paper 트레이딩", path: "/preview/paper" },
      { label: "마켓", path: "/preview/market" },
      { label: "매매 일기", path: "/preview/journal" },
      { label: "백테스트", path: "/preview/backtest" },
      { label: "AI 채팅", path: "/preview/chat" },
    ],
  },
  {
    group: "모달",
    screens: [
      { label: "매수/매도 (Paper)", path: "/preview/modals/trade" },
      { label: "Paper 초기화", path: "/preview/modals/reset" },
      { label: "매매일기 작성", path: "/preview/modals/journal-new" },
      { label: "종목 추가", path: "/preview/modals/holding-add" },
      { label: "보유 수정", path: "/preview/modals/holding-edit" },
      { label: "보유 삭제 확인", path: "/preview/modals/holding-delete" },
    ],
  },
  {
    group: "오버레이",
    screens: [{ label: "온보딩 마법사", path: "/preview/onboarding" }],
  },
  {
    group: "공개 페이지(미리보기 밖)",
    screens: [
      { label: "랜딩", path: "/" },
      { label: "요금제", path: "/pricing" },
      { label: "로그인", path: "/login" },
      { label: "회원가입", path: "/signup" },
      { label: "비밀번호 찾기", path: "/forgot-password" },
      { label: "개인정보처리방침", path: "/privacy" },
      { label: "이용약관", path: "/terms" },
      { label: "이메일 인증", path: "/verify-email" },
    ],
  },
];
