// frontend/console/shell/src/layout/Header/AdminPanel.tsx
import * as React from "react";
import { LogOut } from "lucide-react";
import "./AdminPanel.css";
import { Input } from "../../../../shell/src/shared/ui/input";

interface AdminPanelProps {
  open: boolean;
  fullName?: string;
  email?: string;

  onEditProfile?: () => void;
  onChangeEmail?: () => void;
  onChangePassword?: () => void;
  onLogout?: () => void;

  className?: string;
}

export default function AdminPanel({
  open,
  fullName = "管理者",
  email = "",
  onEditProfile,
  onChangeEmail,
  onChangePassword,
  onLogout,
  className,
}: AdminPanelProps) {
  const [showProfileDialog, setShowProfileDialog] = React.useState(false);
  const [showEmailDialog, setShowEmailDialog] = React.useState(false);
  const [showPasswordDialog, setShowPasswordDialog] = React.useState(false);

  // InvitationPage と同じ形式（プロフィール変更用）
  const [lastName, setLastName] = React.useState("");
  const [lastNameKana, setLastNameKana] = React.useState("");
  const [firstName, setFirstName] = React.useState("");
  const [firstNameKana, setFirstNameKana] = React.useState("");

  // メール変更・PW変更フォーム
  const [newEmail, setNewEmail] = React.useState("");
  const [currentPasswordForEmail, setCurrentPasswordForEmail] =
    React.useState("");

  const [currentPassword, setCurrentPassword] = React.useState("");
  const [newPassword, setNewPassword] = React.useState("");
  const [confirmPassword, setConfirmPassword] = React.useState("");

  if (!open) return null;

  const handleProfileSave = () => {
    // ここで API 呼び出しなどに lastName / firstName などを渡す想定
    onEditProfile?.();
    setShowProfileDialog(false);
  };

  const handleEmailSave = () => {
    // ここで API 呼び出しなどに newEmail / currentPasswordForEmail を渡す想定
    onChangeEmail?.();
    setShowEmailDialog(false);
  };

  const handlePasswordSave = () => {
    // ここで API 呼び出しなどに currentPassword / newPassword などを渡す想定
    onChangePassword?.();
    setShowPasswordDialog(false);
  };

  return (
    <>
      {/* ─────────────────────────────── */}
      {/* ドロップダウン本体            */}
      {/* ─────────────────────────────── */}
      <div
        className={`admin-dropdown ${className || ""}`}
        role="menu"
        aria-label="アカウントメニュー"
      >
        <div className="admin-dropdown-header">
          <div className="admin-dropdown-title">{fullName}</div>
          {email && <div className="admin-dropdown-email">{email}</div>}
        </div>

        <div className="admin-dropdown-sep" />

        <button
          className="admin-dropdown-item"
          role="menuitem"
          onClick={() => setShowProfileDialog(true)}
        >
          プロフィール変更
        </button>
        <button
          className="admin-dropdown-item"
          role="menuitem"
          onClick={() => setShowEmailDialog(true)}
        >
          メールアドレス変更
        </button>
        <button
          className="admin-dropdown-item"
          role="menuitem"
          onClick={() => setShowPasswordDialog(true)}
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

      {/* ─────────────────────────────── */}
      {/* プロフィール変更ダイアログ     */}
      {/* ─────────────────────────────── */}
      {showProfileDialog && (
        <div className="admin-modal-backdrop" role="dialog" aria-modal="true">
          <div className="admin-modal">
            <div className="admin-modal-title">プロフィール変更</div>

            <div className="space-y-4">
              {/* 姓 → 姓（かな） */}
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="admin-modal-label">姓</label>
                  <Input
                    variant="default"
                    className="admin-modal-input"
                    value={lastName}
                    onChange={(e) => setLastName(e.target.value)}
                    placeholder="山田"
                  />
                </div>

                <div>
                  <label className="admin-modal-label">姓（かな）</label>
                  <Input
                    variant="default"
                    className="admin-modal-input"
                    value={lastNameKana}
                    onChange={(e) => setLastNameKana(e.target.value)}
                    placeholder="やまだ"
                  />
                </div>
              </div>

              {/* 名 → 名（かな） */}
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="admin-modal-label">名</label>
                  <Input
                    variant="default"
                    className="admin-modal-input"
                    value={firstName}
                    onChange={(e) => setFirstName(e.target.value)}
                    placeholder="太郎"
                  />
                </div>

                <div>
                  <label className="admin-modal-label">名（かな）</label>
                  <Input
                    variant="default"
                    className="admin-modal-input"
                    value={firstNameKana}
                    onChange={(e) => setFirstNameKana(e.target.value)}
                    placeholder="たろう"
                  />
                </div>
              </div>
            </div>

            <div className="admin-modal-footer">
              <button
                className="admin-modal-button cancel"
                type="button"
                onClick={() => setShowProfileDialog(false)}
              >
                キャンセル
              </button>

              <button
                className="admin-modal-button primary"
                type="button"
                onClick={handleProfileSave}
              >
                保存
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ─────────────────────────────── */}
      {/* メールアドレス変更ダイアログ   */}
      {/* ─────────────────────────────── */}
      {showEmailDialog && (
        <div className="admin-modal-backdrop" role="dialog" aria-modal="true">
          <div className="admin-modal">
            <div className="admin-modal-title">メールアドレス変更</div>

            <div className="space-y-4">
              <div>
                <label className="admin-modal-label">新しいメールアドレス</label>
                <Input
                  variant="default"
                  className="admin-modal-input"
                  type="email"
                  value={newEmail}
                  onChange={(e) => setNewEmail(e.target.value)}
                  placeholder="new@example.com"
                />
              </div>

              <div>
                <label className="admin-modal-label">現在のパスワード</label>
                <Input
                  variant="default"
                  className="admin-modal-input"
                  type="password"
                  value={currentPasswordForEmail}
                  onChange={(e) =>
                    setCurrentPasswordForEmail(e.target.value)
                  }
                  placeholder="現在のパスワード"
                />
              </div>
            </div>

            <div className="admin-modal-footer">
              <button
                className="admin-modal-button cancel"
                type="button"
                onClick={() => setShowEmailDialog(false)}
              >
                キャンセル
              </button>
              <button
                className="admin-modal-button primary"
                type="button"
                onClick={handleEmailSave}
              >
                保存
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ─────────────────────────────── */}
      {/* パスワード変更ダイアログ       */}
      {/* ─────────────────────────────── */}
      {showPasswordDialog && (
        <div className="admin-modal-backdrop" role="dialog" aria-modal="true">
          <div className="admin-modal">
            <div className="admin-modal-title">パスワード変更</div>

            <div className="space-y-4">
              <div>
                <label className="admin-modal-label">現在のパスワード</label>
                <Input
                  variant="default"
                  className="admin-modal-input"
                  type="password"
                  value={currentPassword}
                  onChange={(e) => setCurrentPassword(e.target.value)}
                  placeholder="現在のパスワード"
                />
              </div>

              <div>
                <label className="admin-modal-label">新しいパスワード</label>
                <Input
                  variant="default"
                  className="admin-modal-input"
                  type="password"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  placeholder="新しいパスワード"
                />
              </div>

              <div>
                <label className="admin-modal-label">新しいパスワード（確認）</label>
                <Input
                  variant="default"
                  className="admin-modal-input"
                  type="password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  placeholder="新しいパスワードを再入力"
                />
              </div>
            </div>

            <div className="admin-modal-footer">
              <button
                className="admin-modal-button cancel"
                type="button"
                onClick={() => setShowPasswordDialog(false)}
              >
                キャンセル
              </button>
              <button
                className="admin-modal-button primary"
                type="button"
                onClick={handlePasswordSave}
              >
                保存
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
