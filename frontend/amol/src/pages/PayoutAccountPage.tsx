// frontend/amol/src/pages/PayoutAccountPage.tsx

import "../styles/page-layout.css";
import "../styles/settings-page.css";
import "../styles/payoutAccount-page.css";

import Layout from "../components/layout/Layout";
import PayoutAccountNotice from "../features/payout/components/PayoutAccountNotice";
import PayoutAccountStatusCard from "../features/payout/components/PayoutAccountStatusCard";
import { usePayoutAccountPage } from "../features/payout/hooks/usePayoutAccountPage";

export default function PayoutAccountPage() {
  const {
    payoutAccount,
    isLoading,
    isOpeningStripe,
    errorMessage,
    stripeReturnState,
    statusLabel,
    actionLabel,
    bankName,
    bankLast4,
    handleOpenStripe,
  } = usePayoutAccountPage();

  return (
    <Layout
      title="売上受取口座"
      titleClickable={false}
      showBackButton
      mode="signin"
      backTo="/lists"
    >
      <section className="page-section settings-page payout-account-page">
        <div className="payout-account-page__content">
          <p className="content-page-description payout-account-page__description">
            再販売で発生した売上の受取口座を登録します。
            口座情報の登録・変更はStripeの安全な画面で行います。
          </p>

          <PayoutAccountStatusCard
            statusLabel={statusLabel}
            bankName={bankName}
            bankLast4={bankLast4}
          />

          {payoutAccount && !payoutAccount.payoutsEnabled ? (
            <PayoutAccountNotice>
              Stripeで追加情報の登録または確認が必要です。
              登録を完了すると、再販売の売上を受け取れるようになります。
            </PayoutAccountNotice>
          ) : null}

          {stripeReturnState === "refresh" ? (
            <PayoutAccountNotice>
              Stripeの登録リンクの有効期限が切れました。
              下のボタンからもう一度お進みください。
            </PayoutAccountNotice>
          ) : null}

          {errorMessage ? (
            <p className="payout-account-page__error">
              {errorMessage}
            </p>
          ) : null}

          <button
            type="button"
            onClick={handleOpenStripe}
            disabled={isLoading || isOpeningStripe}
            className="payout-account-page__action-button"
          >
            {actionLabel}
          </button>

          <p className="payout-account-page__note">
            銀行口座番号などの口座情報はStripe上で管理されます。
            AMOLでは口座番号の全桁を保存しません。
          </p>
        </div>
      </section>
    </Layout>
  );
}