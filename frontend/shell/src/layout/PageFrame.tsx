import { useState } from "react";
import { Outlet, useLocation, useNavigate } from "react-router-dom";
import { Menu } from "lucide-react";

/**
 * Solid State Console Layout
 * -----------------------------------------------------------
 * 全ページで共通するレイアウト構造。
 * - Header（アプリタイトル、メニュー開閉ボタン）
 * - Sidebar（各モジュールへのナビゲーション）
 * - Main Content（Outletで各モジュールページを描画）
 * -----------------------------------------------------------
 */

// サイドバーメニュー定義
const menuItems = [
  { label: "ダッシュボード", path: "/" },
  { label: "問い合わせ", path: "/inquiries" },
  { label: "出品", path: "/listings" },
  { label: "運用", path: "/operations" },
  { label: "プレビュー", path: "/preview" },
  { label: "生産計画", path: "/production" },
  { label: "設計", path: "/design" },
  { label: "ミント申請", path: "/mint" },
  { label: "注文管理", path: "/orders" },
  { label: "広告", path: "/ads" },
  { label: "口座", path: "/accounts" },
  { label: "取引履歴", path: "/transactions" },
  { label: "組織管理", path: "/org" },
];

export default function PageFrame() {
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const navigate = useNavigate();
  const location = useLocation();

  return (
    <div className="app-container">
      {/* ─────────────────────────────── Header ─────────────────────────────── */}
      <header className="header">
        <div className="flex items-center gap-3">
          <button
            onClick={() => setSidebarOpen(!sidebarOpen)}
            className="p-2 rounded-md hover:bg-slate-700 focus:outline-none"
          >
            <Menu className="w-5 h-5 text-white" />
          </button>
          <h1 className="text-lg font-semibold tracking-wide">Solid State Console</h1>
        </div>
      </header>

      {/* ─────────────────────────────── Sidebar ─────────────────────────────── */}
      {sidebarOpen && (
        <aside className="sidebar">
          <nav className="flex flex-col mt-2">
            {menuItems.map((item) => {
              const active = location.pathname.startsWith(item.path);
              return (
                <button
                  key={item.path}
                  onClick={() => navigate(item.path)}
                  className={`sidebar-item text-left ${
                    active ? "active" : "hover:bg-slate-700"
                  }`}
                >
                  {item.label}
                </button>
              );
            })}
          </nav>
        </aside>
      )}

      {/* ─────────────────────────────── Main Content ─────────────────────────────── */}
      <main className="main-content">
        <Outlet />
      </main>
    </div>
  );
}
