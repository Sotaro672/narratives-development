// frontend/console/shell/src/auth/presentation/components/Header.tsx

import {
  ChevronDown,
  ChevronUp,
  UserRound,
} from "lucide-react";

import "../../../styles/auth.css";

import AdminPanel from "./AdminPanel";
import { useHeader } from "../hook/useHeader";

interface HeaderProps {
  username?: string;
  email?: string;
}

export default function Header(props: HeaderProps) {
  const {
    openAdmin,
    panelContainerRef,
    triggerRef,
    brandMain,
    fullName,
    displayEmail,
    handleOpenCompanyDetail,
    handleToggleAdmin,
    handleLogout,
  } = useHeader({
    username: props.username ?? "ログインできていません",
    email: props.email ?? "ログインできていません",
  });

  return (
    <header className="app-header">
      <button
        type="button"
        className="brand"
        onClick={handleOpenCompanyDetail}
        aria-label="会社情報を開く"
        style={{
          border: 0,
          background: "transparent",
          padding: 0,
          cursor: "pointer",
        }}
      >
        <span className="brand-main">{brandMain}</span>
        <span className="brand-sub">Console</span>
      </button>

      <div className="actions">
        <div className="relative" ref={panelContainerRef}>
          <button
            ref={triggerRef}
            type="button"
            className="icon-btn user-trigger"
            aria-haspopup="menu"
            aria-expanded={openAdmin}
            aria-controls={openAdmin ? "admin-dropdown" : undefined}
            aria-label={
              openAdmin
                ? "アカウントメニューを閉じる"
                : "アカウントメニューを開く"
            }
            onClick={handleToggleAdmin}
          >
            <UserRound className="icon" aria-hidden />

            {openAdmin ? (
              <ChevronUp
                className="caret"
                aria-hidden
              />
            ) : (
              <ChevronDown
                className="caret"
                aria-hidden
              />
            )}
          </button>

          {openAdmin && (
            <AdminPanel
              fullName={fullName}
              email={displayEmail}
              onLogout={handleLogout}
            />
          )}
        </div>
      </div>
    </header>
  );
}