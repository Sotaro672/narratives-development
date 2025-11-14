import { federation } from "@module-federation/vite";

/**
 * Remote: Company Module
 * 役割: 企業管理（法人情報、住所、登録データ、組織連携など）
 */
export default federation({
  name: "company",

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
