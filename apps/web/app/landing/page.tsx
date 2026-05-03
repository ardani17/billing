import type { Metadata } from "next";
import LandingPageClient from "./LandingPageClient";

export const metadata: Metadata = {
  title: "ISPBoss - Platform Billing & Manajemen Jaringan untuk ISP",
  description:
    "Kelola billing, MikroTik, OLT, dan jaringan FTTH dari satu dashboard. Coba gratis 3 hari.",
  openGraph: {
    title: "ISPBoss - Platform Billing & Manajemen Jaringan untuk ISP",
    description:
      "Kelola billing, MikroTik, OLT, dan jaringan FTTH dari satu dashboard.",
    type: "website",
    locale: "id_ID",
  },
};

export default function LandingPage() {
  return <LandingPageClient />;
}
