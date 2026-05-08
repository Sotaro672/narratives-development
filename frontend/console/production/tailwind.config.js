import path from "node:path";
const r = (p) => path.resolve(new URL(".", import.meta.url).pathname, p);

// モノレポ内の shared を絶対パスで解決
const R_SHARED = path.resolve(new URL(".", import.meta.url).pathname, "../shared");

export default {
  content: [
    // このアプリ自身
    r("./index.html"),
    r("./src/**/*.{ts,tsx,js,jsx}"),

    // 👇 共有UI（Card 等）をスキャン対象に追加
    path.join(R_SHARED, "**/*.{ts,tsx,js,jsx}"),

    // 必要に応じて他の擬似MFアプリも追加可能
    // r("../admin/src/**/*.{ts,tsx,js,jsx}"),
    // r("../model/src/**/*.{ts,tsx,js,jsx}"),
  ],
  theme: { extend: {} },
  plugins: [],
};
