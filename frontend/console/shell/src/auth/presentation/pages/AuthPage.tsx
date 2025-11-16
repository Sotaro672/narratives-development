import * as React from "react";
import { Button } from "../../../shared/ui/button";
import "../styles/auth.css";
import { useAuthPage } from "../hook/useAuthPage";

export default function AuthPage() {
  const {
    mode,
    switchMode,
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
    handleSubmit,
  } = useAuthPage();

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
