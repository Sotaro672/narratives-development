import { useState } from "react";
import { Bell, Menu, User } from "lucide-react";

interface HeaderProps {
  /** サイドバー開閉トグル */
  onToggleSidebar: () => void;
  /** 現在のログインユーザー名（任意） */
  username?: string;
}

export default function Header({ onToggleSidebar, username }: HeaderProps) {
  const [showProfileMenu, setShowProfileMenu] = useState(false);
  const [showNotifications, setShowNotifications] = useState(false);

  return (
    <header className="header">
      {/* ───────────── Left Area ───────────── */}
      <div className="flex items-center gap-3">
        <button
          onClick={onToggleSidebar}
          className="p-2 rounded-md hover:bg-slate-700 focus:outline-none"
          aria-label="Toggle sidebar"
        >
          <Menu className="w-5 h-5 text-white" />
        </button>
        <h1 className="text-lg font-semibold tracking-wide select-none">
          Solid State Console
        </h1>
      </div>

      {/* ───────────── Right Area ───────────── */}
      <div className="flex items-center gap-4 relative">
        {/* 通知ベル */}
        <button
          onClick={() => setShowNotifications((v) => !v)}
          className="relative p-2 rounded-md hover:bg-slate-700 focus:outline-none"
          aria-label="Notifications"
        >
          <Bell className="w-5 h-5 text-white" />
          <span className="absolute -top-0.5 -right-0.5 bg-rose-500 text-[10px] font-bold text-white rounded-full px-[4px]">
            3
          </span>
        </button>

        {/* プロフィール */}
        <div className="relative">
          <button
            onClick={() => setShowProfileMenu((v) => !v)}
            className="flex items-center gap-2 p-2 rounded-md hover:bg-slate-700 focus:outline-none"
          >
            <User className="w-5 h-5 text-white" />
            <span className="hidden md:inline text-sm text-gray-200">
              {username ?? "Guest"}
            </span>
          </button>

          {/* ドロップダウンメニュー */}
          {showProfileMenu && (
            <div className="absolute right-0 mt-2 w-40 bg-white text-gray-700 rounded-md shadow-lg border border-gray-200 z-50">
              <button
                onClick={() => alert("設定画面を開く")}
                className="w-full text-left px-4 py-2 text-sm hover:bg-gray-100"
              >
                設定
              </button>
              <button
                onClick={() => alert("ログアウト")}
                className="w-full text-left px-4 py-2 text-sm hover:bg-gray-100"
              >
                ログアウト
              </button>
            </div>
          )}
        </div>
      </div>

      {/* 通知ドロップダウン（右上） */}
      {showNotifications && (
        <div className="absolute right-4 top-14 w-80 bg-white rounded-lg shadow-lg border border-gray-200 z-50">
          <div className="p-3 border-b font-semibold text-gray-700">通知</div>
          <ul className="max-h-64 overflow-y-auto">
            <li className="px-4 py-2 text-sm hover:bg-gray-50">
              新しい問い合わせが届きました
            </li>
            <li className="px-4 py-2 text-sm hover:bg-gray-50">
              ブランド設定が更新されました
            </li>
            <li className="px-4 py-2 text-sm hover:bg-gray-50">
              注文 #A1023 が出荷されました
            </li>
          </ul>
          <div className="text-center text-sm text-blue-500 py-2 border-t hover:bg-gray-50 cursor-pointer">
            すべて見る
          </div>
        </div>
      )}
    </header>
  );
}
