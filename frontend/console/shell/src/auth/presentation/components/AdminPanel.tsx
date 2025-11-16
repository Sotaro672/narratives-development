// frontend/console/shell/src/auth/presentation/components/AdminPanel.tsx
import { LogOut } from "lucide-react";
import "../styles/auth.css";
import { Input } from "../../../shared/ui/input";
import { useAdminPanel } from "../hook/useAdminPanel";

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

    // ★ 追加: 保存処理
    saveProfile,
  } = useAdminPanel();

  if (!open) return null;

  const handleProfileSave = async () => {
    // Backend 経由でプロフィールを更新
    await saveProfile();
    // 追加で、親に「更新完了」を通知したければここで呼ぶ
    onEditProfile?.();
  };

  const handleEmailSave = () => {
    onChangeEmail?.();
    setShowEmailDialog(false);
  };

  const handlePasswordSave = () => {
    onChangePassword?.();
    setShowPasswordDialog(false);
  };

  // 以下 JSX はそのままでOK（既に value に state をバインドしているので、
  // useAdminPanel が currentMember でプリフィルしてくれている）
  return (
    <>
      {/* ドロップダウン本体 */}
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

      {/* プロフィール変更ダイアログ */}
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

      {/* メール・パスワード変更ダイアログはそのまま */}
      {/* ... */}
    </>
  );
}
