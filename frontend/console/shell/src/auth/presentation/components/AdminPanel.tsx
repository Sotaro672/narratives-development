// frontend/console/shell/src/auth/presentation/components/AdminPanel.tsx

import {
  LogOut,
} from "lucide-react";

import "../../../styles/auth.css";

import {
  Input,
} from "../../../shared/ui/input";
import {
  useAdminPanel,
} from "../hook/useAdminPanel";

interface AdminPanelProps {
  fullName?: string;
  email?: string;
  onLogout?: () => void;
  className?: string;
}

function getErrorCode(
  error: unknown,
): string {
  return error instanceof Error
    ? error.message
    : "";
}

export default function AdminPanel({
  fullName = "管理者",
  email = "",
  onLogout,
  className,
}: AdminPanelProps) {
  const {
    // ダイアログ
    showProfileDialog,
    setShowProfileDialog,
    showEmailDialog,
    setShowEmailDialog,
    showPasswordDialog,
    setShowPasswordDialog,

    // プロフィール
    lastName,
    setLastName,
    lastNameKana,
    setLastNameKana,
    firstName,
    setFirstName,
    firstNameKana,
    setFirstNameKana,

    // メールアドレス
    newEmail,
    setNewEmail,
    currentPasswordForEmail,
    setCurrentPasswordForEmail,

    // 保存処理
    saveProfile,
    saveEmail,
    savePassword,
  } = useAdminPanel();

  // -------------------------
  // プロフィール保存
  // -------------------------

  const handleProfileSave =
    async () => {
      try {
        await saveProfile();
      } catch (error: unknown) {
        console.error(
          "[AdminPanel] handleProfileSave error:",
          error,
        );

        const code =
          getErrorCode(error);

        switch (code) {
          case "MEMBER_NOT_FOUND":
            window.alert(
              "ログインユーザーのメンバー情報を確認できませんでした。",
            );
            break;

          case "KANA_INVALID":
            window.alert(
              "姓・名のかなはひらがなのみで入力してください。",
            );
            break;

          default:
            window.alert(
              "プロフィールの更新に失敗しました。",
            );
        }
      }
    };

  // -------------------------
  // メールアドレス変更
  // -------------------------

  const handleEmailSave =
    async () => {
      try {
        await saveEmail();

        window.alert(
          "メールアドレス変更用の認証メールを送信しました。メールに記載されたリンクから新しいメールアドレスを確認してください。",
        );
      } catch (error: unknown) {
        console.error(
          "[AdminPanel] handleEmailSave error:",
          error,
        );

        const code =
          getErrorCode(error);

        switch (code) {
          case "EMAIL_REQUIRED":
            window.alert(
              "新しいメールアドレスを入力してください。",
            );
            break;

          case "PASSWORD_REQUIRED":
            window.alert(
              "現在のパスワードを入力してください。",
            );
            break;

          case "AUTH_REAUTH_FAILED":
            window.alert(
              "再認証に失敗しました。パスワードを確認してください。",
            );
            break;

          case "AUTH_EMAIL_IN_USE":
            window.alert(
              "このメールアドレスは既に使用されています。",
            );
            break;

          case "AUTH_NO_USER":
            window.alert(
              "ログイン情報が見つかりません。再ログインしてください。",
            );
            break;

          default:
            window.alert(
              "認証メールの送信に失敗しました。",
            );
        }
      }
    };

  // -------------------------
  // パスワード再設定メール送信
  // -------------------------

  const handlePasswordSave =
    async () => {
      try {
        await savePassword();

        window.alert(
          "パスワード再設定用のメールを送信しました。メールに記載のリンクから新しいパスワードを設定してください。",
        );
      } catch (error: unknown) {
        console.error(
          "[AdminPanel] handlePasswordSave error:",
          error,
        );

        const code =
          getErrorCode(error);

        switch (code) {
          case "AUTH_NO_USER":
            window.alert(
              "ログイン情報が見つかりません。再ログインしてください。",
            );
            break;

          default:
            window.alert(
              "パスワード再設定メールの送信に失敗しました。",
            );
        }
      }
    };

  return (
    <>
      <div
        id="admin-dropdown"
        className={`admin-dropdown ${
          className ?? ""
        }`}
        role="menu"
        aria-label="アカウントメニュー"
      >
        <div className="admin-dropdown-header">
          <div className="admin-dropdown-title">
            {fullName}
          </div>

          {email && (
            <div className="admin-dropdown-email">
              {email}
            </div>
          )}
        </div>

        <div className="admin-dropdown-sep" />

        <button
          type="button"
          className="admin-dropdown-item"
          onClick={() =>
            setShowProfileDialog(true)
          }
        >
          プロフィール変更
        </button>

        <button
          type="button"
          className="admin-dropdown-item"
          onClick={() =>
            setShowEmailDialog(true)
          }
        >
          メールアドレス変更
        </button>

        <button
          type="button"
          className="admin-dropdown-item"
          onClick={() =>
            setShowPasswordDialog(true)
          }
        >
          パスワード変更
        </button>

        <div className="admin-dropdown-sep" />

        <button
          type="button"
          className="admin-dropdown-item logout"
          onClick={onLogout}
        >
          <LogOut
            className="logout-icon"
            aria-hidden
          />
          ログアウト
        </button>
      </div>

      {showProfileDialog && (
        <div className="admin-modal-backdrop">
          <div
            className="admin-modal"
            role="dialog"
            aria-modal="true"
            aria-labelledby="profile-dialog-title"
          >
            <div
              id="profile-dialog-title"
              className="admin-modal-title"
            >
              プロフィール変更
            </div>

            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="admin-modal-label">
                    姓
                  </label>

                  <Input
                    className="admin-modal-input"
                    value={lastName}
                    onChange={(event) =>
                      setLastName(
                        event.target.value,
                      )
                    }
                    placeholder="山田"
                  />
                </div>

                <div>
                  <label className="admin-modal-label">
                    姓（かな）
                  </label>

                  <Input
                    className="admin-modal-input"
                    value={lastNameKana}
                    onChange={(event) =>
                      setLastNameKana(
                        event.target.value,
                      )
                    }
                    placeholder="やまだ"
                    inputMode="text"
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="admin-modal-label">
                    名
                  </label>

                  <Input
                    className="admin-modal-input"
                    value={firstName}
                    onChange={(event) =>
                      setFirstName(
                        event.target.value,
                      )
                    }
                    placeholder="太郎"
                  />
                </div>

                <div>
                  <label className="admin-modal-label">
                    名（かな）
                  </label>

                  <Input
                    className="admin-modal-input"
                    value={firstNameKana}
                    onChange={(event) =>
                      setFirstNameKana(
                        event.target.value,
                      )
                    }
                    placeholder="たろう"
                    inputMode="text"
                  />
                </div>
              </div>
            </div>

            <div className="admin-modal-footer">
              <button
                type="button"
                className="admin-modal-button cancel"
                onClick={() =>
                  setShowProfileDialog(false)
                }
              >
                キャンセル
              </button>

              <button
                type="button"
                className="admin-modal-button primary"
                onClick={handleProfileSave}
              >
                保存
              </button>
            </div>
          </div>
        </div>
      )}

      {showEmailDialog && (
        <div className="admin-modal-backdrop">
          <div
            className="admin-modal"
            role="dialog"
            aria-modal="true"
            aria-labelledby="email-dialog-title"
          >
            <div
              id="email-dialog-title"
              className="admin-modal-title"
            >
              メールアドレス変更
            </div>

            <div className="space-y-4">
              <div>
                <label className="admin-modal-label">
                  新しいメールアドレス
                </label>

                <Input
                  className="admin-modal-input"
                  type="email"
                  value={newEmail}
                  onChange={(event) =>
                    setNewEmail(
                      event.target.value,
                    )
                  }
                  placeholder="new@example.com"
                />
              </div>

              <div>
                <label className="admin-modal-label">
                  パスワード
                </label>

                <Input
                  className="admin-modal-input"
                  type="password"
                  value={
                    currentPasswordForEmail
                  }
                  onChange={(event) =>
                    setCurrentPasswordForEmail(
                      event.target.value,
                    )
                  }
                  placeholder="現在のパスワード"
                />
              </div>
            </div>

            <div className="admin-modal-footer">
              <button
                type="button"
                className="admin-modal-button cancel"
                onClick={() =>
                  setShowEmailDialog(false)
                }
              >
                キャンセル
              </button>

              <button
                type="button"
                className="admin-modal-button primary"
                onClick={handleEmailSave}
              >
                認証メールを送信
              </button>
            </div>
          </div>
        </div>
      )}

      {showPasswordDialog && (
        <div className="admin-modal-backdrop">
          <div
            className="admin-modal"
            role="dialog"
            aria-modal="true"
            aria-labelledby="password-dialog-title"
          >
            <div
              id="password-dialog-title"
              className="admin-modal-title"
            >
              パスワード変更
            </div>

            <div className="space-y-4">
              <p className="admin-modal-text">
                現在ログイン中のメールアドレス宛に、
                パスワード再設定用のメールを送信します。
                メールに記載されたリンクから新しいパスワードを設定してください。
              </p>
            </div>

            <div className="admin-modal-footer">
              <button
                type="button"
                className="admin-modal-button cancel"
                onClick={() =>
                  setShowPasswordDialog(false)
                }
              >
                キャンセル
              </button>

              <button
                type="button"
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