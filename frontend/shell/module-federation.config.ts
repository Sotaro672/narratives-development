import { defineConfig } from "@module-federation/vite";
import pkg from "../../package.json";

/**
 * Host: Solid State Console (CRM)
 * This app aggregates multiple remote modules (org, inquiries, listings, etc.)
 * using Vite's Module Federation plugin.
 */
export default defineConfig({
  name: "shell",

  // ───────────────────────────────────────────────
  // remotes: 各機能モジュールを登録
  // （Vite dev server URL または本番build時のリモートURL）
  // ───────────────────────────────────────────────
  remotes: {
    org: "http://localhost:4001/assets/remoteEntry.js",
    inquiries: "http://localhost:4002/assets/remoteEntry.js",
    listings: "http://localhost:4003/assets/remoteEntry.js",
    operations: "http://localhost:4004/assets/remoteEntry.js",
    preview: "http://localhost:4005/assets/remoteEntry.js",
    production: "http://localhost:4006/assets/remoteEntry.js",
    design: "http://localhost:4007/assets/remoteEntry.js",
    mint: "http://localhost:4008/assets/remoteEntry.js",
    orders: "http://localhost:4009/assets/remoteEntry.js",
    ads: "http://localhost:4010/assets/remoteEntry.js",
    accounts: "http://localhost:4011/assets/remoteEntry.js",
    transactions: "http://localhost:4012/assets/remoteEntry.js",

    // 新規追加 ───────────────────────────────
    message: "http://localhost:4013/assets/remoteEntry.js",
    announce: "http://localhost:4014/assets/remoteEntry.js",
  },

  // ───────────────────────────────────────────────
  // shared: 全アプリ間で単一インスタンスとして共有する依存パッケージ
  // （バージョン競合を防止し、状態共有を可能にする）
  // ───────────────────────────────────────────────
  shared: {
    react: {
      singleton: true,
      requiredVersion: pkg.dependencies["react"],
    },
    "react-dom": {
      singleton: true,
      requiredVersion: pkg.dependencies["react-dom"],
    },
    "react-router-dom": {
      singleton: true,
      requiredVersion: pkg.dependencies["react-router-dom"],
    },
    "@tanstack/react-query": {
      singleton: true,
      requiredVersion: pkg.dependencies["@tanstack/react-query"],
    },
    "@apollo/client": {
      singleton: true,
      requiredVersion: pkg.dependencies["@apollo/client"],
    },
  },

  // ───────────────────────────────────────────────
  // exposes: shellは他アプリへコンポーネントを公開しない（ホスト専用）
  // ───────────────────────────────────────────────
  exposes: {},

  // ───────────────────────────────────────────────
  // filename: 出力ファイル名（ホスト自身のremoteEntry定義）
  // ───────────────────────────────────────────────
  filename: "remoteEntry.js",

  // ───────────────────────────────────────────────
  // options: 開発・本番共通設定
  // ───────────────────────────────────────────────
  dts: false, // 型定義出力を無効（必要に応じてtrue）
  manifest: true, // build時にmanifest.jsonを生成
});
