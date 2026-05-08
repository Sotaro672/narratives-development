// frontend/amol/src/pages/SignInPage.tsx
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { signInWithEmailAndPassword } from "firebase/auth";

import "../styles/page-layout.css";
import "../styles/form.css";
import "../styles/signIn-page.css";

import Layout from "../components/layout/Layout";
import Button from "../components/ui/Button";
import Input from "../components/ui/Input";
import { auth } from "../lib/firebase";

export default function SignInPage() {
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleSignIn = async () => {
    setError("");

    if (!email || !password) {
      setError("メールアドレスとパスワードを入力してください。");
      return;
    }

    try {
      setLoading(true);
      await signInWithEmailAndPassword(auth, email, password);
      navigate("/lists");
    } catch (e) {
      if (e instanceof Error) {
        setError(e.message);
      } else {
        setError("ログインに失敗しました。");
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <Layout title="ログイン" showBackButton mode="signin" backTo="/">
      <section className="signin-page-section">
        <div className="signin-page-section__inner">
          <p className="page-description signin-page-description">
            メールアドレスとパスワードを入力してログインしてください。
          </p>

          <div className="form-block signin-form-block">
            <Input
              label="メールアドレス"
              type="email"
              placeholder="example@email.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              autoComplete="email"
              fullWidth
            />

            <Input
              label="パスワード"
              type="password"
              placeholder="パスワードを入力"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="current-password"
              fullWidth
            />

            {error ? <p className="form-error-text">{error}</p> : null}

            <button
              type="button"
              onClick={() => navigate("/password-reset")}
              className="form-link-button"
            >
              パスワードを忘れた方はこちら
            </button>

            <button
              type="button"
              onClick={() => navigate("/signup")}
              className="form-link-button"
            >
              新規登録はこちら
            </button>
          </div>

          <div className="signin-page-actions">
            <Button variant="primary" onClick={handleSignIn} disabled={loading}>
              {loading ? "ログイン中..." : "ログイン"}
            </Button>
          </div>
        </div>
      </section>
    </Layout>
  );
}