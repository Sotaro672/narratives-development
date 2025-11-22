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

    // handlers
    saveProfile,
    saveEmail,
    savePassword,
  } = useAdminPanel();

  if (!open) return null;

  // ─────────────────────────────
  // ユーティリティ: カタカナ判定
  // ─────────────────────────────
  const isKatakana = (value: string): boolean => {
    const v = value.trim();
    if (!v) return true; // 空は許可
    // 全角カタカナ + 長音符 + スペース
    const katakanaRegex = /^[\u30A0-\u30FFー\s]+$/;
    return katakanaRegex.test(v);
  };

  // ─────────────────────────────
  // プロフィール更新（backend PATCH）
  // ─────────────────────────────
  const handleProfileSave = async () => {
    try {
      // ひらがなバリデーション
      if (!isKatakana(lastNameKana) || !isKatakana(firstNameKana)) {
        window.alert("フリガナはひらがなで入力してください。");
        return;
      }

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
      await saveEmail();

      onChangeEmail?.();
      setShowEmailDialog(false);

      window.alert(
        "メールアドレス変更用の認証メールを送信しました。メールに記載されたリンクから新しいメールアドレスを確認してください。",
      );
    } catch (e: any) {
      console.error("[AdminPanel] handleEmailSave error:", e);

      const code = e?.message;
      switch (code) {
        case "EMAIL_REQUIRED":
          window.alert("新しいメールアドレスを入力してください。");
          break;
        case "PASSWORD_REQUIRED":
          window.alert("現在のパスワードを入力してください。");
          break;
        case "AUTH_REAUTH_FAILED":
          window.alert("再認証に失敗しました。パスワードを確認してください。");
          break;
        case "AUTH_EMAIL_IN_USE":
          window.alert("このメールアドレスは既に使用されています。");
          break;
        case "AUTH_NO_USER":
          window.alert("ログイン情報が見つかりません。再ログインしてください。");
          break;
        default:
          window.alert("認証メールの送信に失敗しました。");
      }
    }
  };

  // ─────────────────────────────
  // パスワード再設定メール送信
  // ─────────────────────────────
  const handlePasswordSave = async () => {
    try {
      await savePassword();

      onChangePassword?.();
      setShowPasswordDialog(false);

      window.alert(
        "パスワード再設定用のメールを送信しました。メールに記載のリンクから新しいパスワードを設定してください。",
      );
    } catch (e: any) {
      console.error("[AdminPanel] handlePasswordSave error:", e);

      const code = e?.message;
      switch (code) {
        case "AUTH_NO_USER":
          window.alert("ログイン情報が見つかりません。再ログインしてください。");
          break;
        default:
          window.alert("パスワード再設定メールの送信に失敗しました。");
      }
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

        <button className="admin-dropdown-item logout" onClick={onLogout}>
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
                <label className="admin-modal-label">パスワード</label>
                <Input
                  className="admin-modal-input"
                  type="password"
                  value={currentPasswordForEmail}
                  onChange={(e) => setCurrentPasswordForEmail(e.target.value)}
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
                認証メールを送信
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ---------------------------------- */}
      {/* パスワード変更 → 再設定メール送信ダイアログ */}
      {/* ---------------------------------- */}
      {showPasswordDialog && (
        <div className="admin-modal-backdrop" aria-modal="true">
          <div className="admin-modal">
            <div className="admin-modal-title">パスワード変更</div>

            <div className="space-y-4">
              <p className="admin-modal-text">
                現在ログイン中のメールアドレス宛に、
                パスワード再設定用のメールを送信します。
                メールに記載されたリンクから新しいパスワードを設定してください。
              </p>
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
                再設定メールを送信
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
