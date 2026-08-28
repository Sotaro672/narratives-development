// frontend/amol/src/pages/PayoutAccountConfirmPage.tsx

import { useCallback, useEffect, useState } from "react";
import { getAuth } from "firebase/auth";
import { useNavigate } from "react-router-dom";

import "../styles/page-layout.css";
import "../styles/settings-page.css";
import "../styles/payout-account-confirm-page.css";

import Layout from "../components/layout/Layout";
import FooterNav from "../components/layout/FooterNav";
import { useContactViewport } from "../features/contact/hooks/useContactViewport";
import { registerPayoutAccount } from "../features/payout/api/payoutApi";
import { usePayoutAccountRegistration } from "../features/payout/context/PayoutAccountRegistrationProvider";

function getAccountTypeLabel(accountType: "ordinary" | "current"): string {
  switch (accountType) {
    case "ordinary":
      return "普通";
    case "current":
      return "当座";
    default:
      return "";
  }
}

export default function PayoutAccountConfirmPage() {
  const auth = getAuth();
  const navigate = useNavigate();
  const { isDesktop } = useContactViewport();
  const { draft, isComplete } = usePayoutAccountRegistration();

  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");

  useEffect(() => {
    if (!draft.bankCode.trim() || !draft.bankName.trim()) {
      navigate("/settings/payout-account/bank", { replace: true });
      return;
    }

    if (!draft.branchCode.trim() || !draft.branchName.trim()) {
      navigate("/settings/payout-account/branch", { replace: true });
      return;
    }

    if (
      !draft.accountNumber.trim() ||
      !draft.accountHolderName.trim()
    ) {
      navigate("/settings/payout-account/account", { replace: true });
    }
  }, [
    draft.bankCode,
    draft.bankName,
    draft.branchCode,
    draft.branchName,
    draft.accountNumber,
    draft.accountHolderName,
    navigate,
  ]);

  const handleRegister = useCallback(async () => {
    if (!isComplete || isSubmitting) {
      return;
    }

    const currentUser = auth.currentUser;

    if (!currentUser) {
      navigate("/signin", { replace: true });
      return;
    }

    try {
      setIsSubmitting(true);
      setErrorMessage("");

      const idToken = await currentUser.getIdToken(true);

      await registerPayoutAccount({
        idToken,
        input: {
          bankCode: draft.bankCode.trim(),
          bankName: draft.bankName.trim(),
          branchCode: draft.branchCode.trim(),
          branchName: draft.branchName.trim(),
          accountType: draft.accountType,
          accountNumber: draft.accountNumber.trim(),
          accountHolderName: draft.accountHolderName.trim(),
        },
      });

      navigate("/settings/payout-account/complete", { replace: true });
    } catch (error) {
      console.error(error);

      setErrorMessage(
        error instanceof Error
          ? error.message
          : "売上受取口座の登録に失敗しました。"
      );
    } finally {
      setIsSubmitting(false);
    }
  }, [auth, draft, isComplete, isSubmitting, navigate]);

  const actionButtonDisabled = !isComplete || isSubmitting;

  return (
    <Layout
      title="口座情報を確認"
      titleClickable={false}
      showBackButton
      mode="default"
      backTo="/settings/payout-account/account"
      hideHamburgerMenu
      hideSettingsButton
      actionButtonLabel={
        isDesktop
          ? isSubmitting
            ? "登録中..."
            : "登録する"
          : undefined
      }
      onActionButtonClick={isDesktop ? handleRegister : undefined}
      actionButtonDisabled={actionButtonDisabled}
    >
      <section className="page-section content-page-section settings-page payout-account-confirm-page">
        <p className="content-page-description payout-account-confirm-page__description">
          入力した口座情報を確認してください。内容に間違いがなければ登録してください。
        </p>

        <div className="payout-account-confirm-page__card">
          <div className="payout-account-confirm-page__row">
            <div className="payout-account-confirm-page__row-main">
              <span className="payout-account-confirm-page__label">
                金融機関
              </span>
              <div className="payout-account-confirm-page__value">
                <strong>{draft.bankName}</strong>
                <span>金融機関コード {draft.bankCode}</span>
              </div>
            </div>

            <button
              type="button"
              className="payout-account-confirm-page__edit-button"
              onClick={() =>
                navigate("/settings/payout-account/bank")
              }
              disabled={isSubmitting}
            >
              変更
            </button>
          </div>

          <div className="payout-account-confirm-page__row">
            <div className="payout-account-confirm-page__row-main">
              <span className="payout-account-confirm-page__label">
                支店
              </span>
              <div className="payout-account-confirm-page__value">
                <strong>{draft.branchName}</strong>
                <span>支店コード {draft.branchCode}</span>
              </div>
            </div>

            <button
              type="button"
              className="payout-account-confirm-page__edit-button"
              onClick={() =>
                navigate("/settings/payout-account/branch")
              }
              disabled={isSubmitting}
            >
              変更
            </button>
          </div>

          <div className="payout-account-confirm-page__row">
            <div className="payout-account-confirm-page__row-main">
              <span className="payout-account-confirm-page__label">
                口座種別
              </span>
              <strong className="payout-account-confirm-page__single-value">
                {getAccountTypeLabel(draft.accountType)}
              </strong>
            </div>

            <button
              type="button"
              className="payout-account-confirm-page__edit-button"
              onClick={() =>
                navigate("/settings/payout-account/account")
              }
              disabled={isSubmitting}
            >
              変更
            </button>
          </div>

          <div className="payout-account-confirm-page__row">
            <div className="payout-account-confirm-page__row-main">
              <span className="payout-account-confirm-page__label">
                口座番号
              </span>
              <strong className="payout-account-confirm-page__single-value payout-account-confirm-page__account-number">
                {draft.accountNumber}
              </strong>
            </div>

            <button
              type="button"
              className="payout-account-confirm-page__edit-button"
              onClick={() =>
                navigate("/settings/payout-account/account")
              }
              disabled={isSubmitting}
            >
              変更
            </button>
          </div>

          <div className="payout-account-confirm-page__row">
            <div className="payout-account-confirm-page__row-main">
              <span className="payout-account-confirm-page__label">
                口座名義
              </span>
              <strong className="payout-account-confirm-page__single-value payout-account-confirm-page__holder-name">
                {draft.accountHolderName}
              </strong>
            </div>

            <button
              type="button"
              className="payout-account-confirm-page__edit-button"
              onClick={() =>
                navigate("/settings/payout-account/account")
              }
              disabled={isSubmitting}
            >
              変更
            </button>
          </div>
        </div>

        <div className="payout-account-confirm-page__notice">
          <p className="payout-account-confirm-page__notice-title">
            登録前にご確認ください
          </p>
          <p className="payout-account-confirm-page__notice-text">
            金融機関名、支店名、口座番号、口座名義に誤りがあると売上を受け取れない場合があります。登録内容をもう一度ご確認ください。
          </p>
        </div>

        {errorMessage ? (
          <p
            className="payout-account-confirm-page__error"
            role="alert"
          >
            {errorMessage}
          </p>
        ) : null}
      </section>

      {!isDesktop ? (
        <FooterNav
          variant="action"
          buttonLabel={isSubmitting ? "登録中..." : "登録する"}
          disabled={actionButtonDisabled}
          onButtonClick={handleRegister}
        />
      ) : null}
    </Layout>
  );
}