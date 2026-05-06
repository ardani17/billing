import type { Metadata } from "next";
import { landingFaqs } from "../landing/content";

const FALLBACK_SITE_URL = "https://ispboss.id";

function normalizeSiteUrl(value?: string) {
  if (!value) {
    return FALLBACK_SITE_URL;
  }

  try {
    const parsed = new URL(value);
    if (["localhost", "127.0.0.1", "0.0.0.0"].includes(parsed.hostname)) {
      return FALLBACK_SITE_URL;
    }

    return parsed.origin;
  } catch {
    return FALLBACK_SITE_URL;
  }
}

export const siteUrl = normalizeSiteUrl(process.env.NEXT_PUBLIC_SITE_URL);
export const siteName = "ISPBoss";
export const landingTitle =
  "ISPBoss - Platform Billing dan Manajemen Jaringan untuk ISP";
export const landingDescription =
  "Kelola billing ISP, pelanggan, invoice, pembayaran, MikroTik, OLT, notifikasi, dan peta FTTH dari satu dashboard. Coba gratis 3 hari.";

export function absoluteUrl(path = "/") {
  return new URL(path, siteUrl).toString();
}

export function createLandingMetadata(path = "/"): Metadata {
  const canonical = absoluteUrl(path);
  const image = absoluteUrl("/opengraph-image");

  return {
    title: {
      absolute: landingTitle,
    },
    description: landingDescription,
    alternates: {
      canonical,
    },
    robots: {
      index: true,
      follow: true,
    },
    openGraph: {
      title: landingTitle,
      description: landingDescription,
      url: canonical,
      siteName,
      type: "website",
      locale: "id_ID",
      images: [
        {
          url: image,
          width: 1200,
          height: 630,
          alt: "ISPBoss platform billing ISP dan manajemen jaringan",
        },
      ],
    },
    twitter: {
      card: "summary_large_image",
      title: landingTitle,
      description: landingDescription,
      images: [image],
    },
  };
}

export function createPageMetadata({
  title,
  description,
  path,
}: {
  title: string;
  description: string;
  path: string;
}): Metadata {
  const fullTitle = `${title} - ${siteName}`;
  const canonical = absoluteUrl(path);

  return {
    title: {
      absolute: fullTitle,
    },
    description,
    alternates: {
      canonical,
    },
    openGraph: {
      title: fullTitle,
      description,
      url: canonical,
      siteName,
      type: "website",
      locale: "id_ID",
      images: [
        {
          url: absoluteUrl("/opengraph-image"),
          width: 1200,
          height: 630,
          alt: "ISPBoss platform billing ISP dan manajemen jaringan",
        },
      ],
    },
    twitter: {
      card: "summary_large_image",
      title: fullTitle,
      description,
      images: [absoluteUrl("/opengraph-image")],
    },
  };
}

export const organizationJsonLd = {
  "@context": "https://schema.org",
  "@type": "Organization",
  "@id": absoluteUrl("/#organization"),
  name: siteName,
  url: absoluteUrl("/"),
  logo: absoluteUrl("/icon"),
  sameAs: [absoluteUrl("/")],
};

export const websiteJsonLd = {
  "@context": "https://schema.org",
  "@type": "WebSite",
  "@id": absoluteUrl("/#website"),
  name: siteName,
  url: absoluteUrl("/"),
  publisher: {
    "@id": absoluteUrl("/#organization"),
  },
};

export const softwareApplicationJsonLd = {
  "@context": "https://schema.org",
  "@type": "SoftwareApplication",
  "@id": absoluteUrl("/#software"),
  name: siteName,
  applicationCategory: "BusinessApplication",
  operatingSystem: "Web",
  description: landingDescription,
  url: absoluteUrl("/"),
  publisher: {
    "@id": absoluteUrl("/#organization"),
  },
  offers: {
    "@type": "Offer",
    price: "150000",
    priceCurrency: "IDR",
    availability: "https://schema.org/InStock",
  },
};

export const faqJsonLd = {
  "@context": "https://schema.org",
  "@type": "FAQPage",
  mainEntity: landingFaqs.map((faq) => ({
    "@type": "Question",
    name: faq.q,
    acceptedAnswer: {
      "@type": "Answer",
      text: faq.a,
    },
  })),
};

export function stringifyJsonLd(data: unknown) {
  return JSON.stringify(data).replace(/</g, "\\u003c");
}
