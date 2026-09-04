// frontend/src/pages/PasswordResetPage.tsx
import { useState } from "react";
import { sendPasswordResetEmail } from "firebase/auth";

import "../styles/page-layout.css";
import "../styles/form.css";
import "../styles/signIn-page.css";

import Layout from "../components/layout/Layout";
import Button from "../components/ui/Button";
import Input from "../components/ui/Input";
import { auth } from "../lib/firebase";

export default function PasswordResetPage() {
  const [email, setEmail] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

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
      setNotice(
        "パスワード再設定メールを送信しました。メールをご確認ください。"
      );
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

  return (
    <Layout title="パスワード再設定" showBackButton mode="signin">
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

          {error ? <p className="form-error-text">{error}</p> : null}

          {notice ? (
            <p
              style={{
                color: "#166534",
                fontSize: "14px",
                margin: 0,
              }}
            >
              {notice}
            </p>
          ) : null}
        </div>

        <div className="page-actions signin-page-actions">
          <Button
            variant="primary"
            onClick={handlePasswordReset}
            disabled={loading}
          >
            {loading ? "送信中..." : "再設定メールを送信"}
          </Button>
        </div>
      </section>
    </Layout>
  );
}