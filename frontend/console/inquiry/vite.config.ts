// frontend/inquiries/vite.config.ts
import { defineConfig, type PluginOption } from "vite";
import react from "@vitejs/plugin-react";
import { federation } from "@module-federation/vite";
import mfOptions from "./module-federation.config"; // ← 名前をmfOptionsに統一（shellと同じ）

export default defineConfig({
  plugins: [
    react(),
    federation(mfOptions) as unknown as PluginOption, // ← 型キャストでVite Pluginとして扱う
  ],
  server: {
    port: 4002, // shellのremotes設定と一致させる
    open: false,
  },
  build: {
    target: "esnext",
    outDir: "dist",
  },
});
