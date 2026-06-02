import type { MetadataRoute } from "next";
import { BRAND, SITE_NAME, SITE_TITLE_DEFAULT } from "@/lib/seo";

export default function manifest(): MetadataRoute.Manifest {
  return {
    name: SITE_TITLE_DEFAULT,
    short_name: SITE_NAME,
    description:
      "한국·미국 자산을 한 화면에. 자연어로 묻고, 즉시 분석을 받으세요.",
    start_url: "/",
    display: "standalone",
    background_color: BRAND.bg,
    theme_color: BRAND.bg,
    icons: [
      { src: "/icon", sizes: "32x32", type: "image/png" },
      { src: "/brand-512", sizes: "512x512", type: "image/png" },
    ],
  };
}
