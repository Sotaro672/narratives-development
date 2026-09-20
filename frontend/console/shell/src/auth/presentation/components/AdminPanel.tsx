// frontend/console/shell/src/auth/presentation/components/AdminPanel.tsx

import { LogOut } from "lucide-react";

import "../../../styles/auth.css";

import { Button } from "../../../shared/ui/button";
import { Input } from "../../../shared/ui/input";
import { Label } from "../../../shared/ui/label";
import { Modal } from "../../../shared/ui/modal";
import { Separator } from "../../../shared/ui/separator";
import { useAdminPanel } from "../hook/useAdminPanel";

interface AdminPanelProps {
  fullName?: string;
  email?: string;
  onLogout?: () => void;
  className?: string;
}

function getErrorCode(error: unknown): string {
  return error instanceof Error ? error.message : "";
}

export default function AdminPanel({
  fullName = "管理者",
  email = "",
  onLogout,
  className,
}: AdminPanelProps) {
  const {
    showProfileDialog,
    setShowProfileDialog,
    showEmailDialog,
    setShowEmailDialog,
    showPasswordDialog,
    setShowPasswordDialog,
    lastName,
    setLastName,
    lastNameKana,
    setLastNameKana,
    firstName,
    setFirstName,
    firstNameKana,
    setFirstNameKana,
    newEmail,
    setNewEmail,
    currentPasswordForEmail,
    setCurrentPasswordForEmail,
    saveProfile,
    saveEmail,
    savePassword,
  } = useAdminPanel();

  const handleProfileSave = async () => {
    try {
      await saveProfile();
    } catch (error: unknown) {
      const code = getErrorCode(error);

      switch (code) {
        case "MEMBER_NOT_FOUND":
          window.alert("ログインユーザーのメンバー情報を確認できませんでした。");
          break;
        case "KANA_INVALID":
          window.alert("姓・名のかなはひらがなのみで入力してください。");
          break;
        default:
          window.alert("プロフィールの更新に失敗しました。");
      }
    }
  };

  const handleEmailSave = async () => {
    try {
      await saveEmail();
      window.alert(
        "メールアドレス変更用の認証メールを送信しました。メールに記載されたリンクから新しいメールアドレスを確認してください。",
      );
    } catch (error: unknown) {
      const code = getErrorCode(error);

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

  const handlePasswordSave = async () => {
    try {
      await savePassword();
      window.alert(
        "パスワード再設定用のメールを送信しました。メールに記載のリンクから新しいパスワードを設定してください。",
      );
    } catch (error: unknown) {
      const code = getErrorCode(error);

      switch (code) {
        case "AUTH_NO_USER":
          window.alert("ログイン情報が見つかりません。再ログインしてください。");
          break;
        default:
          window.alert("パスワード再設定メールの送信に失敗しました。");
      }
    }
  };

  return (
    <>
      <div
        id="admin-dropdown"
        className={`admin-dropdown ${className ?? ""}`.trim()}
        role="menu"
        aria-label="アカウントメニュー"
      >
        <div className="admin-dropdown-header">
          <div className="admin-dropdown-title">{fullName}</div>
          {email && <div className="admin-dropdown-email">{email}</div>}
        </div>

        <Separator className="admin-dropdown-sep" />

        <Button
          type="button"
          variant="ghost"
          className="admin-dropdown-item"
          role="menuitem"
          onClick={() => setShowProfileDialog(true)}
        >
          プロフィール変更
        </Button>

        <Button
          type="button"
          variant="ghost"
          className="admin-dropdown-item"
          role="menuitem"
          onClick={() => setShowEmailDialog(true)}
        >
          メールアドレス変更
        </Button>

        <Button
          type="button"
          variant="ghost"
          className="admin-dropdown-item"
          role="menuitem"
          onClick={() => setShowPasswordDialog(true)}
        >
          パスワード変更
        </Button>

        <Separator className="admin-dropdown-sep" />

        <Button
          type="button"
          variant="ghost"
          className="admin-dropdown-item logout"
          role="menuitem"
          onClick={onLogout}
        >
          <LogOut className="logout-icon" aria-hidden />
          ログアウト
        </Button>
      </div>

      <Modal
        open={showProfileDialog}
        title="プロフィール変更"
        onClose={() => setShowProfileDialog(false)}
        footer={
          <>
            <Button
              type="button"
              variant="outline"
              onClick={() => setShowProfileDialog(false)}
            >
              キャンセル
            </Button>

            <Button
              type="button"
              onClick={() => void handleProfileSave()}
            >
              保存
            </Button>
          </>
        }
      >
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-2">
              <Label htmlFor="admin-profile-last-name">姓</Label>
              <Input
                id="admin-profile-last-name"
                value={lastName}
                onChange={(event) => setLastName(event.target.value)}
                placeholder="山田"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="admin-profile-last-name-kana">姓（かな）</Label>
              <Input
                id="admin-profile-last-name-kana"
                value={lastNameKana}
                onChange={(event) => setLastNameKana(event.target.value)}
                placeholder="やまだ"
                inputMode="text"
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-2">
              <Label htmlFor="admin-profile-first-name">名</Label>
              <Input
                id="admin-profile-first-name"
                value={firstName}
                onChange={(event) => setFirstName(event.target.value)}
                placeholder="太郎"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="admin-profile-first-name-kana">名（かな）</Label>
              <Input
                id="admin-profile-first-name-kana"
                value={firstNameKana}
                onChange={(event) => setFirstNameKana(event.target.value)}
                placeholder="たろう"
                inputMode="text"
              />
            </div>
          </div>
        </div>
      </Modal>

      <Modal
        open={showEmailDialog}
        title="メールアドレス変更"
        onClose={() => setShowEmailDialog(false)}
        footer={
          <>
            <Button
              type="button"
              variant="outline"
              onClick={() => setShowEmailDialog(false)}
            >
              キャンセル
            </Button>

            <Button
              type="button"
              onClick={() => void handleEmailSave()}
            >
              認証メールを送信
            </Button>
          </>
        }
      >
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="admin-email-new">新しいメールアドレス</Label>
            <Input
              id="admin-email-new"
              type="email"
              value={newEmail}
              onChange={(event) => setNewEmail(event.target.value)}
              placeholder="new@example.com"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="admin-email-current-password">パスワード</Label>
            <Input
              id="admin-email-current-password"
              type="password"
              value={currentPasswordForEmail}
              onChange={(event) =>
                setCurrentPasswordForEmail(event.target.value)
              }
              placeholder="現在のパスワード"
            />
          </div>
        </div>
      </Modal>

      <Modal
        open={showPasswordDialog}
        title="パスワード変更"
        description="現在ログイン中のメールアドレス宛に、パスワード再設定用のメールを送信します。メールに記載されたリンクから新しいパスワードを設定してください。"
        onClose={() => setShowPasswordDialog(false)}
        footer={
          <>
            <Button
              type="button"
              variant="outline"
              onClick={() => setShowPasswordDialog(false)}
            >
              キャンセル
            </Button>

            <Button
              type="button"
              onClick={() => void handlePasswordSave()}
            >
              再設定メールを送信
            </Button>
          </>
        }
      />
    </>
  );
}