//frontend\src\pages\PasswordPage.tsx
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  EmailAuthProvider,
  getAuth,
  reauthenticateWithCredential,
  sendPasswordResetEmail,
} from "firebase/auth";

import "../styles/page-layout.css";
import "../styles/form.css";

import Layout from "../components/layout/Layout";
import Input from "../components/ui/Input";
import Button from "../components/ui/Button";

export default function PasswordPage() {
  const navigate = useNavigate();
  const [currentPassword, setCurrentPassword] = useState("");
  const [saving, setSaving] = useState(false);

  const handleSave = async () => {
    try {
      if (!currentPassword) {
        window.alert("現在のパスワードを入力してください。");
        return;
      }

      setSaving(true);

      const auth = getAuth();
      const user = auth.currentUser;

      if (!user || !user.email) {
        window.alert("ログイン情報が見つかりません。再度ログインしてください。");
        navigate("/signin");
        return;
      }

      const credential = EmailAuthProvider.credential(
        user.email,
        currentPassword
      );

      await reauthenticateWithCredential(user, credential);

      await sendPasswordResetEmail(auth, user.email, {
        url: `${window.location.origin}/signin`,
        handleCodeInApp: false,
      });

      window.alert(
        "パスワード再設定メールを送信しました。メール内のリンクを開いて新しいパスワードを設定してください。"
      );
      navigate("/settings");
    } catch (error) {
      console.error(error);

      const firebaseError = error as { code?: string };

      switch (firebaseError.code) {
        case "auth/wrong-password":
          window.alert("現在のパスワードが正しくありません。");
          break;
        case "auth/user-not-found":
          window.alert("ユーザー情報が見つかりません。");
          break;
        case "auth/invalid-email":
          window.alert("メールアドレスの形式が正しくありません。");
          break;
        case "auth/requires-recent-login":
          window.alert("再度ログインしてからお試しください。");
          navigate("/signin");
          break;
        default:
          window.alert("パスワード再設定メールの送信に失敗しました。");
          break;
      }
    } finally {
      setSaving(false);
    }
  };

  return (
    <Layout
      title="パスワード変更"
      showBackButton
      mode="signin"
      backTo="/lists"
    >
      <section className="page-section">
        <p className="page-description">
          現在のパスワードを入力すると、登録メールアドレス宛にパスワード再設定メールを送信します。
        </p>

        <div className="form-block">
          <Input
            id="current-password"
            name="currentPassword"
            label="現在のパスワード"
            type="password"
            value={currentPassword}
            onChange={(e) => setCurrentPassword(e.target.value)}
            placeholder="現在のパスワードを入力"
            autoComplete="current-password"
            disabled={saving}
          />
        </div>

        <div className="page-actions">
          <Button
            variant="primary"
            size="md"
            onClick={handleSave}
            disabled={saving}
          >
            {saving ? "送信中..." : "再設定メールを送信"}
          </Button>
        </div>
      </section>
    </Layout>
  );
}