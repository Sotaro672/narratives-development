//frontend\admin\shell\src\pages\LoginPage.tsx
import {
  type FormEvent,
  useState,
} from "react";

import {
  authenticateAdmin,
  createAdminSession,
} from "../auth/adminAuth";

import "./LoginPage.css";

type LoginPageProps = {
  onLogin: () => void;
};

export default function LoginPage({
  onLogin,
}: LoginPageProps) {
  const [email, setEmail] = useState("");
  const [password, setPassword] =
    useState("");
  const [error, setError] = useState<
    string | null
  >(null);

  function handleSubmit(
    event: FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault();

    const authenticated =
      authenticateAdmin(
        email,
        password,
      );

    if (!authenticated) {
      setError(
        "メールアドレスまたはパスワードが正しくありません。",
      );
      return;
    }

    createAdminSession();
    setError(null);
    onLogin();
  }

  return (
    <div className="login-page">
      <div className="login-card">
        <div className="login-brand">
          Admin
        </div>

        <h1 className="login-title">
          ログイン
        </h1>

        <form
          className="login-form"
          onSubmit={handleSubmit}
        >
          <label className="login-field">
            <span>メールアドレス</span>

            <input
              type="email"
              value={email}
              onChange={(event) =>
                setEmail(
                  event.target.value,
                )
              }
              autoComplete="username"
              required
            />
          </label>

          <label className="login-field">
            <span>パスワード</span>

            <input
              type="password"
              value={password}
              onChange={(event) =>
                setPassword(
                  event.target.value,
                )
              }
              autoComplete="current-password"
              required
            />
          </label>

          {error && (
            <p
              className="login-error"
              role="alert"
            >
              {error}
            </p>
          )}

          <button
            type="submit"
            className="login-submit"
          >
            ログイン
          </button>
        </form>
      </div>
    </div>
  );
}