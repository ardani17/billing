import type { Metadata } from "next";
import { GeistSans } from "geist/font/sans";
import { GeistMono } from "geist/font/mono";
import { absoluteUrl, landingDescription, siteName, siteUrl } from "./lib/seo";
import "./globals.css";

export const metadata: Metadata = {
  metadataBase: new URL(siteUrl),
  applicationName: siteName,
  title: {
    default: siteName,
    template: `%s - ${siteName}`,
  },
  description: landingDescription,
  creator: siteName,
  publisher: siteName,
  formatDetection: {
    telephone: false,
    email: false,
    address: false,
  },
  icons: {
    icon: "/icon",
    apple: "/apple-icon",
  },
  manifest: "/manifest.webmanifest",
  openGraph: {
    title: siteName,
    description: landingDescription,
    url: absoluteUrl("/"),
    siteName,
    locale: "id_ID",
    type: "website",
    images: [
      {
        url: absoluteUrl("/opengraph-image"),
        width: 1200,
        height: 630,
        alt: "ISPBoss platform billing ISP dan manajemen jaringan",
      },
    ],
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="id" className={`${GeistSans.variable} ${GeistMono.variable}`}>
      <body className="font-sans antialiased">{children}</body>
    </html>
  );
}
