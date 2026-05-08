// frontend/shell/vite.config.ts
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
  ],
  server: {
    port: 4000,
    open: true,
  },
  build: {
    target: "esnext",
    modulePreload: false,
    outDir: "dist",
  },
});