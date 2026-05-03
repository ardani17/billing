import type { Metadata } from "next";
import BrandingSettingsClient from "./BrandingSettingsClient";

export const metadata: Metadata = {
  title: "White Label / Branding - ISPBoss",
  description: "Pengaturan logo, warna, domain, invoice, dan walled garden tenant.",
};

export default function BrandingSettingsPage() {
  return <BrandingSettingsClient />;
}
