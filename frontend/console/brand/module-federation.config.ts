import { federation } from "@module-federation/vite";

/**
 * Remote: Brand Module
 * 役割: ブランド管理（ブランド設定・ブランドプロフィール・ロゴなど）
 */
export default federation({
  name: "brand",

  // ───────────────────────────────────────────────
  // exposes: shell に提供するエントリポイント
  // ───────────────────────────────────────────────
  exposes: {
    "./routes": "./src/routes.tsx",
  },

  // ───────────────────────────────────────────────
  // shared: React 関連の依存関係を singleton 化
  // ───────────────────────────────────────────────
  shared: {
    react: { singleton: true },
    "react-dom": { singleton: true },
    "react-router-dom": { singleton: true },
    "@tanstack/react-query": { singleton: true },
    "@apollo/client": { singleton: true },
  },

  // ───────────────────────────────────────────────
  // 出力設定
  // ───────────────────────────────────────────────
  filename: "remoteEntry.js",
  manifest: true,
  dts: false,
});
