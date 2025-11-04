// frontend/shell/src/pages/MainPage.tsx
import { useState } from "react";
import Header from "../layout/Header/Header";
import Sidebar from "../layout/Sidebar/Sidebar";
import Main from "../layout/Main/Main";
import "./MainPage.css";

export default function MainPage() {
  const [isSidebarOpen, setIsSidebarOpen] = useState(true);

  return (
    <div className="min-h-screen flex flex-col">
      <Header
        username="Demo User"
        onToggleSidebar={() => setIsSidebarOpen((v) => !v)}
      />

      {/* 横並びにするFlexコンテナ */}
      <div className="flex flex-1 flex-row">
        {/* Sidebar（固定幅240px） */}
        <aside className="sidebar flex">
          <Sidebar isOpen={isSidebarOpen} />
        </aside>

        {/* Main領域（画面幅 - 240px） */}
        <main className="flex-1">
          <Main />
        </main>
      </div>
    </div>
  );
}

