import { ImageResponse } from "next/og";
import { SquareMark } from "@/lib/brand-mark";

export function GET() {
  return new ImageResponse(<SquareMark size={512} />, {
    width: 512,
    height: 512,
  });
}
