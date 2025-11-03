import { federation } from "@module-federation/vite";

/**
 * Remote: Inventory Module
 * 役割: 在庫管理（SKU、ロット、入出庫、在庫数同期など）
 */
export default federation({
  name: "inventory",

  // ───────────────────────────────────────────────
  // exposes: shell 側に提供するルートエントリ
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
