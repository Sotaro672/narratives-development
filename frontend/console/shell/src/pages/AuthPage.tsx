// frontend/console/shell/src/pages/AuthPage.tsx

import { useEffect } from "react";

import { useAuthPage } from "../auth/presentation/hook/useAuthPage";
import { Button } from "../shared/ui/button";
import { ErrorMessage } from "../shared/ui/error";

import "../styles/auth.css";

export default function AuthPage() {
  const {
    mode,
    switchMode,

    forgotPasswordMode,
    setForgotPasswordMode,

    email,
    setEmail,
    password,
    setPassword,
    confirmPassword,
    setConfirmPassword,

    lastName,
    setLastName,
    firstName,
    setFirstName,
    lastNameKana,
    setLastNameKana,
    firstNameKana,
    setFirstNameKana,

    companyName,
    setCompanyName,

    submitting,
    error,
    setError,

    signupCompleted,
    resetSignupFlow,

    handleFormSubmit,
  } = useAuthPage();

  useEffect(() => {
    if (signupCompleted && mode === "signup") {
      window.alert(
        "ご登録のメールアドレス宛に確認メールを送信しました。\nメール内のリンクをクリックして認証を完了してください。",
      );
      resetSignupFlow();
    }
  }, [signupCompleted, mode, resetSignupFlow]);

  return (
    <div className="auth-page">
      <div className="auth-card">
        <h1 className="auth-title">
          {mode === "signup"
            ? "管理アカウントの新規登録"
            : forgotPasswordMode
              ? "パスワード再設定"
              : "ログイン"}
        </h1>

        <form className="auth-form" onSubmit={handleFormSubmit}>
          {mode === "signup" && (
            <>
              <div className="auth-row">
                <label className="auth-label auth-label-inline">
                  姓（漢字）
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
                    placeholder="姓（せい）"
                    required
                  />
                </label>
              </div>

              <div className="auth-row">
                <label className="auth-label auth-label-inline">
                  名（漢字）
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
                    placeholder="名（めい）"
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

          {mode === "signup" && (
            <label className="auth-label">
              会社名・団体名
              <input
                type="text"
                className="auth-input"
                value={companyName}
                onChange={(e) => setCompanyName(e.target.value)}
                placeholder="会社名・団体名を入力してください"
              />
            </label>
          )}

          {!(mode === "signin" && forgotPasswordMode) && (
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
          )}

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

          {mode === "signin" && !forgotPasswordMode && (
            <div className="auth-forgot">
              <button
                type="button"
                className="auth-forgot-link"
                onClick={() => {
                  setError(null);
                  setForgotPasswordMode(true);
                }}
              >
                パスワードをお忘れの方はこちら
              </button>
            </div>
          )}

          {mode === "signin" && forgotPasswordMode && (
            <div className="auth-forgot">
              <button
                type="button"
                className="auth-forgot-link"
                onClick={() => {
                  setError(null);
                  setForgotPasswordMode(false);
                }}
              >
                ログイン画面に戻る
              </button>
            </div>
          )}

          {error && <ErrorMessage size="xs">{error}</ErrorMessage>}

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
                  : forgotPasswordMode
                    ? "送信中..."
                    : "ログイン中..."
                : mode === "signup"
                  ? "管理アカウントを登録する"
                  : forgotPasswordMode
                    ? "パスワード再設定メールを送信"
                    : "ログインする"}
            </Button>
          </div>
        </form>

        <div className="auth-switch">
          {mode === "signup" ? (
            <p>
              すでにアカウントをお持ちの方{" "}
              <button
                onClick={() => {
                  resetSignupFlow();
                  switchMode("signin");
                }}
              >
                ログインする
              </button>
            </p>
          ) : (
            <p>
              アカウントをお持ちでない方{" "}
              <button
                onClick={() => {
                  resetSignupFlow();
                  switchMode("signup");
                }}
              >
                新規登録する
              </button>
            </p>
          )}
        </div>
      </div>
    </div>
  );
}