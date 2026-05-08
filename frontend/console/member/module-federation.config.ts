import { federation } from "@module-federation/vite";

/**
 * Remote: Member Module
 * 役割: 組織メンバー管理 (ユーザー一覧、招待、ロール設定など)
 */
export default federation({
  name: "member",

  // ───────────────────────────────────────────────
  // exposes: このモジュールが shell に提供するエントリ
  // ───────────────────────────────────────────────
  exposes: {
    "./routes": "./src/routes.tsx",
  },

  // ───────────────────────────────────────────────
  // shared: React 関連ライブラリを単一インスタンスに統一
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
