// frontend/admin/shell/src/pages/LoginPage.tsx
import { type FormEvent, useState } from "react";

import { signInAdmin } from "../auth/application/adminAuth";

import "./LoginPage.css";

type LoginPageProps = {
  onLogin: () => void;
};

const EMAIL_VERIFICATION_MESSAGE =
  "メールアドレスが未認証です。認証メールを送信しました。メール内のリンクから認証してください。";

function resolveLoginErrorMessage(error: unknown): string {
  if (error instanceof Error && error.message === EMAIL_VERIFICATION_MESSAGE) {
    return error.message;
  }

  return "メールアドレスまたはパスワードが正しくないか、Admin権限がありません。";
}

export default function LoginPage({ onLogin }: LoginPageProps) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (submitting) {
      return;
    }

    setSubmitting(true);
    setError(null);

    try {
      await signInAdmin(email, password);
      onLogin();
    } catch (error) {
      console.error("[admin-login] sign in failed", error);
      setError(resolveLoginErrorMessage(error));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="login-page">
      <div className="login-card">
        <div className="login-brand">Admin</div>
        <h1 className="login-title">ログイン</h1>

        <form className="login-form" onSubmit={handleSubmit}>
          <label className="login-field">
            <span>メールアドレス</span>
            <input
              type="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              autoComplete="username"
              disabled={submitting}
              required
            />
          </label>

          <label className="login-field">
            <span>パスワード</span>
            <input
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              autoComplete="current-password"
              disabled={submitting}
              required
            />
          </label>

          {error && (
            <p className="login-error" role="alert">
              {error}
            </p>
          )}

          <button type="submit" className="login-submit" disabled={submitting}>
            {submitting ? "ログイン中..." : "ログイン"}
          </button>
        </form>
      </div>
    </div>
  );
}