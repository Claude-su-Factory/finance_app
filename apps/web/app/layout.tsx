import type { Metadata } from "next";
import { Geist_Mono } from "next/font/google";
import localFont from "next/font/local";
import { PostHogProvider } from "@/components/analytics/PostHogProvider";
import "./globals.css";

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

// Pretendard Variable — pretendard 패키지의 woff2를 next/font/local로 self-host.
// CSS @import 대비 preload + display: swap + 빌드 최적화 적용 (FOIT 회피).
const pretendard = localFont({
  src: "../node_modules/pretendard/dist/web/variable/woff2/PretendardVariable.woff2",
  variable: "--font-pretendard",
  display: "swap",
  weight: "45 920", // variable font weight range
});

export const metadata: Metadata = {
  title: "Quotient — Portfolio Intelligence Terminal",
  description: "한국·미국 자산을 한 화면에. 자연어로 묻고, 즉시 분석을 받으세요.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="ko" className={`${geistMono.variable} ${pretendard.variable} dark`}>
      <body>
        <PostHogProvider>{children}</PostHogProvider>
      </body>
    </html>
  );
}
