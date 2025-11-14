// frontend/shell/vite.config.ts
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { federation } from "@module-federation/vite";
import mfOptions from "./module-federation.config";

// ============================================================================
// Vite 設定ファイル
// Firebase Hosting 対応版
// ============================================================================
// - base: "/" により SPA ルーティングを正しく処理
// - outDir: "dist" → Firebase "public": "shell/dist" と一致させる
// - Module Federation, Tailwind v4, React18 に対応
// ============================================================================

export default defineConfig({
  base: "/", // ✅ Firebase Hosting / BrowserRouter 対応（重要）
  plugins: [
    react(),
    tailwindcss(),          // ✅ Tailwind v4 plugin
    federation(mfOptions),  // ✅ Module Federation 設定
  ],
  server: {
    port: 4000,
    open: true,
  },
  build: {
    target: "esnext",
    modulePreload: false,
    outDir: "dist",         // ✅ ビルド出力先 (Hostingのpublic=shell/distと一致)
    emptyOutDir: true,
  },
  // Firebase など外部環境変数を import.meta.env.* で扱う
  define: {
    "process.env": {}, // Nodeのprocess参照エラー防止 (Viteでの安全策)
  },
});
