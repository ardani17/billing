import type { MetadataRoute } from "next";
import { absoluteUrl } from "./lib/seo";

export default function robots(): MetadataRoute.Robots {
  return {
    rules: {
      userAgent: "*",
      allow: [
        "/",
        "/landing",
        "/register",
        "/login",
        "/forgot-password",
        "/verify-email",
        "/contact",
        "/privacy",
        "/opengraph-image",
        "/twitter-image",
        "/icon",
        "/apple-icon",
      ],
      disallow: [
        "/api",
        "/dashboard",
        "/customers",
        "/packages",
        "/invoices",
        "/payments",
        "/reports",
        "/settings",
        "/super-admin",
        "/mikrotik",
        "/olt",
        "/network-map",
        "/cashflow",
        "/expenses",
        "/inventory",
        "/notifications",
        "/reseller",
        "/resellers",
        "/vouchers",
        "/walled-garden",
      ],
    },
    sitemap: absoluteUrl("/sitemap.xml"),
  };
}
