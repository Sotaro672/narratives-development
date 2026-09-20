// frontend/mall/src/pages/SignUpPage.tsx

import { useEffect, useState } from "react";

import Layout from "../components/layout/Layout";
import Alert from "../components/ui/Alert";
import Button from "../components/ui/Button";
import Card from "../components/ui/Card";
import Input from "../components/ui/Input";
import TextState from "../components/ui/TextState";
import { useSignUpPage } from "../features/auth/hooks/useSignUpPage";

import "../styles/page-layout.css";
import "../styles/form.css";
import "../styles/signUp-page.css";

export default function SignUpPage() {
  const vm = useSignUpPage();

  const [termsText, setTermsText] = useState("");
  const [termsLoading, setTermsLoading] = useState(true);
  const [termsError, setTermsError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    const loadTerms = async () => {
      try {
        setTermsLoading(true);
        setTermsError(null);

        const response = await fetch("/assets/terms-for-user.txt");
        const contentType = response.headers.get("content-type") ?? "";

        if (
          !response.ok ||
          !contentType.toLowerCase().startsWith("text/plain")
        ) {
          throw new Error("利用規約を読み込めませんでした。");
        }

        const text = await response.text();

        if (!cancelled) {
          setTermsText(text);
        }
      } catch {
        if (!cancelled) {
          setTermsText("");
          setTermsError("利用規約を読み込めませんでした。");
        }
      } finally {
        if (!cancelled) {
          setTermsLoading(false);
        }
      }
    };

    void loadTerms();

    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <Layout title="AMOL">
      <section className="page-section signup-page-section">
        <p className="page-description">
          {vm.topMessage}
        </p>

        <div className="form-block signup-form-block">
          <Input
            label="メールアドレス"
            type="email"
            placeholder="example@email.com"
            value={vm.email}
            onChange={(event) => {
              vm.setEmail(event.target.value);
              vm.clearError();
            }}
            disabled={vm.loading}
            autoComplete="email"
            fullWidth
          />

          <Input
            label="パスワード"
            type="password"
            placeholder="パスワードを入力"
            value={vm.password}
            onChange={(event) => {
              vm.setPassword(event.target.value);
              vm.clearError();
            }}
            disabled={vm.loading}
            autoComplete="new-password"
            fullWidth
          />

          <Input
            label="パスワード確認用"
            type="password"
            placeholder="もう一度パスワードを入力"
            value={vm.passwordConfirmation}
            onChange={(event) => {
              vm.setPasswordConfirmation(event.target.value);
              vm.clearError();
            }}
            disabled={vm.loading}
            autoComplete="new-password"
            fullWidth
          />

          <div className="terms-block">
            <p className="terms-title">
              利用規約
            </p>

            <Card
              padding="sm"
              className="terms-scroll-box"
            >
              {termsLoading ? (
                <TextState variant="loading">
                  利用規約を読み込み中...
                </TextState>
              ) : termsError ? (
                <TextState variant="error">
                  {termsError}
                </TextState>
              ) : (
                <pre className="terms-text">
                  {termsText}
                </pre>
              )}
            </Card>
          </div>

          <label className="form-checkbox-row">
            <input
              type="checkbox"
              checked={vm.agree}
              disabled={
                vm.loading ||
                termsLoading ||
                Boolean(termsError)
              }
              onChange={(event) => {
                vm.setAgree(event.target.checked);
                vm.clearError();
              }}
            />

            <span>
              利用規約に同意します
            </span>
          </label>

          {vm.error ? (
            <Alert
              variant="error"
              className="signup-page__error"
            >
              {vm.error}
            </Alert>
          ) : null}
        </div>

        <div className="page-actions signup-page-actions">
          <Button
            variant="primary"
            fullWidth
            onClick={vm.handleSignUp}
            disabled={
              !vm.canSubmit ||
              termsLoading ||
              Boolean(termsError)
            }
          >
            {vm.loading ? "送信中..." : "認証メールを送信"}
          </Button>
        </div>
      </section>
    </Layout>
  );
}