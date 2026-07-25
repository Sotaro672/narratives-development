// frontend/console/shell/vite.config.ts

import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  base: "/",

  plugins: [
    react(),
    tailwindcss(),
  ],

  resolve: {
    dedupe: [
      "react",
      "react-dom",
      "react-router",
      "react-router-dom",
      "lucide-react",
      "@tanstack/react-query",
      "@apollo/client",
      "firebase",
      "qrcode",
      "react-color",
      "pdf-lib",
    ],
  },

  server: {
    port: 4000,
    open: true,
  },

  build: {
    target: "esnext",
    modulePreload: false,
    outDir: "dist",
    emptyOutDir: true,
  },

  define: {
    "process.env": {},
  },
});