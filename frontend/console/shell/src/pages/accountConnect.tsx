// frontend/console/shell/src/pages/accountConnect.tsx

import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

import { auth } from "../auth/infrastructure/config/firebaseClient";
import { accountRepositoryHTTP } from "../features/account/infrastructure/http/accountRepositoryHTTP";
import { Button } from "../shared/ui/button";
import { Card, CardContent } from "../shared/ui/card";

import "../styles/account.css";

function getErrorMessage(error: unknown): string {
  if (error instanceof Error && error.message) {
    return error.message;
  }

  return "Stripe口座との接続に失敗しました。";
}

export default function AccountConnectPage() {
  const navigate = useNavigate();

  const [contactEmail, setContactEmail] = useState(auth.currentUser?.email ?? "");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [completed, setCompleted] = useState(false);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    setCompleted(params.get("completed") === "1");
  }, []);

  const handleAccountManagement = () => {
    navigate("/account");
  };

  const handleConnect = async () => {
    if (submitting) {
      return;
    }

    const normalizedEmail = contactEmail.trim();

    if (!normalizedEmail) {
      setError("Stripe口座に使用するメールアドレスを入力してください。");
      return;
    }

    try {
      setSubmitting(true);
      setError(null);

      const origin = window.location.origin;

      const response = await accountRepositoryHTTP.connect({
        contactEmail: normalizedEmail,
        country: "JP",
        returnUrl: `${origin}/account/connect?completed=1`,
        refreshUrl: `${origin}/account/connect`,
      });

      const onboardingUrl = response?.onboardingUrl?.trim();

      if (!onboardingUrl) {
        throw new Error("Stripe onboarding URLを取得できませんでした。");
      }

      window.location.assign(onboardingUrl);
    } catch (caughtError: unknown) {
      setError(getErrorMessage(caughtError));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="account-connect-page">
      <div className="account-connect-container">
        <Card>
          <CardContent>
            <div className="account-connect-content">
              <h1 className="account-connect-title">口座接続</h1>

              <p className="account-connect-description">
                売上受取口座としてStripeを接続します。
                接続した口座は、ブランド登録・編集時に売上受取口座として選択できます。
              </p>

              {completed && (
                <div className="account-connect-completed">
                  Stripeの口座登録画面から戻りました。
                  接続状態はStripe側の審査・入力状況によって反映まで時間がかかる場合があります。
                </div>
              )}

              {error && (
                <div
                  role="alert"
                  className="account-connect-error"
                >
                  {error}
                </div>
              )}

              <div className="account-connect-field">
                <label
                  htmlFor="account-connect-email"
                  className="account-connect-label"
                >
                  Stripe連絡先メールアドレス
                </label>

                <input
                  id="account-connect-email"
                  type="email"
                  className="account-connect-input"
                  value={contactEmail}
                  onChange={(event) => {
                    setContactEmail(event.target.value);

                    if (error) {
                      setError(null);
                    }
                  }}
                  disabled={submitting}
                  required
                />
              </div>

              <div className="account-connect-summary">
                <div className="account-connect-summary-title">
                  接続内容
                </div>

                <div>
                  Stripe連絡先：
                  {contactEmail || "未入力"}
                </div>

                <div>国・地域：日本</div>
              </div>

              <div className="account-connect-actions">
                <Button
                  type="button"
                  variant="solid"
                  size="lg"
                  disabled={
                    submitting ||
                    !contactEmail.trim()
                  }
                  onClick={() => {
                    void handleConnect();
                  }}
                >
                  {submitting
                    ? "Stripeへ接続中..."
                    : "Stripeと接続する"}
                </Button>

                <Button
                  type="button"
                  variant="outline"
                  size="lg"
                  disabled={submitting}
                  onClick={handleAccountManagement}
                >
                  口座管理へ
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}