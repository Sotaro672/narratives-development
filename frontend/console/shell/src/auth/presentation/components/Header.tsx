// frontend/console/shell/src/auth/presentation/components/Header.tsx
import { Bell, MessageSquare, UserRound, ChevronDown } from "lucide-react";
import "../styles/auth.css";
import AdminPanel from "./AdminPanel";
import { useHeader } from "../hook/useHeader";

interface HeaderProps {
  username?: string;
  email?: string;
  announcementsCount?: number;
  messagesCount?: number;
}

export default function Header(props: HeaderProps) {
  const {
    openAdmin,
    panelContainerRef,
    triggerRef,
    brandMain,
    fullName,
    displayEmail,
    announcementsCount,
    messagesCount,
    handleNotificationClick,
    handleMessageClick,
    handleToggleAdmin,
    handleLogout,
  } = useHeader(props);

  return (
    <header className="app-header">
      {/* Left: Brand */}
      <div className="brand">
        <span className="brand-main">{brandMain}</span>
        <span className="brand-sub">Console</span>
      </div>

      {/* Right: Actions */}
      <div className="actions">
        {/* 通知 */}
        <button
          className="icon-btn"
          aria-label="通知"
          onClick={handleNotificationClick}
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
        <button
          className="icon-btn"
          aria-label="メッセージ"
          onClick={handleMessageClick}
        >
          <span className="icon-wrap">
            <MessageSquare className="icon" aria-hidden />
            {messagesCount > 0 && (
              <span className="badge" aria-label={`${messagesCount}件の新着メッセージ`}>
                {messagesCount}
              </span>
            )}
          </span>
        </button>

        {/* ユーザードロップダウン */}
        <div className="relative" ref={panelContainerRef}>
          <button
            ref={triggerRef}
            className="icon-btn user-trigger"
            aria-haspopup="menu"
            aria-expanded={openAdmin}
            aria-controls="admin-dropdown"
            onClick={handleToggleAdmin}
          >
            <UserRound className="icon" aria-hidden />
            <ChevronDown
              className={`caret ${openAdmin ? "open" : ""}`}
              aria-hidden
            />
          </button>

          <AdminPanel
            open={openAdmin}
            fullName={fullName}
            email={displayEmail}
            onEditProfile={() => console.log("プロフィール変更")}
            onChangeEmail={() => console.log("メールアドレス変更")}
            onChangePassword={() => console.log("パスワード変更")}
            onLogout={handleLogout}
          />
        </div>
      </div>
    </header>
  );
}
