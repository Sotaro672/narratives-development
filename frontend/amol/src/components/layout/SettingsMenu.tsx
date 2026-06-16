// frontend/amol/src/components/layout/SettingsMenu.tsx
import { useState } from "react";
import {
  EmailAuthProvider,
  getAuth,
  reauthenticateWithCredential,
  signOut,
} from "firebase/auth";
import { useNavigate } from "react-router-dom";

import Item from "../ui/Item";

type SettingsMenuProps = {
  onItemClick?: () => void;
};

export default function SettingsMenu({ onItemClick }: SettingsMenuProps) {
  const navigate = useNavigate();
  const auth = getAuth();

  const [isDeleting, setIsDeleting] = useState(false);

  const handleNavigate = (to: string) => {
    onItemClick?.();
    navigate(to);
  };

  const handleLogout = async () => {
    try {
      await signOut(auth);
      onItemClick?.();
      navigate("/signin", { replace: true });
    } catch (error) {
      console.error(error);
      window.alert("ログアウトに失敗しました。");
    }
  };

  const handleDeleteAccount = async () => {
    const currentUser = auth.currentUser;

    if (!currentUser || !currentUser.email) {
      window.alert("ログイン情報を確認できませんでした。");
      return;
    }

    const confirmed = window.confirm(
      "本当にアカウントを削除しますか？\nFirebase Auth、Firestore、秘密鍵が削除されます。"
    );
    if (!confirmed) return;

    const password = window.prompt(
      "本人確認のため現在のパスワードを入力してください。"
    );
    if (!password) return;

    try {
      setIsDeleting(true);

      const credential = EmailAuthProvider.credential(
        currentUser.email,
        password
      );

      await reauthenticateWithCredential(currentUser, credential);

      const idToken = await currentUser.getIdToken(true);
      const backendUrl = import.meta.env.VITE_API_BASE_URL;

      if (!backendUrl) {
        throw new Error("VITE_API_BASE_URL が設定されていません。");
      }

      const response = await fetch(`${backendUrl}/api/me`, {
        method: "DELETE",
        headers: {
          Authorization: `Bearer ${idToken}`,
        },
      });

      let responseBody: { error?: string; status?: string } | null = null;
      const contentType = response.headers.get("content-type") || "";

      if (contentType.includes("application/json")) {
        responseBody = await response.json();
      }

      if (!response.ok) {
        throw new Error(
          responseBody?.error || "アカウント削除に失敗しました。"
        );
      }

      await signOut(auth);
      onItemClick?.();
      navigate("/signin", { replace: true });
    } catch (error) {
      console.error(error);

      if (error instanceof Error) {
        window.alert(error.message);
      } else {
        window.alert("アカウント削除に失敗しました。");
      }
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <ul className="settings-list">
      <Item
        label="メールアドレス変更"
        onClick={() => handleNavigate("/settings/email")}
      />

      <Item
        label="パスワード変更"
        onClick={() => handleNavigate("/settings/password")}
      />

      <Item
        label="支払方法"
        onClick={() => handleNavigate("/settings/payment-method")}
      />

      <Item
        label="配送先情報"
        onClick={() => handleNavigate("/settings/shipping-address")}
      />

      <Item label="ログアウト" onClick={handleLogout} />

      <Item
        label={isDeleting ? "削除中..." : "アカウント削除"}
        danger
        onClick={isDeleting ? undefined : handleDeleteAccount}
      />
    </ul>
  );
}