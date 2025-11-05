/** @type {import('tailwindcss').Config} */
export default {
  content: [
    // shell 自身
    "./index.html",
    "./src/**/*.{ts,tsx,js,jsx}",

    // 共有 UI
    "../shared/**/*.{ts,tsx,js,jsx}",

    // ← ここを追加：擬似モノリス内の他アプリをまとめて拾う
    "../**/src/**/*.{ts,tsx,js,jsx}",

    // もし node_modules 内でクラスを持つ自作パッケージがあれば追加
    // "../../packages/**/src/**/*.{ts,tsx,js,jsx}",
  ],
  theme: { extend: {} },
  plugins: [],
};
