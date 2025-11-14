// frontend/shell/src/app/App.tsx
import { BrowserRouter } from "react-router-dom";
import MainPage from "../pages/MainPage"; // ← MainPage をインポート

/**
 * App.tsx
 * - 画面全体のルート構成を管理
 * - BrowserRouter 配下で MainPage を呼び出す
 */
export default function App() {
  return (
    <BrowserRouter>
      <MainPage /> {/* ← これが全体レイアウトを表示 */}
    </BrowserRouter>
  );
}
