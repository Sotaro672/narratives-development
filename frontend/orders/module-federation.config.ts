import { defineConfig } from "@module-federation/vite";
import pkgJson from "../../package.json" assert { type: "json" };

/**
 * Remote: orders
 * ------------------------------------------------------------------------
 * 注文管理モジュール（Solid State Console のサブアプリ）
 * exposes: "./routes" をホスト（shell）に公開し、
 * import("orders/routes") で遅延ロード可能にする。
 * ------------------------------------------------------------------------
 */
export default defineConfig({
  // ───────────────────────────────────────────────
  // name: ホスト側 remotes キーと一致させる
  // ───────────────────────────────────────────────
  name: "orders",

  // ───────────────────────────────────────────────
  // exposes: shell が import("orders/routes") で利用するエントリポイント
  // ───────────────────────────────────────────────
  exposes: {
    "./routes": "./src/routes.tsx",
  },

  // ───────────────────────────────────────────────
  // shared: shell と依存関係を共有（React系をシングルトン化）
  // ───────────────────────────────────────────────
  shared: {
    react: {
      singleton: true,
      requiredVersion: (pkgJson as any).dependencies["react"],
    },
    "react-dom": {
      singleton: true,
      requiredVersion: (pkgJson as any).dependencies["react-dom"],
    },
    "react-router-dom": {
      singleton: true,
      requiredVersion: (pkgJson as any).dependencies["react-router-dom"],
    },
    "@tanstack/react-query": {
      singleton: true,
      requiredVersion: (pkgJson as any).dependencies["@tanstack/react-query"],
    },
    "@apollo/client": {
      singleton: true,
      requiredVersion: (pkgJson as any).dependencies["@apollo/client"],
    },
  },

  // ───────────────────────────────────────────────
  // filename: ホストがロードするリモートエントリ名
  // ───────────────────────────────────────────────
  filename: "remoteEntry.js",

  // ───────────────────────────────────────────────
  // その他オプション
  // ───────────────────────────────────────────────
  manifest: true, // build時に manifest.json を生成
  dts: false,     // 型定義ファイルを出力しない
});
