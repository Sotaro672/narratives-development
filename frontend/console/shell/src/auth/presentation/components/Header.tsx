// frontend/console/shell/src/auth/presentation/components/Header.tsx
import { UserRound, ChevronDown } from "lucide-react";
import "../styles/auth.css";
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
    handleToggleAdmin,
    handleLogout,
  } = useHeader({
    username: props.username ?? "ログインできていません",
    email: props.email ?? "ログインできていません",
  });

  return (
    <header className="app-header">
      <div className="brand">
        <span className="brand-main">{brandMain}</span>
        <span className="brand-sub">Console</span>
      </div>

      <div className="actions">
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
            onEditProfile={() => undefined}
            onChangeEmail={() => undefined}
            onChangePassword={() => undefined}
            onLogout={handleLogout}
          />
        </div>
      </div>
    </header>
  );
}