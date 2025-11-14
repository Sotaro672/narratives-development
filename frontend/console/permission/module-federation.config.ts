import { federation } from "@module-federation/vite";

/**
 * Remote: Permission Module
 * 役割: 権限・ロール管理（ユーザー権限、ロール階層、アクセス制御など）
 */
export default federation({
  name: "permission",

  // ───────────────────────────────────────────────
  // exposes: shell 側に公開するエントリポイント
  // ───────────────────────────────────────────────
  exposes: {
    "./routes": "./src/routes.tsx",
  },

  // ───────────────────────────────────────────────
  // shared: 共通依存ライブラリを単一インスタンスに統一
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
