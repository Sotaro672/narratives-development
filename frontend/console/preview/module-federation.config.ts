import { defineConfig } from "@module-federation/vite";
import pkgJson from "../../package.json" assert { type: "json" };

/**
 * Remote: preview
 * ------------------------------------------------------------------------
 * プレビュー管理モジュール（Solid State Console のサブアプリ）
 * exposes: "./routes" をホスト（shell）に公開し、
 * import("preview/routes") で遅延ロード可能にする。
 * ------------------------------------------------------------------------
 */
export default defineConfig({
  // ───────────────────────────────────────────────
  // name: ホスト側の remotes キーと一致させる
  // ───────────────────────────────────────────────
  name: "preview",

  // ───────────────────────────────────────────────
  // exposes: ホスト（shell）に公開するエントリポイント
  // ───────────────────────────────────────────────
  exposes: {
    "./routes": "./src/routes.tsx",
  },

  // ───────────────────────────────────────────────
  // shared: ホストと共有する依存パッケージ
  // （react, react-dom, react-router-dom, React Query, Apolloなど）
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
