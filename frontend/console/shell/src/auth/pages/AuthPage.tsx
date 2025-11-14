// src/auth/pages/AuthPage.tsx
import * as React from "react";
import { Button } from "../../shared/ui/button";
import "../styles/AuthPage.css";
import { useAuthActions } from "../application/useAuthActions";

export default function AuthPage() {
  const { signIn, submitting, error, setError } = useAuthActions();
  const [email, setEmail] = React.useState("");
  const [password, setPassword] = React.useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    await signIn(email, password);
  };

  return (
    <div className="auth-page">
      <div className="auth-card">
        <h1 className="auth-title">Narratives Console</h1>
        <p className="auth-description">
          ブランド・運営向けコンソールにアクセスするには、ログインが必要です。
        </p>

        <form className="auth-form" onSubmit={handleSubmit}>
          <label className="auth-label">
            メールアドレス
            <input
              type="email"
              className="auth-input"
              value={email}
              onChange={(e) => {
                setEmail(e.target.value);
                if (error) setError(null);
              }}
              required
            />
          </label>

          <label className="auth-label">
            パスワード
            <input
              type="password"
              className="auth-input"
              value={password}
              onChange={(e) => {
                setPassword(e.target.value);
                if (error) setError(null);
              }}
              required
            />
          </label>

          {error && <p className="auth-error">{error}</p>}

          <div className="auth-actions">
            <Button
              type="submit"
              variant="solid"
              size="lg"
              disabled={submitting}
            >
              {submitting ? "ログイン中..." : "Console に入る"}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
