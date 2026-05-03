import type { NextConfig } from "next";

// Konfigurasi Next.js untuk aplikasi web ISPBoss.
const nextConfig: NextConfig = {
  transpilePackages: ["@ispboss/ui", "@ispboss/types"],
};

export default nextConfig;
