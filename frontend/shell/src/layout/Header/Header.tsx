// frontend/shell/src/layout/Header/Header.tsx
import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom"; // ← 追加
import { Bell, MessageSquare, UserRound, ChevronDown } from "lucide-react";
import "./Header.css";
import AdminPanel from "./AdminPanel";

interface HeaderProps {
  username?: string;
  email?: string;
  announcementsCount?: number;
  messagesCount?: number;
}

export default function Header({
  username = "管理者",
  email = "admin@narratives.com",
  announcementsCount = 3,
  messagesCount = 2,
}: HeaderProps) {
  const [openAdmin, setOpenAdmin] = useState(false);
  const navigate = useNavigate(); // ← 追加

  const panelContainerRef = useRef<HTMLDivElement | null>(null);
  const triggerRef = useRef<HTMLButtonElement | null>(null);

  // 外側クリックで閉じる
  useEffect(() => {
    const onDocClick = (e: MouseEvent) => {
      const t = e.target as Node;
      if (!panelContainerRef.current) return;
      if (panelContainerRef.current.contains(t)) return;
      setOpenAdmin(false);
    };
    document.addEventListener("mousedown", onDocClick);
    return () => document.removeEventListener("mousedown", onDocClick);
  }, []);

  // Esc キーで閉じる
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpenAdmin(false);
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, []);

  // 通知ボタン押下時の遷移処理
  const handleNotificationClick = () => {
    navigate("/announcement");
  };

  return (
    <header className="app-header">
      {/* Left: Brand */}
      <div className="brand">
        <span className="brand-main">Solid State</span>
        <span className="brand-sub">Console</span>
      </div>

      {/* Right: Actions */}
      <div className="actions">
        {/* 通知 */}
        <button
          className="icon-btn"
          aria-label="通知"
          onClick={handleNotificationClick} // ← 遷移を追加
        >
          <span className="icon-wrap">
            <Bell className="icon" aria-hidden />
            {announcementsCount > 0 && (
              <span className="badge" aria-label={`${announcementsCount}件の通知`}>
                {announcementsCount}
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

        {/* ユーザー（トリガー & ドロップダウン） */}
        <div className="relative" ref={panelContainerRef}>
          <button
            ref={triggerRef}
            className="icon-btn user-trigger"
            aria-haspopup="menu"
            aria-expanded={openAdmin}
            aria-controls="admin-dropdown"
            onClick={() => setOpenAdmin((v) => !v)}
          >
            <UserRound className="icon" aria-hidden />
            <ChevronDown className={`caret ${openAdmin ? "open" : ""}`} aria-hidden />
          </button>

          <AdminPanel
            open={openAdmin}
            displayName={username}
            email={email}
            onEditProfile={() => console.log("プロフィール変更")}
            onChangeEmail={() => console.log("メールアドレス変更")}
            onChangePassword={() => console.log("パスワード変更")}
            onLogout={() => console.log("ログアウト")}
          />
        </div>
      </div>
    </header>
  );
}
