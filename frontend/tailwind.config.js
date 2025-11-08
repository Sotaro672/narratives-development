// frontend/tailwind.config.js

/** @type {import('tailwindcss').Config} */
const config = {
  content: [
    "./index.html",
    "./**/*.{js,ts,jsx,tsx}",

    // 共有UIコンポーネント
    "./shared/ui/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      // 必要ならここに共通トークンなど
    },
  },
  plugins: [],
};

module.exports = config;
