import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import "../index.css"; // Tailwindやreset.cssなどの全体スタイル

// ============================================================================
// Vite + React 18 Entry Point
// ============================================================================
//
// このファイルは「アプリのエントリーポイント」です。
// App.tsx（ルート構成）をブラウザの <div id="root"> にマウントします。
// ここ以外ではReactDOMを呼び出しません。
// ============================================================================

// HTML側のroot要素を取得
const container = document.getElementById("root");

if (!container) {
  throw new Error(
    "Root element not found. Ensure index.html includes <div id='root'></div>."
  );
}

// React 18 の createRoot API を使用
const root = ReactDOM.createRoot(container);

// -----------------------------------------------------------------------------
// StrictMode: 開発中の副作用チェック（本番では無効）
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
// 今後の拡張ポイント例：
// -----------------------------------------------------------------------------
// 1. Firebase AppCheck や Analytics の初期化
// 2. Sentry.init({ dsn: "...", environment: import.meta.env.MODE });
// 3. Cloud Run 環境変数の初期ロード
// -----------------------------------------------------------------------------
