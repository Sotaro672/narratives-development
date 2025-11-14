// frontend/shell/src/pages/MainPage.tsx
import { useState } from "react";
import Header from "../layout/Header/Header";
import Sidebar from "../layout/Sidebar/Sidebar";
import Main from "../layout/Main/Main";
import "./MainPage.css";

export default function MainPage() {
  // 現状は常に開いた状態だが、将来のUI拡張に備えて state は残しておく
  const [isSidebarOpen] = useState(true);

  return (
    <div className="min-h-screen flex flex-col">
      {/* ヘッダー */}
      <Header username="Demo User" />

      {/* 横並びレイアウト */}
      <div className="flex flex-1 flex-row">
        {/* Sidebar（固定幅） */}
        <aside className="sidebar flex">
          <Sidebar isOpen={isSidebarOpen} />
        </aside>

        {/* Main領域：/console 配下のルート内容がここに表示される */}
        <main className="flex-1">
          <Main />
        </main>
      </div>
    </div>
  );
}
