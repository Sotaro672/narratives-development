// frontend/amol/src/pages/PayoutAccountCompletePage.tsx

import { useEffect, useRef } from "react";
import { CheckCircle2 } from "lucide-react";
import { useNavigate } from "react-router-dom";

import "../styles/page-layout.css";
import "../styles/settings-page.css";
import "../styles/payout-account-complete-page.css";

import Layout from "../components/layout/Layout";
import FooterNav from "../components/layout/FooterNav";
import { useContactViewport } from "../features/contact/hooks/useContactViewport";
import { usePayoutAccountRegistration } from "../features/payout/context/PayoutAccountRegistrationProvider";

export default function PayoutAccountCompletePage() {
  const navigate = useNavigate();
  const { isDesktop } = useContactViewport();
  const { isComplete, resetDraft } = usePayoutAccountRegistration();

  const registrationCompleted = useRef(isComplete).current;

  useEffect(() => {
    if (!registrationCompleted) {
      navigate("/settings/payout-account", { replace: true });
      return;
    }

    resetDraft();
  }, [navigate, registrationCompleted, resetDraft]);

  const handleBackToPayoutAccount = () => {
    navigate("/settings/payout-account", { replace: true });
  };

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
      actionButtonLabel={
        isDesktop ? "売上受取口座へ戻る" : undefined
      }
      onActionButtonClick={
        isDesktop ? handleBackToPayoutAccount : undefined
      }
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
            口座情報の登録を受け付けました。
          </p>
        </div>

        <div className="payout-account-complete-page__notice">
          <p className="payout-account-complete-page__notice-title">
            売上の受取について
          </p>

          <p className="payout-account-complete-page__notice-text">
            登録した口座が実際に売上の受取先として利用可能になるまで、確認が必要な場合があります。利用状況は売上受取口座画面から確認できます。
          </p>
        </div>

        <p className="payout-account-complete-page__security-note">
          入力した口座番号の全桁は、この画面への遷移後にブラウザ上の登録情報から削除されます。
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