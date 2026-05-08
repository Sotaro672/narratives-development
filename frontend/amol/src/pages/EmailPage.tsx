//frontend\src\pages\EmailPage.tsx
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  EmailAuthProvider,
  getAuth,
  reauthenticateWithCredential,
  verifyBeforeUpdateEmail,
} from "firebase/auth";

import "../styles/page-layout.css";
import "../styles/form.css";

import Layout from "../components/layout/Layout";
import Input from "../components/ui/Input";
import Button from "../components/ui/Button";

export default function EmailPage() {
  const navigate = useNavigate();
  const [currentEmail, setCurrentEmail] = useState("");
  const [newEmail, setNewEmail] = useState("");
  const [password, setPassword] = useState("");
  const [saving, setSaving] = useState(false);

  const handleSave = async () => {
    try {
      const trimmedCurrentEmail = currentEmail.trim();
      const trimmedNewEmail = newEmail.trim();

      if (!trimmedCurrentEmail) {
        window.alert("現在のメールアドレスを入力してください。");
        return;
      }

      if (!trimmedNewEmail) {
        window.alert("新しいメールアドレスを入力してください。");
        return;
      }

      if (!password) {
        window.alert("パスワードを入力してください。");
        return;
      }

      if (trimmedCurrentEmail === trimmedNewEmail) {
        window.alert(
          "新しいメールアドレスは現在のメールアドレスと別のものを入力してください。"
        );
        return;
      }

      setSaving(true);

      const auth = getAuth();
      const user = auth.currentUser;

      if (!user) {
        window.alert("ログイン情報が見つかりません。再度ログインしてください。");
        navigate("/signin");
        return;
      }

      const emailForCredential = user.email ?? trimmedCurrentEmail;

      const credential = EmailAuthProvider.credential(
        emailForCredential,
        password
      );

      await reauthenticateWithCredential(user, credential);

      await verifyBeforeUpdateEmail(user, trimmedNewEmail, {
        url: `${window.location.origin}/settings/email`,
        handleCodeInApp: false,
      });

      window.alert(
        "新しいメールアドレス宛に確認メールを送信しました。メール内のリンクを開いて変更を完了してください。"
      );
      navigate("/settings");
    } catch (error) {
      console.error(error);

      const firebaseError = error as { code?: string };

      switch (firebaseError.code) {
        case "auth/wrong-password":
          window.alert("パスワードが正しくありません。");
          break;
        case "auth/invalid-email":
          window.alert("メールアドレスの形式が正しくありません。");
          break;
        case "auth/email-already-in-use":
          window.alert("このメールアドレスは既に使用されています。");
          break;
        case "auth/requires-recent-login":
          window.alert("再度ログインしてからお試しください。");
          navigate("/signin");
          break;
        case "auth/user-mismatch":
        case "auth/user-not-found":
          window.alert("現在のメールアドレスまたは認証情報が一致しません。");
          break;
        case "auth/operation-not-allowed":
          window.alert(
            "メールアドレス変更が許可されていません。Firebase のメール確認設定を確認してください。"
          );
          break;
        default:
          window.alert("メールアドレスの更新に失敗しました。");
          break;
      }
    } finally {
      setSaving(false);
    }
  };

  return (
    <Layout
      title="メールアドレス変更"
      showBackButton
      mode="signin"
      backTo="/lists"
    >
      <section className="page-section">
        <p className="page-description">
          現在の情報を入力してメールアドレスを変更してください。
        </p>

        <div className="form-block">
          <Input
            id="current-email"
            name="currentEmail"
            label="現在のメールアドレス"
            type="email"
            value={currentEmail}
            onChange={(e) => setCurrentEmail(e.target.value)}
            placeholder="現在のメールアドレスを入力"
            autoComplete="email"
            disabled={saving}
          />

          <Input
            id="new-email"
            name="newEmail"
            label="新しいメールアドレス"
            type="email"
            value={newEmail}
            onChange={(e) => setNewEmail(e.target.value)}
            placeholder="新しいメールアドレスを入力"
            autoComplete="off"
            disabled={saving}
          />

          <Input
            id="current-password"
            name="currentPassword"
            label="パスワード"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="パスワードを入力"
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
            {saving ? "送信中..." : "確認メールを送信"}
          </Button>
        </div>
      </section>
    </Layout>
  );
}