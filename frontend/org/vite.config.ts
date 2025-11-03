import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import federation from "@module-federation/vite";
import mfConfig from "./module-federation.config";

/**
 * Vite Configuration for Org Remote Module
 * ------------------------------------------------------------------------
 * - port: 4001
 * - React + TypeScript + TailwindCSS 対応
 * - Module Federation（@module-federation/vite）で shell に接続
 * ------------------------------------------------------------------------
 */
export default defineConfig({
  plugins: [
    react(),
    federation(mfConfig), // Module Federation 設定を統合
  ],

  // ───────────────────────────────────────────────
  // 開発サーバー設定
  // ───────────────────────────────────────────────
  server: {
    port: 4001,   // shell 側 remotes と一致させる
    open: false,  // 自動ブラウザオープン無効
    cors: true,   // 跨ドメイン読み込み許可（shellからの読み込み用）
  },

  // ───────────────────────────────────────────────
  // ビルド設定
  // ───────────────────────────────────────────────
  build: {
    target: "esnext", // 最新構文で出力
    outDir: "dist",
    modulePreload: false,
    minify: true,
  },

  // ───────────────────────────────────────────────
  // resolve 設定（srcエイリアスなどをサポート）
  // ───────────────────────────────────────────────
  resolve: {
    alias: {
      "@": "/src",
    },
  },

  // ───────────────────────────────────────────────
  // optimizeDeps 設定（React系依存を明示）
  // ───────────────────────────────────────────────
  optimizeDeps: {
    include: [
      "react",
      "react-dom",
      "react-router-dom",
      "@tanstack/react-query",
      "@apollo/client",
    ],
  },
});
