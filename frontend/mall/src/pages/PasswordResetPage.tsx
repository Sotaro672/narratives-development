// frontend/mall/src/pages/PasswordResetPage.tsx

import { useState } from "react";
import { sendPasswordResetEmail } from "firebase/auth";
import { useNavigate } from "react-router-dom";

import "../styles/page-layout.css";
import "../styles/form.css";
import "../styles/signIn-page.css";

import { useMobilePortrait } from "../components/hooks/useMobilePortrait";
import Layout from "../components/layout/Layout";
import MobileSwipeRightDismissPage from "../components/layout/MobileSwipeRightDismissPage";
import Button from "../components/ui/Button";
import Input from "../components/ui/Input";
import TextState from "../components/ui/TextState";
import { auth } from "../lib/firebase";

export default function PasswordResetPage() {
  const navigate = useNavigate();
  const isMobilePortrait = useMobilePortrait();

  const [email, setEmail] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  const handleDismiss = () => {
    navigate(-1);
  };

  const handlePasswordReset = async () => {
    setError("");
    setNotice("");

    if (!email) {
      setError("現在使用中のメールアドレスを入力してください。");
      return;
    }

    try {
      setLoading(true);
      await sendPasswordResetEmail(auth, email);
      setNotice("パスワード再設定メールを送信しました。メールをご確認ください。");
    } catch (e) {
      if (e instanceof Error) {
        setError(e.message);
      } else {
        setError("パスワード再設定メールの送信に失敗しました。");
      }
    } finally {
      setLoading(false);
    }
  };

  const content = (
    <Layout
      title="パスワード再設定"
      titleClickable={false}
      mode="signin"
      showHeader={!isMobilePortrait}
      showBackButton={!isMobilePortrait}
      onBackButtonClick={handleDismiss}
    >
      <section className="page-section signin-page-section">
        <p className="page-description">
          現在使用中のメールアドレスを入力してください。パスワード再設定用のメールを送信します。
        </p>

        <div className="form-block signin-form-block">
          <Input
            label="現在使用中のメールアドレス"
            type="email"
            placeholder="example@email.com"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            autoComplete="email"
            fullWidth
          />

          {error ? <TextState variant="error">{error}</TextState> : null}

          {notice ? <TextState variant="success">{notice}</TextState> : null}
        </div>

        <div className="page-actions signin-page-actions">
          <Button
            variant="primary"
            onClick={handlePasswordReset}
            disabled={loading}
            style={{ width: "fit-content" }}
          >
            {loading ? "送信中..." : "再設定メールを送信"}
          </Button>
        </div>
      </section>
    </Layout>
  );

  if (!isMobilePortrait) {
    return content;
  }

  return (
    <MobileSwipeRightDismissPage
      title="パスワード再設定"
      onDismiss={handleDismiss}
    >
      {content}
    </MobileSwipeRightDismissPage>
  );
}