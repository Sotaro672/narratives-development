import { defineConfig } from "@module-federation/vite";
import pkgJson from "../../package.json" assert { type: "json" };

/**
 * Remote: announce
 * ------------------------------------------------------------------------
 * アナウンス・お知らせ配信モジュール（Solid State Console のサブアプリ）
 * exposes: "./routes" をホスト（shell）に公開し、
 * import("announce/routes") で遅延ロード可能にする。
 * ------------------------------------------------------------------------
 */
export default defineConfig({
  // ───────────────────────────────────────────────
  // name: ホスト側 remotes のキーと一致
  // ───────────────────────────────────────────────
  name: "announce",

  // ───────────────────────────────────────────────
  // exposes: shell が import("announce/routes") で読み込む公開エントリ
  // ───────────────────────────────────────────────
  exposes: {
    "./routes": "./src/routes.tsx",
  },

  // ───────────────────────────────────────────────
  // shared: shell と共通依存を共有（React, Router, React Query, Apollo）
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
  // filename: ホストが読み込む remoteEntry 名
  // ───────────────────────────────────────────────
  filename: "remoteEntry.js",

 
