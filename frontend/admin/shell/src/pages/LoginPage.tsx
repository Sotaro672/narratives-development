// frontend/admin/shell/src/pages/LoginPage.tsx

import { type FormEvent, useState } from "react";

import { signInAdmin } from "../auth/application/adminAuth";

import "./LoginPage.css";

type LoginPageProps = {
  onLogin: () => void;
};

function resolveLoginErrorMessage(error: unknown): string {
  if (error instanceof Error && error.message === "invalid_admin_password") {
    return "パスワードが正しくありません。";
  }

  return "ログインに失敗しました。もう一度お試しください。";
}

export default function LoginPage({ onLogin }: LoginPageProps) {
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (submitting) {
      return;
    }

    if (!password) {
      setError("パスワードを入力してください。");
      return;
    }

    setSubmitting(true);
    setError(null);

    try {
      await signInAdmin(password);
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
            <span>パスワード</span>
            <input
              type="password"
              value={password}
              onChange={(event) => {
                setPassword(event.target.value);
                setError(null);
              }}
              autoComplete="current-password"
              disabled={submitting}
              autoFocus
              required
            />
          </label>

          {error ? (
            <p className="login-error" role="alert">
              {error}
            </p>
          ) : null}

          <button type="submit" className="login-submit" disabled={submitting}>
            {submitting ? "ログイン中..." : "ログイン"}
          </button>
        </form>
      </div>
    </div>
  );
}