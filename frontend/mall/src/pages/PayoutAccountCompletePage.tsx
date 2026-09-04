// frontend/amol/src/pages/PayoutAccountCompletePage.tsx

import { useCallback, useEffect } from "react";
import { CheckCircle2 } from "lucide-react";
import { useLocation, useNavigate } from "react-router-dom";

import "../styles/page-layout.css";
import "../styles/settings-page.css";
import "../styles/payout-account-complete-page.css";

import Layout from "../components/layout/Layout";
import FooterNav from "../components/layout/FooterNav";
import { useContactViewport } from "../features/contact/hooks/useContactViewport";

type PayoutAccountCompleteLocationState = {
  registrationCompleted?: boolean;
};

export default function PayoutAccountCompletePage() {
  const navigate = useNavigate();
  const location = useLocation();
  const { isDesktop } = useContactViewport();

  const state = location.state as PayoutAccountCompleteLocationState | null;
  const registrationCompleted = state?.registrationCompleted === true;

  useEffect(() => {
    if (!registrationCompleted) {
      navigate("/settings/payout-account", { replace: true });
    }
  }, [navigate, registrationCompleted]);

  const handleBackToPayoutAccount = useCallback(() => {
    navigate("/settings/payout-account", { replace: true });
  }, [navigate]);

  if (!registrationCompleted) {
    return null;
  }

  return (
    <Layout
      title="口座登録完了"
      titleClickable={false}
      showBackButton={false}
      mode="default"
      hideHamburgerMenu
      hideSettingsButton
      actionButtonLabel={isDesktop ? "売上受取口座へ戻る" : undefined}
      onActionButtonClick={isDesktop ? handleBackToPayoutAccount : undefined}
    >
      <section className="page-section content-page-section settings-page payout-account-complete-page">
        <div className="payout-account-complete-page__hero">
          <div
            className="payout-account-complete-page__icon"
            aria-hidden="true"
          >
            <CheckCircle2 size={52} strokeWidth={1.8} />
          </div>

          <h1 className="payout-account-complete-page__title">
            売上受取口座を登録しました
          </h1>

          <p className="payout-account-complete-page__description">
            口座情報の登録が完了しました。
          </p>
        </div>

        <div className="payout-account-complete-page__notice">
          <p className="payout-account-complete-page__notice-title">
            売上の受取について
          </p>

          <p className="payout-account-complete-page__notice-text">
            登録した銀行口座は、再販売で発生した売上の受取先として使用されます。登録内容は売上受取口座画面から確認・変更できます。
          </p>
        </div>

        <p className="payout-account-complete-page__security-note">
          銀行口座番号は暗号化して保存し、AMOLの画面では末尾4桁のみ表示します。
        </p>
      </section>

      {!isDesktop ? (
        <FooterNav
          variant="action"
          buttonLabel="売上受取口座へ戻る"
          onButtonClick={handleBackToPayoutAccount}
        />
      ) : null}
    </Layout>
  );
}