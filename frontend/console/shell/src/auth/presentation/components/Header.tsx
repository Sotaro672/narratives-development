// frontend/console/shell/src/auth/presentation/components/Header.tsx

import {
  Bell,
  ChevronDown,
  ChevronUp,
  UserRound,
} from "lucide-react";
import { useNavigate } from "react-router-dom";

import { useNotificationUnreadCount } from "../../../features/notification/presentation/hooks/useNotificationUnreadCount";
import { Badge } from "../../../shared/ui/badge";
import { Button } from "../../../shared/ui/button";
import Text from "../../../shared/ui/text";
import AdminPanel from "./AdminPanel";
import { useHeader } from "../hook/useHeader";

import "../../../styles/auth.css";

interface HeaderProps {
  username?: string;
  email?: string;
}

export default function Header(props: HeaderProps) {
  const navigate = useNavigate();

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

  const { unreadCount } = useNotificationUnreadCount();

  const handleOpenNotifications = () => {
    if (openAdmin) {
      handleToggleAdmin();
    }

    navigate("/notifications");
  };

  const notificationAriaLabel =
    unreadCount > 0
      ? `通知を開く。未読${unreadCount}件`
      : "通知を開く";

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

        <Text
          size="md"
          tone="muted"
          className="brand-sub"
        >
          Console
        </Text>
      </button>

      <div className="actions">
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="header-icon-button"
          aria-label={notificationAriaLabel}
          title="通知"
          onClick={handleOpenNotifications}
        >
          <Bell
            size={22}
            strokeWidth={1.8}
            aria-hidden="true"
          />

          {unreadCount > 0 ? (
            <Badge
              variant="danger"
              className="header-notification-badge"
            >
              {unreadCount > 99 ? "99+" : unreadCount}
            </Badge>
          ) : null}
        </Button>

        <div className="relative" ref={panelContainerRef}>
          <Button
            ref={triggerRef}
            type="button"
            variant="ghost"
            size="sm"
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
            <UserRound
              size={22}
              strokeWidth={1.8}
              aria-hidden="true"
            />

            {openAdmin ? (
              <ChevronUp
                size={14}
                strokeWidth={1.8}
                aria-hidden="true"
              />
            ) : (
              <ChevronDown
                size={14}
                strokeWidth={1.8}
                aria-hidden="true"
              />
            )}
          </Button>

          {openAdmin ? (
            <AdminPanel
              fullName={fullName}
              email={displayEmail}
              onLogout={handleLogout}
            />
          ) : null}
        </div>
      </div>
    </header>
  );
}