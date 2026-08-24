// frontend/console/shell/src/pages/accountConnect.tsx

import { useAccountConnect } from "../features/account/presentation/hook/useAccountConnect";
import PageStyle from "../layout/PageStyle/PageStyle";
import { Card, CardContent } from "../shared/ui/card";

import "../styles/account.css";

export default function AccountConnectPage() {
  const {
    bankName,
    branchName,
    accountNumber,

    submitting,
    error,
    completed,
    canConnect,

    handleBankNameChange,
    handleBranchNameChange,
    handleAccountNumberChange,

    handleBack,
    handleConnect,
  } = useAccountConnect();

  return (
    <PageStyle
      title="口座接続"
      onBack={handleBack}
      onConnect={handleConnect}
      isConnecting={submitting}
      connectLabel="口座登録"
      connectBusyLabel="口座登録中..."
      connectDisabled={!canConnect}
    >
      <div className="account-connect-page">
        <div className="account-connect-container">
          <Card>
            <CardContent>
              <div className="account-connect-content">
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
                    htmlFor="account-connect-bank-name"
                    className="account-connect-label"
                  >
                    銀行名
                  </label>

                  <input
                    id="account-connect-bank-name"
                    type="text"
                    className="account-connect-input"
                    value={bankName}
                    onChange={(event) => {
                      handleBankNameChange(
                        event.target.value,
                      );
                    }}
                    disabled={submitting}
                    required
                  />
                </div>

                <div className="account-connect-field">
                  <label
                    htmlFor="account-connect-branch-name"
                    className="account-connect-label"
                  >
                    支店名
                  </label>

                  <input
                    id="account-connect-branch-name"
                    type="text"
                    className="account-connect-input"
                    value={branchName}
                    onChange={(event) => {
                      handleBranchNameChange(
                        event.target.value,
                      );
                    }}
                    disabled={submitting}
                    required
                  />
                </div>

                <div className="account-connect-field">
                  <label
                    htmlFor="account-connect-account-number"
                    className="account-connect-label"
                  >
                    口座番号
                  </label>

                  <input
                    id="account-connect-account-number"
                    type="text"
                    inputMode="numeric"
                    className="account-connect-input"
                    value={accountNumber}
                    onChange={(event) => {
                      handleAccountNumberChange(
                        event.target.value,
                      );
                    }}
                    disabled={submitting}
                    maxLength={8}
                    required
                  />
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </PageStyle>
  );
}