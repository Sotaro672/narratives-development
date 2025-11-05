// frontend/shell/vite.config.ts
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { federation } from "@module-federation/vite";
import mfOptions from "./module-federation.config";

export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),           // ✅ Tailwind v4 プラグイン（型整合済み）
    federation(mfOptions),   // ✅ Module Federation設定
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
