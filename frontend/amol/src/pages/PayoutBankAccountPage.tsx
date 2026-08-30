// frontend/amol/src/pages/PayoutBankAccountPage.tsx

import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import "../styles/page-layout.css";
import "../styles/settings-page.css";
import "../styles/payout-bank-account-page.css";

import Layout from "../components/layout/Layout";
import FooterNav from "../components/layout/FooterNav";
import { useContactViewport } from "../features/contact/hooks/useContactViewport";
import { usePayoutAccountRegistration } from "../features/payout/context/PayoutAccountRegistrationProvider";
import { usePayoutAccountRegistrationRules } from "../features/payout/hooks/usePayoutAccountRegistrationRules";
import type { PayoutBankAccountType } from "../features/shared/types/payoutAccount";

export default function PayoutBankAccountPage() {
  const navigate = useNavigate();
  const { isDesktop } = useContactViewport();
  const { draft, setAccountDetails } = usePayoutAccountRegistration();
  const {
    isLoading,
    isReady,
    isTestMode,
    errorMessage,
    testAccountNumber,
    normalizeAccountNumber,
    validateBankCode,
    validateBranchCode,
    validateAccountNumber,
  } = usePayoutAccountRegistrationRules();

  const [accountType, setAccountType] = useState<PayoutBankAccountType>(
    draft.accountType || "ordinary",
  );
  const [accountNumber, setAccountNumber] = useState(draft.accountNumber);
  const [accountHolderName, setAccountHolderName] = useState(draft.accountHolderName);

  useEffect(() => {
    if (!draft.bankCode.trim() || !draft.bankName.trim()) {
      navigate("/settings/payout-account/bank", { replace: true });
      return;
    }

    if (!draft.branchCode.trim() || !draft.branchName.trim()) {
      navigate("/settings/payout-account/branch", { replace: true });
    }
  }, [
    draft.bankCode,
    draft.bankName,
    draft.branchCode,
    draft.branchName,
    navigate,
  ]);

  useEffect(() => {
    if (!isReady) {
      return;
    }

    if (validateBankCode(draft.bankCode)) {
      navigate("/settings/payout-account/bank", { replace: true });
      return;
    }

    if (validateBranchCode(draft.bankCode, draft.branchCode)) {
      navigate("/settings/payout-account/branch", { replace: true });
    }
  }, [
    draft.bankCode,
    draft.branchCode,
    isReady,
    navigate,
    validateBankCode,
    validateBranchCode,
  ]);

  useEffect(() => {
    if (isReady && isTestMode) {
      setAccountNumber(testAccountNumber);
    }
  }, [isReady, isTestMode, testAccountNumber]);

  const normalizedAccountHolderName = accountHolderName.trim();

  const bankCodeError = useMemo(() => {
    if (!isReady) {
      return "";
    }

    return validateBankCode(draft.bankCode);
  }, [draft.bankCode, isReady, validateBankCode]);

  const branchCodeError = useMemo(() => {
    if (!isReady) {
      return "";
    }

    return validateBranchCode(draft.bankCode, draft.branchCode);
  }, [
    draft.bankCode,
    draft.branchCode,
    isReady,
    validateBranchCode,
  ]);

  const accountNumberError = useMemo(() => {
    if (!accountNumber || !isReady) {
      return "";
    }

    return validateAccountNumber(
      draft.bankCode,
      draft.branchCode,
      accountNumber,
    );
  }, [
    accountNumber,
    draft.bankCode,
    draft.branchCode,
    isReady,
    validateAccountNumber,
  ]);

  const actionButtonDisabled =
    isLoading ||
    !isReady ||
    Boolean(bankCodeError) ||
    Boolean(branchCodeError) ||
    !accountNumber ||
    Boolean(accountNumberError) ||
    !normalizedAccountHolderName;

  const handleAccountNumberChange = (value: string) => {
    if (isTestMode) {
      return;
    }

    setAccountNumber(normalizeAccountNumber(value));
  };

  const handleNext = () => {
    if (!isReady || actionButtonDisabled) {
      return;
    }

    const currentBankCodeError = validateBankCode(draft.bankCode);

    if (currentBankCodeError) {
      navigate("/settings/payout-account/bank", { replace: true });
      return;
    }

    const currentBranchCodeError = validateBranchCode(
      draft.bankCode,
      draft.branchCode,
    );

    if (currentBranchCodeError) {
      navigate("/settings/payout-account/branch", { replace: true });
      return;
    }

    const currentAccountNumberError = validateAccountNumber(
      draft.bankCode,
      draft.branchCode,
      accountNumber,
    );

    if (currentAccountNumberError) {
      return;
    }

    setAccountDetails({
      accountType,
      accountNumber,
      accountHolderName: normalizedAccountHolderName,
    });

    navigate("/settings/payout-account/confirm");
  };

  return (
    <Layout
      title="口座情報を入力"
      titleClickable={false}
      showBackButton
      mode="default"
      backTo="/settings/payout-account/branch"
      hideHamburgerMenu
      hideSettingsButton
      actionButtonLabel={isDesktop ? "確認へ進む" : undefined}
      onActionButtonClick={isDesktop ? handleNext : undefined}
      actionButtonDisabled={actionButtonDisabled}
    >
      <section className="page-section content-page-section settings-page payout-bank-account-page">
        <p className="content-page-description payout-bank-account-page__description">
          売上を受け取る銀行口座の情報を入力してください。
        </p>

        <div className="payout-bank-account-page__destination">
          <div className="payout-bank-account-page__destination-row">
            <span className="payout-bank-account-page__destination-label">
              金融機関
            </span>
            <div className="payout-bank-account-page__destination-value">
              <strong>{draft.bankName}</strong>
              <span>金融機関コード {draft.bankCode}</span>
            </div>
          </div>

          <div className="payout-bank-account-page__destination-row">
            <span className="payout-bank-account-page__destination-label">
              支店
            </span>
            <div className="payout-bank-account-page__destination-value">
              <strong>{draft.branchName}</strong>
              <span>支店コード {draft.branchCode}</span>
            </div>
          </div>
        </div>

        {isLoading ? (
          <p className="content-page-description payout-bank-account-page__description">
            Stripe設定を確認しています...
          </p>
        ) : null}

        {!isLoading && errorMessage ? (
          <p className="payout-bank-account-page__error" role="alert">
            {errorMessage}
          </p>
        ) : null}

        <div className="payout-bank-account-page__form">
          <fieldset
            className="payout-bank-account-page__field payout-bank-account-page__fieldset"
            disabled={!isReady}
          >
            <legend className="payout-bank-account-page__label">
              口座種別
            </legend>

            <div className="payout-bank-account-page__account-type">
              <label
                className={[
                  "payout-bank-account-page__account-type-option",
                  accountType === "ordinary"
                    ? "payout-bank-account-page__account-type-option--selected"
                    : "",
                ]
                  .filter(Boolean)
                  .join(" ")}
              >
                <input
                  type="radio"
                  name="accountType"
                  value="ordinary"
                  checked={accountType === "ordinary"}
                  onChange={() => setAccountType("ordinary")}
                />
                <span>普通</span>
              </label>

              <label
                className={[
                  "payout-bank-account-page__account-type-option",
                  accountType === "current"
                    ? "payout-bank-account-page__account-type-option--selected"
                    : "",
                ]
                  .filter(Boolean)
                  .join(" ")}
              >
                <input
                  type="radio"
                  name="accountType"
                  value="current"
                  checked={accountType === "current"}
                  onChange={() => setAccountType("current")}
                />
                <span>当座</span>
              </label>
            </div>
          </fieldset>

          <div className="payout-bank-account-page__field">
            <label
              htmlFor="payout-account-number"
              className="payout-bank-account-page__label"
            >
              口座番号
            </label>

            <input
              id="payout-account-number"
              type="text"
              inputMode="numeric"
              autoComplete="off"
              maxLength={7}
              value={accountNumber}
              onChange={(event) => handleAccountNumberChange(event.target.value)}
              placeholder={isTestMode ? testAccountNumber : "1234567"}
              readOnly={isTestMode}
              disabled={!isReady}
              className={[
                "payout-bank-account-page__input",
                accountNumberError
                  ? "payout-bank-account-page__input--error"
                  : "",
              ]
                .filter(Boolean)
                .join(" ")}
              aria-invalid={Boolean(accountNumberError)}
              aria-describedby={
                accountNumberError
                  ? "payout-account-number-error"
                  : "payout-account-number-help"
              }
            />

            {accountNumberError ? (
              <p
                id="payout-account-number-error"
                className="payout-bank-account-page__error"
              >
                {accountNumberError}
              </p>
            ) : (
              <p
                id="payout-account-number-help"
                className="payout-bank-account-page__help"
              >
                {isTestMode
                  ? `開発環境ではStripeのテスト口座番号 ${testAccountNumber} を使用します。`
                  : "7桁の口座番号を入力してください。"}
              </p>
            )}
          </div>

          <div className="payout-bank-account-page__field">
            <label
              htmlFor="payout-account-holder-name"
              className="payout-bank-account-page__label"
            >
              口座名義
            </label>

            <input
              id="payout-account-holder-name"
              type="text"
              autoComplete="off"
              maxLength={64}
              value={accountHolderName}
              onChange={(event) => setAccountHolderName(event.target.value)}
              placeholder="例：ヤマダ タロウ"
              className="payout-bank-account-page__input"
              disabled={!isReady}
            />

            <p className="payout-bank-account-page__help">
              通帳や銀行アプリに登録されている口座名義を入力してください。
            </p>
          </div>
        </div>

        <div className="payout-bank-account-page__notice">
          <p className="payout-bank-account-page__notice-title">
            口座情報の取り扱い
          </p>
          <p className="payout-bank-account-page__notice-text">
            入力した口座番号は登録処理のために使用します。登録後、AMOLの画面では口座番号の末尾4桁のみを表示します。
          </p>
        </div>
      </section>

      {!isDesktop ? (
        <FooterNav
          variant="action"
          buttonLabel="確認へ進む"
          disabled={actionButtonDisabled}
          onButtonClick={handleNext}
        />
      ) : null}
    </Layout>
  );
}