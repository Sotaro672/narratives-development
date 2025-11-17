// frontend/shell/src/main.tsx

// ============================================================================
// ① グローバルスタイル: Tailwind v4 + 共通トークンを最優先で読み込む
// ============================================================================
import "../../shell/src/shared/index.css";

import React from "react";
import ReactDOM from "react-dom/client";
import App from "./app/App"; // ← App.tsx 経由で MainPage を呼び出す

// ============================================================================
// ② Shell 固有のグローバルCSS（reset, overridesなど）
// ============================================================================
import "./assets/styles/global.css";

// ============================================================================
// ③ React 18 + Vite エントリポイント
// ============================================================================
const container = document.getElementById("root");
if (!container) {
  throw new Error(
    "Root element not found. Please ensure index.html includes <div id='root'></div>."
  );
}

const root = ReactDOM.createRoot(container);

// StrictMode は開発時のみ有効（本番では自動的に最適化）
root.render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);

// ============================================================================
// ④ Hot Module Replacement (HMR)
// ============================================================================
if (import.meta.hot) {
  import.meta.hot.accept();
}
