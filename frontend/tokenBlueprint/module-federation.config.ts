import { federation } from "@module-federation/vite";

/**
 * Remote: Token Blueprint Module
 * 役割: トークン設計ブループリント管理（NFTテンプレート、メタデータ構造、認証仕様など）
 */
export default federation({
  name: "tokenBlueprint",

  // ───────────────────────────────────────────────
  // exposes: shell 側に提供するルートエントリポイント
  // ───────────────────────────────────────────────
  exposes: {
    "./routes": "./src/routes.tsx",
  },

  // ───────────────────────────────────────────────
  // shared: 共通ライブラリを singleton 化して競合を防止
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
