// frontend/admin/src/main.tsx
// ★ 共有のトークン & Tailwind を最初に読み込む
import "../../shared/index.css"
import React from "react"
import ReactDOM from "react-dom/client"
import App from "./App"

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
)
