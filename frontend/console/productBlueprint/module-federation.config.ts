import { federation } from "@module-federation/vite";

/**
 * Remote: Product Blueprint Module
 * 役割: 商品設計ブループリント管理（製品仕様テンプレート、タグ構成、設計構造など）
 */
export default federation({
  name: "productBlueprint",

  // ───────────────────────────────────────────────
  // exposes: shell 側に提供するルートエントリポイント
  // ───────────────────────────────────────────────
  exposes: {
    "./routes": "./src/routes.tsx",
  },

  // ───────────────────────────────────────────────
  // shared: 共通ライブラリを singleton 化して競合回避
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
