// frontend/vite.config.ts
import { defineConfig, type PluginOption } from "vite";
import react from "@vitejs/plugin-react";
import tailwind from "@tailwindcss/vite";

// ────────────────────────────────────────────────────────────
// ESM だけで安全にパスを解決するヘルパ（__dirname / node:path 不要）
// ────────────────────────────────────────────────────────────
const r = (p: string) => new URL(p, import.meta.url).pathname;

// ────────────────────────────────────────────────────────────
/** Module Federation プラグイン（任意読み込み） */
// ./shell/module-federation.config.(ts|js) が存在しない場合でもビルドを通す
// 見つかった場合のみ plugins に追加されます
let mfPlugin: PluginOption | undefined;
try {
  // Vite の設定ファイルは ESM なので dynamic import を使う
  // 拡張子は Vite が解決してくれる前提（.ts/.js どちらでもOK）
  // eslint-disable-next-line @typescript-eslint/ban-ts-comment
  // @ts-ignore - dynamic import path
  const mod = await import("./shell/module-federation.config");
  // 型の衝突を避けるために PluginOption にキャスト
  mfPlugin = (mod.default ?? mod) as PluginOption;
} catch {
  // 見つからないときは無視（MFなしで動作）
  mfPlugin = undefined;
}

export default defineConfig({
  plugins: [
    react(),
    tailwind(),
    ...(mfPlugin ? [mfPlugin] : []), // 存在する時だけ追加
  ],

  resolve: {
    alias: {
      // ルート直下の各パッケージを絶対パスで参照できるようにする
      "@shared": r("./shared"),
      "@shared-ui": r("./shared/ui"),
      "@shell": r("./shell"),
      "@admin": r("./admin"),
      "@productBlueprint": r("./productBlueprint"),
    },
  },

  server: {
    port: 4000,
    open: true,
    fs: {
      // ルート外参照の許可（monorepo 風ディレクトリを明示許可）
      allow: [
        r("."), // frontend/
        r("./shared"),
        r("./shell"),
        r("./admin"),
        r("./productBlueprint"),
        r("./inquiry"),
        r("./production"),
        r("./inventory"),
        r("./tokenBlueprint"),
        r("./mintRequest"),
        r("./operation"),
        r("./list"),
        r("./order"),
        r("./member"),
        r("./brand"),
        r("./permission"),
        r("./ad"),
        r("./account"),
        r("./transaction"),
      ],
    },
  },

  build: {
    target: "esnext",
    modulePreload: false,
    outDir: "dist",
  },
});
