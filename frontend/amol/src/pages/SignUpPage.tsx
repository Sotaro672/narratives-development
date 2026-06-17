//frontend\amol\src\pages\SignUpPage.tsx
import { useEffect, useState } from "react";

import "../styles/page-layout.css";
import "../styles/form.css";
import "../styles/signUp-page.css";

import Layout from "../components/layout/Layout";
import Button from "../components/ui/Button";
import Input from "../components/ui/Input";
import { useSignUpPage } from "../features/auth/hooks/useSignUpPage";

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

        const response = await fetch("/assets/terms.txt");

        if (!response.ok) {
          throw new Error("利用規約を読み込めませんでした。");
        }

        const text = await response.text();

        if (!cancelled) {
          setTermsText(text);
        }
      } catch {
        if (!cancelled) {
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
        <p className="page-description">{vm.topMessage}</p>

        <div className="form-block signup-form-block">
          <Input
            label="メールアドレス"
            type="email"
            placeholder="example@email.com"
            value={vm.email}
            onChange={(e) => {
              vm.setEmail(e.target.value);
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
            onChange={(e) => {
              vm.setPassword(e.target.value);
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
            onChange={(e) => {
              vm.setPasswordConfirmation(e.target.value);
              vm.clearError();
            }}
            disabled={vm.loading}
            autoComplete="new-password"
            fullWidth
          />

          <div className="terms-block">
            <p className="terms-title">利用規約</p>

            <div className="terms-scroll-box">
              {termsLoading ? (
                <p className="terms-status-text">利用規約を読み込み中...</p>
              ) : termsError ? (
                <p className="terms-error-text">{termsError}</p>
              ) : (
                <pre className="terms-text">{termsText}</pre>
              )}
            </div>
          </div>

          <label className="form-checkbox-row">
            <input
              type="checkbox"
              checked={vm.agree}
              disabled={vm.loading || termsLoading || Boolean(termsError)}
              onChange={(e) => {
                vm.setAgree(e.target.checked);
                vm.clearError();
              }}
            />
            <span>利用規約に同意します</span>
          </label>

          {vm.error ? <p className="form-error-text">{vm.error}</p> : null}
        </div>

        <div className="page-actions signup-page-actions">
          <Button
            variant="primary"
            onClick={vm.handleSignUp}
            disabled={!vm.canSubmit || termsLoading || Boolean(termsError)}
          >
            {vm.loading ? "送信中..." : "認証メールを送信"}
          </Button>
        </div>
      </section>
    </Layout>
  );
}