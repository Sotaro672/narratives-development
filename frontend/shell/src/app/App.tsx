import { useState } from "react";
import { BrowserRouter } from "react-router-dom"; // ← 追加
import Header from "../layout/Header/Header";
import Sidebar from "../layout/Sidebar/Sidebar";
import PageFrame from "../layout/PageFrame";

export default function App() {
  const [isSidebarOpen, setIsSidebarOpen] = useState(true);

  return (
    <BrowserRouter> {/* ← これで Sidebar のフックが有効になる */}
      <div className="min-h-screen flex flex-col bg-slate-900 text-white">
        <Header
          onToggleSidebar={() => setIsSidebarOpen((v) => !v)}
          username="Demo User"
        />

        <div className="flex flex-1">
          <Sidebar isOpen={isSidebarOpen} />
          <main className="flex-1 p-8">
            <PageFrame />
          </main>
        </div>
      </div>
    </BrowserRouter>
  );
}
