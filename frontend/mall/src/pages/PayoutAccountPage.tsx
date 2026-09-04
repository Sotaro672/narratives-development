// frontend/amol/src/pages/PayoutAccountPage.tsx

import "../styles/page-layout.css";
import "../styles/settings-page.css";
import "../styles/payoutAccount-page.css";

import Layout from "../components/layout/Layout";
import PayoutAccountStatusCard from "../features/payout/components/PayoutAccountStatusCard";
import { usePayoutAccountPage } from "../features/payout/hooks/usePayoutAccountPage";

export default function PayoutAccountPage() {
  const {
    isLoading,
    errorMessage,
    statusLabel,
    actionLabel,
    bankName,
    bankLast4,
    handleOpenRegistration,
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
            登録した口座は、再販売の売上を受け取る際に使用されます。
          </p>

          <PayoutAccountStatusCard
            statusLabel={statusLabel}
            bankName={bankName}
            bankLast4={bankLast4}
          />

          {errorMessage ? (
            <p className="payout-account-page__error">
              {errorMessage}
            </p>
          ) : null}

          <button
            type="button"
            onClick={handleOpenRegistration}
            disabled={isLoading}
            className="payout-account-page__action-button"
          >
            {actionLabel}
          </button>

          <p className="payout-account-page__note">
            登録後の画面では、口座番号は末尾4桁のみ表示します。
          </p>
        </div>
      </section>
    </Layout>
  );
}