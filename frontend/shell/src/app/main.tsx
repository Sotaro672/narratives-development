// frontend/shell/src/main.tsx
import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import "../assets/styles/global.css"; // Tailwind・reset.css など全体スタイル

// ============================================================================
// Vite + React 18 Entry Point
// ============================================================================
// App.tsx（ルート構成）をブラウザの <div id="root"> にマウントします。
// ここ以外では ReactDOM を呼び出しません。
// ============================================================================

const container = document.getElementById("root");
if (!container) {
  throw new Error(
    "Root element not found. Please ensure index.html includes <div id='root'></div>."
  );
}

// React 18 の createRoot API
const root = ReactDOM.createRoot(container);

// -----------------------------------------------------------------------------
// StrictMode: 開発時の副作用検出（本番では無効）
// -----------------------------------------------------------------------------
root.render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);

// -----------------------------------------------------------------------------
// HMR (Hot Module Replacement)
// Viteの開発サーバでホットリロードを有効化する
// -----------------------------------------------------------------------------
if (import.meta.hot) {
  import.meta.hot.accept();
}

// -----------------------------------------------------------------------------
// 拡張ポイント（例）
// -----------------------------------------------------------------------------
// 1. Firebase AppCheck / Analytics 初期化
// 2. Sentry.init({ dsn: "...", environment: import.meta.env.MODE });
// 3. Cloud Run 環境変数のロード
// -----------------------------------------------------------------------------
