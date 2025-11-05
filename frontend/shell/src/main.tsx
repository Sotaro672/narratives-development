// frontend/shell/src/main.tsx

// ① Tailwind v4 + 共有トークンを最優先で読み込む（超重要）
import "../../shared/index.css";

import React from "react";
import ReactDOM from "react-dom/client";
import App from "./app/App";

// （任意）Shell 固有のグローバルCSS（reset 等）
// ※ 共有トークンの上書きが必要なら、このファイルで行う
import "./assets/styles/global.css";

// ============================================================================
// Vite + React 18 Entry Point
// ============================================================================
const container = document.getElementById("root");
if (!container) {
  throw new Error(
    "Root element not found. Please ensure index.html includes <div id='root'></div>."
  );
}

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
// HMR (Hot Module Replacement): Viteの開発サーバでホットリロードを有効化
// -----------------------------------------------------------------------------
if (import.meta.hot) {
  import.meta.hot.accept();
}

// -----------------------------------------------------------------------------
// 拡張ポイント（例）
// 1. Firebase AppCheck / Analytics 初期化
// 2. Sentry.init({ dsn: "...", environment: import.meta.env.MODE });
// 3. Cloud Run 環境変数のロード
// -----------------------------------------------------------------------------
