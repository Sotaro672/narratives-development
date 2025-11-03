import { useLocation, useNavigate } from "react-router-dom";
import {
  MessageSquare,
  PackageSearch,
  Wrench,
  Eye,
  CalendarRange,
  PenTool,
  Sparkles,
  ShoppingCart,
  Megaphone,
  Banknote,
  Receipt,
  Building,
} from "lucide-react";

interface SidebarProps {
  /** サイドバー開閉状態（モバイル対応用） */
  isOpen: boolean;
}

/**
 * Sidebar
 * ------------------------------------------------------------------------
 * Solid State Console の左ナビゲーション。
 * ルートベースでアクティブ項目を強調表示し、
 * アイコン付きのメニュー項目を表示する。
 * ------------------------------------------------------------------------
 */
export default function Sidebar({ isOpen }: SidebarProps) {
  const navigate = useNavigate();
  const location = useLocation();

  const menuItems = [
    { label: "問い合わせ", path: "/inquiries", icon: MessageSquare },
    { label: "出品", path: "/listings", icon: PackageSearch },
    { label: "運用", path: "/operations", icon: Wrench },
    { label: "プレビュー", path: "/preview", icon: Eye },
    { label: "生産計画", path: "/production", icon: CalendarRange },
    { label: "設計", path: "/design", icon: PenTool },
    { label: "ミント申請", path: "/mint", icon: Sparkles },
    { label: "注文管理", path: "/orders", icon: ShoppingCart },
    { label: "広告", path: "/ads", icon: Megaphone },
    { label: "口座", path: "/accounts", icon: Banknote },
    { label: "取引履歴", path: "/transactions", icon: Receipt },
    { label: "組織管理", path: "/org", icon: Building },
  ];

  if (!isOpen) return null;

  return (
    <aside className="sidebar">
      <nav className="flex flex-col mt-2">
        {menuItems.map(({ label, path, icon: Icon }) => {
          const active =
            location.pathname === path || location.pathname.startsWith(path + "/");
          return (
            <button
              key={path}
              onClick={() => navigate(path)}
              className={`sidebar-item text-left flex items-center gap-3 ${
                active ? "active" : "hover:bg-slate-700"
              }`}
            >
              <Icon className="w-4 h-4" />
              <span className="text-sm">{label}</span>
            </button>
          );
        })}
      </nav>
    </aside>
  );
}
