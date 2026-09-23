// frontend/console/shell/src/pages/accountCreate.tsx

import { useAccountConnect } from "../features/account/presentation/hook/useAccountCreate";
import PageStyle from "../layout/PageStyle/PageStyle";
import { Card, CardContent } from "../shared/ui/card";
import { ErrorMessage } from "../shared/ui/error";
import { Label } from "../shared/ui/label";

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
                  入力内容を審査させていただきます。審査には時間がかかる場合があります。
                </p>

                {completed && (
                  <div className="account-connect-completed">
                    登録申請を受理しました。
                    審査内容の返答には時間がかかる場合があります。
                  </div>
                )}

                {error && (
                  <ErrorMessage
                    variant="panel"
                    className="account-connect-error"
                  >
                    {error}
                  </ErrorMessage>
                )}

                <div className="account-connect-field">
                  <Label
                    htmlFor="account-connect-bank-name"
                    className="account-connect-label"
                  >
                    銀行名
                  </Label>

                  <input
                    id="account-connect-bank-name"
                    type="text"
                    className="account-connect-input"
                    value={bankName}
                    onChange={(event) => {
                      handleBankNameChange(event.target.value);
                    }}
                    disabled={submitting}
                    required
                  />
                </div>

                <div className="account-connect-field">
                  <Label
                    htmlFor="account-connect-branch-name"
                    className="account-connect-label"
                  >
                    支店名
                  </Label>

                  <input
                    id="account-connect-branch-name"
                    type="text"
                    className="account-connect-input"
                    value={branchName}
                    onChange={(event) => {
                      handleBranchNameChange(event.target.value);
                    }}
                    disabled={submitting}
                    required
                  />
                </div>

                <div className="account-connect-field">
                  <Label
                    htmlFor="account-connect-account-number"
                    className="account-connect-label"
                  >
                    口座番号
                  </Label>

                  <input
                    id="account-connect-account-number"
                    type="text"
                    inputMode="numeric"
                    className="account-connect-input"
                    value={accountNumber}
                    onChange={(event) => {
                      handleAccountNumberChange(event.target.value);
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