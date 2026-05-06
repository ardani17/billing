import type { MetadataRoute } from "next";
import { landingDescription, siteName } from "./lib/seo";

export default function manifest(): MetadataRoute.Manifest {
  return {
    id: "/",
    name: `${siteName} - Platform Billing ISP`,
    short_name: siteName,
    description: landingDescription,
    start_url: "/",
    scope: "/",
    display: "standalone",
    background_color: "#ffffff",
    theme_color: "#2563eb",
    icons: [
      {
        src: "/icon",
        sizes: "64x64",
        type: "image/png",
      },
      {
        src: "/apple-icon",
        sizes: "180x180",
        type: "image/png",
      },
    ],
  };
}
