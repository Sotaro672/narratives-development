import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import federation from "@module-federation/vite";
import mfConfig from "./module-federation.config";

export default defineConfig({
  plugins: [
    react(),
    federation(mfConfig), // Module Federation 設定を適用
  ],

  server: {
    port: 4014, // ✅ announce用ポート
    open: false,
    cors: true, // ✅ shell からのアクセス許可
  },

  build: {
    target: "esnext", // ✅ 最新ブラウザ向け（Module Federation想定）
    outDir: "dist",
  },

  resolve: {
    alias: {
      "@": "/src", // ✅ srcパス短縮
    },
  },

  optimizeDeps: {
    include: ["react", "react-dom", "react-router-dom"],
  },
});
