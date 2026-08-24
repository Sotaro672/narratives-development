// frontend/console/shell/src/pages/accountConnect.tsx

import { useAccountConnect } from "../features/account/presentation/hook/useAccountConnect";
import { Button } from "../shared/ui/button";
import { Card, CardContent } from "../shared/ui/card";

import "../styles/account.css";

export default function AccountConnectPage() {
  const {
    contactEmail,
    submitting,
    error,
    completed,
    canConnect,

    handleContactEmailChange,
    handleConnect,
    handleAccountManagement,
  } = useAccountConnect();

  return (
    <div className="account-connect-page">
      <div className="account-connect-container">
        <Card>
          <CardContent>
            <div className="account-connect-content">
              <h1 className="account-connect-title">
                口座接続
              </h1>

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
                    handleContactEmailChange(
                      event.target.value,
                    );
                  }}
                  disabled={submitting}
                  required
                />
              </div>

              <div className="account-connect-summary">
                <div>
                  Stripe連絡先：
                  {contactEmail || "未入力"}
                </div>
              </div>

              <div className="account-connect-actions">
                <Button
                  type="button"
                  variant="solid"
                  size="lg"
                  disabled={!canConnect}
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
                  onClick={
                    handleAccountManagement
                  }
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