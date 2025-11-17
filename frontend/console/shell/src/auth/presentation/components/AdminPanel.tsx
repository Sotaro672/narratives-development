//frontend\console\shell\src\auth\presentation\components\AdminPanel.tsx
import { LogOut } from "lucide-react";
import "../styles/auth.css";
import { Input } from "../../../shared/ui/input";
import { useAdminPanel } from "../hook/useAdminPanel";
import {
  changeEmail,
  changePassword,
} from "../../application/profileService";

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
  const {
    // dialog flags
    showProfileDialog,
    setShowProfileDialog,
    showEmailDialog,
    setShowEmailDialog,
    showPasswordDialog,
    setShowPasswordDialog,

    // profile fields
    lastName,
    setLastName,
    lastNameKana,
    setLastNameKana,
    firstName,
    setFirstName,
    firstNameKana,
    setFirstNameKana,

    // email fields
    newEmail,
    setNewEmail,
    currentPasswordForEmail,
    setCurrentPasswordForEmail,

    // password fields
    currentPassword,
    setCurrentPassword,
    newPassword,
    setNewPassword,
    confirmPassword,
    setConfirmPassword,

    // backend profile update
    saveProfile,
  } = useAdminPanel();

  if (!open) return null;

  // ─────────────────────────────
  // プロフィール更新（backend PATCH）
  // ─────────────────────────────
  const handleProfileSave = async () => {
    try {
      await saveProfile();
      onEditProfile?.();
      setShowProfileDialog(false);
    } catch (e) {
      console.error("[AdminPanel] handleProfileSave error:", e);
      window.alert("プロフィールの更新に失敗しました。");
    }
  };

  // ─────────────────────────────
  // メールアドレス変更（Firebase Auth）
  // ─────────────────────────────
  const handleEmailSave = async () => {
    try {
      if (!newEmail.trim()) {
        window.alert("新しいメールアドレスを入力してください。");
        return;
      }
      if (!currentPasswordForEmail) {
        window.alert("現在のパスワードを入力してください。");
        return;
      }

      await changeEmail(currentPasswordForEmail, newEmail.trim());

      onChangeEmail?.();
      setShowEmailDialog(false);

      // reset
      setNewEmail("");
      setCurrentPasswordForEmail("");
    } catch (e) {
      console.error("[AdminPanel] handleEmailSave error:", e);
      window.alert("メールアドレス変更に失敗しました。");
    }
  };

  // ─────────────────────────────
  // パスワード変更（Firebase Auth）
  // ─────────────────────────────
  const handlePasswordSave = async () => {
    try {
      if (!currentPassword) {
        window.alert("現在のパスワードを入力してください。");
        return;
      }
      if (!newPassword) {
        window.alert("新しいパスワードを入力してください。");
        return;
      }
      if (newPassword !== confirmPassword) {
        window.alert("新しいパスワードと確認用パスワードが一致していません。");
        return;
      }

      await changePassword(currentPassword, newPassword);

      onChangePassword?.();
      setShowPasswordDialog(false);

      // reset
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
    } catch (e) {
      console.error("[AdminPanel] handlePasswordSave error:", e);
      window.alert("パスワード変更に失敗しました。");
    }
  };

  // ==================================================================
  // JSX
  // ==================================================================
  return (
    <>
      {/* メニュー本体 */}
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
          onClick={() => setShowProfileDialog(true)}
        >
          プロフィール変更
        </button>

        <button
          className="admin-dropdown-item"
          onClick={() => setShowEmailDialog(true)}
        >
          メールアドレス変更
        </button>

        <button
          className="admin-dropdown-item"
          onClick={() => setShowPasswordDialog(true)}
        >
          パスワード変更
        </button>

        <div className="admin-dropdown-sep" />

        <button
          className="admin-dropdown-item logout"
          onClick={onLogout}
        >
          <LogOut className="logout-icon" />
          ログアウト
        </button>
      </div>

      {/* ---------------------------------- */}
      {/* プロフィール変更ダイアログ */}
      {/* ---------------------------------- */}
      {showProfileDialog && (
        <div className="admin-modal-backdrop" aria-modal="true">
          <div className="admin-modal">
            <div className="admin-modal-title">プロフィール変更</div>

            <div className="space-y-4">
              {/* 姓 / 姓かな */}
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="admin-modal-label">姓</label>
                  <Input
                    className="admin-modal-input"
                    value={lastName}
                    onChange={(e) => setLastName(e.target.value)}
                    placeholder="山田"
                  />
                </div>
                <div>
                  <label className="admin-modal-label">姓（かな）</label>
                  <Input
                    className="admin-modal-input"
                    value={lastNameKana}
                    onChange={(e) => setLastNameKana(e.target.value)}
                    placeholder="やまだ"
                  />
                </div>
              </div>

              {/* 名 / 名かな */}
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="admin-modal-label">名</label>
                  <Input
                    className="admin-modal-input"
                    value={firstName}
                    onChange={(e) => setFirstName(e.target.value)}
                    placeholder="太郎"
                  />
                </div>
                <div>
                  <label className="admin-modal-label">名（かな）</label>
                  <Input
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
                onClick={() => setShowProfileDialog(false)}
              >
                キャンセル
              </button>
              <button
                className="admin-modal-button primary"
                onClick={handleProfileSave}
              >
                保存
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ---------------------------------- */}
      {/* メールアドレス変更 */}
      {/* ---------------------------------- */}
      {showEmailDialog && (
        <div className="admin-modal-backdrop" aria-modal="true">
          <div className="admin-modal">
            <div className="admin-modal-title">メールアドレス変更</div>

            <div className="space-y-4">
              <div>
                <label className="admin-modal-label">新しいメールアドレス</label>
                <Input
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
                onClick={() => setShowEmailDialog(false)}
              >
                キャンセル
              </button>
              <button
                className="admin-modal-button primary"
                onClick={handleEmailSave}
              >
                保存
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ---------------------------------- */}
      {/* パスワード変更 */}
      {/* ---------------------------------- */}
      {showPasswordDialog && (
        <div className="admin-modal-backdrop" aria-modal="true">
          <div className="admin-modal">
            <div className="admin-modal-title">パスワード変更</div>

            <div className="space-y-4">
              <div>
                <label className="admin-modal-label">現在のパスワード</label>
                <Input
                  className="admin-modal-input"
                  type="password"
                  value={currentPassword}
                  onChange={(e) =>
                    setCurrentPassword(e.target.value)
                  }
                  placeholder="現在のパスワード"
                />
              </div>

              <div>
                <label className="admin-modal-label">新しいパスワード</label>
                <Input
                  className="admin-modal-input"
                  type="password"
                  value={newPassword}
                  onChange={(e) =>
                    setNewPassword(e.target.value)
                  }
                  placeholder="新しいパスワード"
                />
              </div>

              <div>
                <label className="admin-modal-label">
                  新しいパスワード（確認）
                </label>
                <Input
                  className="admin-modal-input"
                  type="password"
                  value={confirmPassword}
                  onChange={(e) =>
                    setConfirmPassword(e.target.value)
                  }
                  placeholder="新しいパスワードを再入力"
                />
              </div>
            </div>

            <div className="admin-modal-footer">
              <button
                className="admin-modal-button cancel"
                onClick={() => setShowPasswordDialog(false)}
              >
                キャンセル
              </button>
              <button
                className="admin-modal-button primary"
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
