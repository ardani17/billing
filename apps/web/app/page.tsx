import type { Metadata } from "next";
import LandingPageClient from "./landing/LandingPageClient";
import {
  createLandingMetadata,
  faqJsonLd,
  organizationJsonLd,
  softwareApplicationJsonLd,
  stringifyJsonLd,
  websiteJsonLd,
} from "./lib/seo";

export const metadata: Metadata = createLandingMetadata("/");

export default function Home() {
  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: stringifyJsonLd([
            organizationJsonLd,
            websiteJsonLd,
            softwareApplicationJsonLd,
            faqJsonLd,
          ]),
        }}
      />
      <LandingPageClient />
    </>
  );
}
