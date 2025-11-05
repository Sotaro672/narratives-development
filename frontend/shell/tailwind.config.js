// frontend/shell/tailwind.config.js
import path from "node:path";
const r = (p) => path.resolve(new URL(".", import.meta.url).pathname, p);

export default {
  content: [
    // shell 自身
    r("./index.html"),
    r("./src/**/*.{ts,tsx,js,jsx}"),

    // 共有 UI（Card 等）
    r("../shared/**/*.{ts,tsx,js,jsx}"),

    // ★ 各 “擬似 MF” アプリを明示的に追加（最低限 admin は必須）
    r("../admin/src/**/*.{ts,tsx,js,jsx}"),
    // 必要なら他も：
    // r("../productBlueprint/src/**/*.{ts,tsx,js,jsx}"),
    // r("../member/src/**/*.{ts,tsx,js,jsx}"),
    // r("../**/src/**/*.{ts,tsx,js,jsx}"), // ←ワイルドカードで一括も可
  ],
  theme: { extend: {} },
  plugins: [],
};
