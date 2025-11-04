// frontend/shell/src/app/App.tsx
import { useState } from "react";
import { BrowserRouter } from "react-router-dom";
import Header from "../layout/Header/Header";
import Sidebar from "../layout/Sidebar/Sidebar";
import Main from "../layout/Main/Main";

export default function App() {
  const [isSidebarOpen, setIsSidebarOpen] = useState(true);

  return (
    <BrowserRouter>
      <div className="min-h-screen flex flex-col bg-slate-900 text-white">
        <Header
          onToggleSidebar={() => setIsSidebarOpen((v) => !v)}
          username="Demo User"
        />
        <div className="flex flex-1">
          <Sidebar isOpen={isSidebarOpen} />
          <main className="flex-1 p-6 overflow-y-auto">
            <Main /> {/* ← MainがRoutesを管理 */}
          </main>
        </div>
      </div>
    </BrowserRouter>
  );
}
