// frontend/console/shell/src/auth/pages/AuthPage.tsx
import * as React from "react";
import { Button } from "../../shared/ui/button";
import "../styles/AuthPage.css";
import { useAuthActions } from "../application/useAuthActions";

export default function AuthPage() {
  const { signUp, signIn, submitting, error, setError } = useAuthActions();

  const [mode, setMode] = React.useState<"signup" | "signin">("signin");

  const [email, setEmail] = React.useState("");
  const [password, setPassword] = React.useState("");
  const [confirmPassword, setConfirmPassword] = React.useState("");

  // 姓名＋かな
  const [lastName, setLastName] = React.useState("");
  const [firstName, setFirstName] = React.useState("");
  const [lastNameKana, setLastNameKana] = React.useState("");
  const [firstNameKana, setFirstNameKana] = React.useState("");

  // 会社名・団体名（signup 時のみ使用 / 任意入力）
  const [companyName, setCompanyName] = React.useState("");

  const resetForm = () => {
    setEmail("");
    setPassword("");
    setConfirmPassword("");
    setLastName("");
    setFirstName("");
    setLastNameKana("");
    setFirstNameKana("");
    setCompanyName("");
    setError(null);
  };

  const switchMode = (next: "signup" | "signin") => {
    setMode(next);
    resetForm();
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (mode === "signup") {
      if (password !== confirmPassword) {
        setError("パスワードが一致していません。");
        return;
      }

      await signUp(email, password, {
        lastName,
        firstName,
        lastNameKana,
        firstNameKana,
        companyName, // ← ここで会社名を委譲
      });
      return;
    }

    // ログイン(Sign In)
    await signIn(email, password);
  };

  return (
    <div className="auth-page">
      <div className="auth-card">
        <h1 className="auth-title">
          {mode === "signup" ? "管理アカウントの新規登録" : "ログイン"}
        </h1>

        <form className="auth-form" onSubmit={handleSubmit}>
          {/* 新規登録モード：姓名 + かな（2カラム） */}
          {mode === "signup" && (
            <>
              <div className="auth-row">
                <label className="auth-label auth-label-inline">
                  姓
                  <input
                    type="text"
                    className="auth-input"
                    value={lastName}
                    onChange={(e) => setLastName(e.target.value)}
                    required
                  />
                </label>

                <label className="auth-label auth-label-inline">
                  姓（かな）
                  <input
                    type="text"
                    className="auth-input"
                    value={lastNameKana}
                    onChange={(e) => setLastNameKana(e.target.value)}
                    required
                  />
                </label>
              </div>

              <div className="auth-row">
                <label className="auth-label auth-label-inline">
                  名
                  <input
                    type="text"
                    className="auth-input"
                    value={firstName}
                    onChange={(e) => setFirstName(e.target.value)}
                    required
                  />
                </label>

                <label className="auth-label auth-label-inline">
                  名（かな）
                  <input
                    type="text"
                    className="auth-input"
                    value={firstNameKana}
                    onChange={(e) => setFirstNameKana(e.target.value)}
                    required
                  />
                </label>
              </div>
            </>
          )}

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

          {/* 新規登録時のみ：会社名・団体名（任意） */}
          {mode === "signup" && (
            <label className="auth-label">
              会社名・団体名（任意）
              <input
                type="text"
                className="auth-input"
                value={companyName}
                onChange={(e) => setCompanyName(e.target.value)}
                placeholder="例）LUMINA Fashion"
              />
            </label>
          )}

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

          {mode === "signup" && (
            <label className="auth-label">
              パスワード（確認用）
              <input
                type="password"
                className="auth-input"
                value={confirmPassword}
                onChange={(e) => {
                  setConfirmPassword(e.target.value);
                  if (error) setError(null);
                }}
                required
              />
            </label>
          )}

          {error && <p className="auth-error">{error}</p>}

          <div className="auth-actions" style={{ justifyContent: "center" }}>
            <Button
              type="submit"
              variant="solid"
              size="lg"
              disabled={submitting}
            >
              {submitting
                ? mode === "signup"
                  ? "登録中..."
                  : "ログイン中..."
                : mode === "signup"
                ? "管理アカウントを登録する"
                : "ログインする"}
            </Button>
          </div>
        </form>

        {/* モード切り替え */}
        <div className="auth-switch">
          {mode === "signup" ? (
            <p>
              すでにアカウントをお持ちの方{" "}
              <button onClick={() => switchMode("signin")}>ログインする</button>
            </p>
          ) : (
            <p>
              アカウントをお持ちでない方{" "}
              <button onClick={() => switchMode("signup")}>新規登録する</button>
            </p>
          )}
        </div>
      </div>
    </div>
  );
}
