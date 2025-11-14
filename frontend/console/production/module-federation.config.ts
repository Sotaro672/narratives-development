import { defineConfig } from "@module-federation/vite";
import pkgJson from "../../package.json" assert { type: "json" };

/**
 * Remote: production
 * ------------------------------------------------------------------------
 * 生産計画・進捗管理モジュール（Solid State Console のサブアプリ）
 * exposes: "./routes" をホスト（shell）に公開し、
 * import("production/routes") で遅延ロード可能にする。
 * ------------------------------------------------------------------------
 */
export default defineConfig({
  // ───────────────────────────────────────────────
  // name: ホスト（shell）の remotes キーと一致させる
  // ───────────────────────────────────────────────
  name: "production",

  // ───────────────────────────────────────────────
  // exposes: ホストが読み込む公開エントリポイント
  // ───────────────────────────────────────────────
  exposes: {
    "./routes": "./src/routes.tsx",
  },

  // ───────────────────────────────────────────────
  // shared: ホストと共有する依存パッケージ
  // （React・Router・React Query・Apolloなどをシングルトン共有）
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
  // filename: ホストが読み込むリモートエントリ名
  // ───────────────────────────────────────────────
  filename: "remoteEntry.js",

  // ───────────────────────────────────────────────
  // その他オプション
  // ───────────────────────────────────────────────
  manifest: true, // build時に manifest.json を生成
  dts: false,     // 型定義ファイルを出力しない
});
