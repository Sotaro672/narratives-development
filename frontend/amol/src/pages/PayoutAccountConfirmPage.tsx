// frontend/amol/src/pages/PayoutAccountConfirmPage.tsx

import { useCallback, useEffect } from "react";
import { useNavigate } from "react-router-dom";

import "../styles/page-layout.css";
import "../styles/settings-page.css";
import "../styles/payout-account-confirm-page.css";

import Layout from "../components/layout/Layout";
import FooterNav from "../components/layout/FooterNav";
import { useContactViewport } from "../features/contact/hooks/useContactViewport";
import { usePayoutAccountRegistration } from "../features/payout/context/PayoutAccountRegistrationProvider";
import { usePayoutAccountRegistrationRules } from "../features/payout/hooks/usePayoutAccountRegistrationRules";
import { usePayoutAccountRegistrationSubmit } from "../features/payout/hooks/usePayoutAccountRegistrationSubmit";

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
  const navigate = useNavigate();
  const { isDesktop } = useContactViewport();

  const {
    draft,
    isComplete,
    returnAfterRegistration,
    resetDraft,
  } = usePayoutAccountRegistration();

  const {
    isLoading: isRulesLoading,
    isReady: isRulesReady,
    errorMessage: rulesErrorMessage,
    validateBankCode,
    validateBranchCode,
    validateAccountNumber,
    isRegistrationInputValid,
  } = usePayoutAccountRegistrationRules();

  const {
    isSubmitting,
    errorMessage,
    clearError,
    submitPayoutAccountRegistration,
  } = usePayoutAccountRegistrationSubmit();

  useEffect(() => {
    if (!draft.bankCode.trim() || !draft.bankName.trim()) {
      navigate("/settings/payout-account/bank", { replace: true });
      return;
    }

    if (!draft.branchCode.trim() || !draft.branchName.trim()) {
      navigate("/settings/payout-account/branch", { replace: true });
      return;
    }

    if (!draft.accountNumber.trim() || !draft.accountHolderName.trim()) {
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

  useEffect(() => {
    if (!isRulesReady) {
      return;
    }

    if (validateBankCode(draft.bankCode)) {
      navigate("/settings/payout-account/bank", { replace: true });
      return;
    }

    if (validateBranchCode(draft.bankCode, draft.branchCode)) {
      navigate("/settings/payout-account/branch", { replace: true });
      return;
    }

    if (
      validateAccountNumber(
        draft.bankCode,
        draft.branchCode,
        draft.accountNumber,
      )
    ) {
      navigate("/settings/payout-account/account", { replace: true });
    }
  }, [
    draft.accountNumber,
    draft.bankCode,
    draft.branchCode,
    isRulesReady,
    navigate,
    validateAccountNumber,
    validateBankCode,
    validateBranchCode,
  ]);

  const registrationInputValid =
    isRulesReady &&
    isComplete &&
    isRegistrationInputValid(draft);

  const handleRegister = useCallback(async () => {
    if (
      isRulesLoading ||
      !isRulesReady ||
      !isComplete ||
      !isRegistrationInputValid(draft) ||
      isSubmitting
    ) {
      return;
    }

    try {
      const payoutAccount = await submitPayoutAccountRegistration(draft);

      resetDraft();

      if (
        returnAfterRegistration &&
        payoutAccount.status === "registered" &&
        payoutAccount.payoutReady
      ) {
        navigate(returnAfterRegistration.pathname, {
          replace: true,
          state: returnAfterRegistration.state,
        });
        return;
      }

      navigate("/settings/payout-account/complete", {
        replace: true,
        state: {
          registrationCompleted: true,
        },
      });
    } catch {
      // エラーメッセージはusePayoutAccountRegistrationSubmit側で管理する。
    }
  }, [
    draft,
    isComplete,
    isRegistrationInputValid,
    isRulesLoading,
    isRulesReady,
    isSubmitting,
    navigate,
    resetDraft,
    returnAfterRegistration,
    submitPayoutAccountRegistration,
  ]);

  const handleEditBank = useCallback(() => {
    clearError();
    navigate("/settings/payout-account/bank");
  }, [clearError, navigate]);

  const handleEditBranch = useCallback(() => {
    clearError();
    navigate("/settings/payout-account/branch");
  }, [clearError, navigate]);

  const handleEditAccount = useCallback(() => {
    clearError();
    navigate("/settings/payout-account/account");
  }, [clearError, navigate]);

  const actionButtonDisabled =
    isRulesLoading ||
    !registrationInputValid ||
    isSubmitting;

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
            : isRulesLoading
              ? "確認中..."
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
              onClick={handleEditBank}
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
              onClick={handleEditBranch}
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
              onClick={handleEditAccount}
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
              onClick={handleEditAccount}
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
              onClick={handleEditAccount}
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

        {isRulesLoading ? (
          <p className="content-page-description payout-account-confirm-page__description">
            Stripe設定を確認しています...
          </p>
        ) : null}

        {!isRulesLoading && rulesErrorMessage ? (
          <p className="payout-account-confirm-page__error" role="alert">
            {rulesErrorMessage}
          </p>
        ) : null}

        {errorMessage ? (
          <p className="payout-account-confirm-page__error" role="alert">
            {errorMessage}
          </p>
        ) : null}
      </section>

      {!isDesktop ? (
        <FooterNav
          variant="action"
          buttonLabel={
            isSubmitting
              ? "登録中..."
              : isRulesLoading
                ? "確認中..."
                : "登録する"
          }
          disabled={actionButtonDisabled}
          onButtonClick={handleRegister}
        />
      ) : null}
    </Layout>
  );
}