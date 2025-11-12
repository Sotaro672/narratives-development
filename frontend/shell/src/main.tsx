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

// ============================================================================
// ⑤ 拡張ポイント（Firebase / Sentry / Analytics などの初期化）
// ----------------------------------------------------------------------------
// 例：
// import { initializeApp } from "firebase/app";
// import { getAnalytics } from "firebase/analytics";
// const firebaseConfig = {
//   apiKey: "AIzaSyDTetB8PcVlSHhXbItMZv2thd5lY4d5nIQ",
//   authDomain: "narratives-development-26c2d.firebaseapp.com",
//   projectId: "narratives-development-26c2d",
//   storageBucket: "narratives-development-26c2d.firebasestorage.app",
//   messagingSenderId: "871263659099",
//   appId: "1:871263659099:web:0d4bbdc36e59d7ed8d4b7e",
//   measurementId: "G-T77JW1DF4V",
// };
// const app = initializeApp(firebaseConfig);
// const analytics = getAnalytics(app);
// ============================================================================
