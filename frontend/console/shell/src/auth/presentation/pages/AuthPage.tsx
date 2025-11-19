// frontend/console/shell/src/auth/presentation/pages/AuthPage.tsx
import { useEffect } from "react";
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

    signupCompleted,
    resetSignupFlow,

    handleFormSubmit,
  } = useAuthPage();

  // ★ signup 完了時にダイアログを表示
  useEffect(() => {
    if (signupCompleted && mode === "signup") {
      window.alert(
        "ご登録のメールアドレス宛に確認メールを送信しました。\nメール内のリンクをクリックして認証を完了してください。"
      );
      // 再送信やログイン切り替え時に余計な発火を防ぐ
      resetSignupFlow();
    }
  }, [signupCompleted, mode, resetSignupFlow]);

  return (
    <div className="auth-page">
      <div className="auth-card">
        <h1 className="auth-title">
          {mode === "signup" ? "管理アカウントの新規登録" : "ログイン"}
        </h1>

        <form className="auth-form" onSubmit={handleFormSubmit}>
          {/* ▼ signup：姓名 + かな */}
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

          {/* ▼ メールアドレス */}
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

          {/* ▼ signup：会社名 */}
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

          {/* ▼ パスワード */}
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

          {/* ▼ signup：パスワード確認 */}
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

          {/* ▼ アクションボタン */}
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

        {/* ▼ モード切り替え */}
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
