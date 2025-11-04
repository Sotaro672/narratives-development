import { LogOut } from "lucide-react";
import "./AdminPanel.css";

interface AdminPanelProps {
  /** ドロップダウンの開閉状態（Header が制御） */
  open: boolean;

  /** 表示名・メール */
  displayName?: string; // 例: "管理者"
  email?: string;       // 例: "admin@narratives.com"

  /** アクション */
  onEditProfile?: () => void;
  onChangeEmail?: () => void;
  onChangePassword?: () => void;
  onLogout?: () => void;

  /** ルートの className（任意） */
  className?: string;
}

/**
 * AdminPanel はドロップダウン本体のみを描画します。
 * - トリガー（ユーザーアイコン）は Header 側で管理
 * - 開閉状態も Header 側で管理（controlled）
 */
export default function AdminPanel({
  open,
  displayName = "管理者",
  email = "",
  onEditProfile,
  onChangeEmail,
  onChangePassword,
  onLogout,
  className,
}: AdminPanelProps) {
  if (!open) return null;

  return (
    <div className={`admin-dropdown ${className || ""}`} role="menu" aria-label="アカウントメニュー">
      <div className="admin-dropdown-header">
        <div className="admin-dropdown-title">{displayName}</div>
        {email && <div className="admin-dropdown-email">{email}</div>}
      </div>

      <div className="admin-dropdown-sep" />

      <button
        className="admin-dropdown-item"
        role="menuitem"
        onClick={onEditProfile}
      >
        プロフィール変更
      </button>
      <button
        className="admin-dropdown-item"
        role="menuitem"
        onClick={onChangeEmail}
      >
        メールアドレス変更
      </button>
      <button
        className="admin-dropdown-item"
        role="menuitem"
        onClick={onChangePassword}
      >
        パスワード変更
      </button>

      <div className="admin-dropdown-sep" />

      <button
        className="admin-dropdown-item logout"
        role="menuitem"
        onClick={onLogout}
      >
        <LogOut className="logout-icon" aria-hidden />
        ログアウト
      </button>
    </div>
  );
}
