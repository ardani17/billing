import type { Config } from "tailwindcss";

// Konfigurasi Tailwind CSS untuk aplikasi web ISPBoss.
const config: Config = {
  content: [
    "./app/**/*.{ts,tsx}",
    "./src/**/*.{ts,tsx}",
    "../../packages/ui/src/**/*.{ts,tsx}",
  ],
};

export default config;
