import { useState } from "react";
import { Bell, MessageSquare, User } from "lucide-react";
import "./Header.css";

interface HeaderProps {
  /** 未読通知件数（赤バッジ） */
  notificationsCount?: number;
  /** 未読メッセージ件数（赤バッジ） */
  messagesCount?: number;
  /** サイドバー開閉トグル（オプション） */
  onToggleSidebar?: () => void;
  /** ログイン中ユーザー名 */
  username?: string;
}

export default function Header({
  notificationsCount = 3,
  messagesCount = 2,
  username = "Guest",
}: HeaderProps) {
  const [showProfileMenu, setShowProfileMenu] = useState(false);

  return (
    <header className="app-header">
      {/* ───────────── Left: Brand / Menu ───────────── */}
      <div className="brand">
        <span className="brand-main">Solid State</span>
        <span className="brand-sub">Console</span>
      </div>

      {/* ───────────── Right: Actions ───────────── */}
      <div className="actions">
        {/* 通知 */}
        <button className="icon-btn" aria-label="通知">
          <span className="icon-wrap">
            <Bell className="icon" aria-hidden />
            {notificationsCount > 0 && (
              <span className="badge" aria-label={`${notificationsCount}件の通知`}>
                {notificationsCount}
              </span>
            )}
          </span>
        </button>

        {/* メッセージ */}
        <button className="icon-btn" aria-label="メッセージ">
          <span className="icon-wrap">
            <MessageSquare className="icon" aria-hidden />
            {messagesCount > 0 && (
              <span className="badge" aria-label={`${messagesCount}件の新着メッセージ`}>
                {messagesCount}
              </span>
            )}
          </span>
        </button>

        {/* プロフィール */}
        <div className="relative">
          <button
            className="icon-btn"
            aria-label="プロフィール"
            onClick={() => setShowProfileMenu((v) => !v)}
          >
            <User className="icon" aria-hidden />
            <span className="username">{username}</span>
          </button>

          {showProfileMenu && (
            <div className="profile-menu">
              <button className="menu-item">設定</button>
              <button className="menu-item">ログアウト</button>
            </div>
          )}
        </div>
      </div>
    </header>
  );
}
