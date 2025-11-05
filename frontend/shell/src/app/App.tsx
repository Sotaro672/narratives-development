// frontend/shell/src/app/App.tsx
import { BrowserRouter } from "react-router-dom";
import Header from "../layout/Header/Header";
import Sidebar from "../layout/Sidebar/Sidebar";
import Main from "../layout/Main/Main";

/**
 * App.tsx
 * - 画面全体のレイアウトを定義
 * - Sidebar / Header は固定
 * - Main（右エリア）のみスクロール可能
 */
export default function App() {
  return (
    <BrowserRouter>
      {/* 全体を固定レイアウト化 */}
      <div className="h-screen w-screen overflow-hidden flex flex-col bg-background text-foreground">
        {/* 固定ヘッダー */}
        <Header />

        {/* メインコンテンツ: Sidebar + Main */}
        <div className="flex flex-1 min-h-0">
          {/* 左サイドバー（固定） */}
          <Sidebar isOpen={true} />

          {/* 右のメイン領域（スクロール対象） */}
          <main className="flex-1 relative">
            <Main />
          </main>
        </div>
      </div>
    </BrowserRouter>
  );
}
