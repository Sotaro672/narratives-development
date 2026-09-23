// frontend/console/shell/src/pages/AuthPage.tsx

import { useEffect } from "react";

import { useAuthPage } from "../auth/presentation/hook/useAuthPage";
import { Button } from "../shared/ui/button";
import {
  Card,
  CardContent,
  CardField,
} from "../shared/ui/card";
import { ErrorMessage } from "../shared/ui/error";
import { Input } from "../shared/ui/input";
import { Label } from "../shared/ui/label";
import Stack from "../shared/ui/stack";
import Text from "../shared/ui/text";

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
      <Card className="auth-card" elevated largeRadius>
        <CardContent size="large">
          <h1 className="auth-title">
            {mode === "signup"
              ? "管理アカウントの新規登録"
              : forgotPasswordMode
                ? "パスワード再設定"
                : "ログイン"}
          </h1>

          <form onSubmit={handleFormSubmit}>
            <Stack gap="md">
              {mode === "signup" && (
                <>
                  <div className="auth-row">
                    <CardField className="auth-field--inline">
                      <Label htmlFor="lastName">姓（漢字）</Label>
                      <Input
                        id="lastName"
                        type="text"
                        value={lastName}
                        onChange={(event) => setLastName(event.target.value)}
                        required
                      />
                    </CardField>

                    <CardField className="auth-field--inline">
                      <Label htmlFor="lastNameKana">姓（かな）</Label>
                      <Input
                        id="lastNameKana"
                        type="text"
                        value={lastNameKana}
                        onChange={(event) => setLastNameKana(event.target.value)}
                        placeholder="姓（せい）"
                        required
                      />
                    </CardField>
                  </div>

                  <div className="auth-row">
                    <CardField className="auth-field--inline">
                      <Label htmlFor="firstName">名（漢字）</Label>
                      <Input
                        id="firstName"
                        type="text"
                        value={firstName}
                        onChange={(event) => setFirstName(event.target.value)}
                        required
                      />
                    </CardField>

                    <CardField className="auth-field--inline">
                      <Label htmlFor="firstNameKana">名（かな）</Label>
                      <Input
                        id="firstNameKana"
                        type="text"
                        value={firstNameKana}
                        onChange={(event) => setFirstNameKana(event.target.value)}
                        placeholder="名（めい）"
                        required
                      />
                    </CardField>
                  </div>
                </>
              )}

              <CardField>
                <Label htmlFor="email">メールアドレス</Label>
                <Input
                  id="email"
                  type="email"
                  value={email}
                  onChange={(event) => {
                    setEmail(event.target.value);
                    if (error) {
                      setError(null);
                    }
                  }}
                  required
                />
              </CardField>

              {mode === "signup" && (
                <CardField>
                  <Label htmlFor="companyName">会社名・団体名</Label>
                  <Input
                    id="companyName"
                    type="text"
                    value={companyName}
                    onChange={(event) => setCompanyName(event.target.value)}
                    placeholder="会社名・団体名を入力してください"
                  />
                </CardField>
              )}

              {!(mode === "signin" && forgotPasswordMode) && (
                <CardField>
                  <Label htmlFor="password">パスワード</Label>
                  <Input
                    id="password"
                    type="password"
                    value={password}
                    onChange={(event) => {
                      setPassword(event.target.value);
                      if (error) {
                        setError(null);
                      }
                    }}
                    required
                  />
                </CardField>
              )}

              {mode === "signup" && (
                <CardField>
                  <Label htmlFor="confirmPassword">パスワード（確認用）</Label>
                  <Input
                    id="confirmPassword"
                    type="password"
                    value={confirmPassword}
                    onChange={(event) => {
                      setConfirmPassword(event.target.value);
                      if (error) {
                        setError(null);
                      }
                    }}
                    required
                  />
                </CardField>
              )}

              {mode === "signin" && !forgotPasswordMode && (
                <div className="auth-forgot">
                  <Button
                    type="button"
                    variant="link"
                    onClick={() => {
                      setError(null);
                      setForgotPasswordMode(true);
                    }}
                  >
                    パスワードをお忘れの方はこちら
                  </Button>
                </div>
              )}

              {mode === "signin" && forgotPasswordMode && (
                <div className="auth-forgot">
                  <Button
                    type="button"
                    variant="link"
                    onClick={() => {
                      setError(null);
                      setForgotPasswordMode(false);
                    }}
                  >
                    ログイン画面に戻る
                  </Button>
                </div>
              )}

              {error ? (
                <ErrorMessage size="xs">
                  {error}
                </ErrorMessage>
              ) : null}

              <div className="auth-actions">
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
            </Stack>
          </form>

          <div className="auth-switch">
            {mode === "signup" ? (
              <Text as="p" size="sm" tone="muted">
                すでにアカウントをお持ちの方{" "}
                <Button
                  type="button"
                  variant="link"
                  onClick={() => {
                    resetSignupFlow();
                    switchMode("signin");
                  }}
                >
                  ログインする
                </Button>
              </Text>
            ) : (
              <Text as="p" size="sm" tone="muted">
                アカウントをお持ちでない方{" "}
                <Button
                  type="button"
                  variant="link"
                  onClick={() => {
                    resetSignupFlow();
                    switchMode("signup");
                  }}
                >
                  新規登録する
                </Button>
              </Text>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}