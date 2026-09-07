// frontend/mall/src/pages/SignInPage.tsx

import { useState } from "react";
import {
  useNavigate,
  useSearchParams,
} from "react-router-dom";
import { FirebaseError } from "firebase/app";
import { signInWithEmailAndPassword } from "firebase/auth";

import "../styles/page-layout.css";
import "../styles/form.css";
import "../styles/signIn-page.css";

import Layout from "../components/layout/Layout";
import Button from "../components/ui/Button";
import Input from "../components/ui/Input";
import { auth } from "../lib/firebase";

function resolveRedirectPath(
  redirect: string | null,
): string {
  if (
    !redirect ||
    !redirect.startsWith("/") ||
    redirect.startsWith("//")
  ) {
    return "/lists";
  }

  return redirect;
}

function resolveSignInErrorMessage(
  error: unknown,
): string {
  if (!(error instanceof FirebaseError)) {
    return "ログインに失敗しました。もう一度お試しください。";
  }

  switch (error.code) {
    case "auth/invalid-credential":
    case "auth/invalid-login-credentials":
    case "auth/user-not-found":
    case "auth/wrong-password":
      return "メールアドレスまたはパスワードが正しくありません。";

    case "auth/invalid-email":
      return "メールアドレスの形式が正しくありません。";

    case "auth/user-disabled":
      return "このアカウントは現在利用できません。";

    case "auth/too-many-requests":
      return "ログイン試行回数が多すぎます。しばらく時間をおいてから、もう一度お試しください。";

    case "auth/network-request-failed":
      return "通信に失敗しました。インターネット接続を確認して、もう一度お試しください。";

    default:
      return "ログインに失敗しました。もう一度お試しください。";
  }
}

export default function SignInPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

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

      await signInWithEmailAndPassword(
        auth,
        email,
        password,
      );

      const redirectPath = resolveRedirectPath(
        searchParams.get("redirect"),
      );

      navigate(redirectPath, {
        replace: true,
      });
    } catch (e) {
      setError(resolveSignInErrorMessage(e));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Layout
      title="AMOL"
      showBackButton
      mode="signin"
      backTo="/"
    >
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

            {error ? (
              <p className="form-error-text">
                {error}
              </p>
            ) : null}

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
            <Button
              variant="primary"
              onClick={handleSignIn}
              disabled={loading}
            >
              {loading ? "ログイン中..." : "ログイン"}
            </Button>
          </div>
        </div>
      </section>
    </Layout>
  );
}