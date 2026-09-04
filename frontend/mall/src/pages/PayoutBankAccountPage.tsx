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

function hasNonWhitespace(value: string): boolean {
  return /\S/.test(value);
}

export default function PayoutBankAccountPage() {
  const navigate = useNavigate();
  const { isDesktop } = useContactViewport();
  const { draft, setAccountDetails } = usePayoutAccountRegistration();
  const {
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
    if (validateBankCode(draft.bankCode) || !hasNonWhitespace(draft.bankName)) {
      navigate("/settings/payout-account/bank", { replace: true });
      return;
    }

    if (validateBranchCode(draft.branchCode) || !hasNonWhitespace(draft.branchName)) {
      navigate("/settings/payout-account/branch", { replace: true });
    }
  }, [
    draft.bankCode,
    draft.bankName,
    draft.branchCode,
    draft.branchName,
    navigate,
    validateBankCode,
    validateBranchCode,
  ]);

  const bankCodeError = useMemo(
    () => validateBankCode(draft.bankCode),
    [draft.bankCode, validateBankCode],
  );

  const branchCodeError = useMemo(
    () => validateBranchCode(draft.branchCode),
    [draft.branchCode, validateBranchCode],
  );

  const accountNumberError = useMemo(() => {
    if (!accountNumber) {
      return "";
    }

    return validateAccountNumber(accountNumber);
  }, [accountNumber, validateAccountNumber]);

  const accountHolderNameValid = hasNonWhitespace(accountHolderName);

  const actionButtonDisabled =
    Boolean(bankCodeError) ||
    Boolean(branchCodeError) ||
    !accountNumber ||
    Boolean(accountNumberError) ||
    !accountHolderNameValid;

  const handleNext = () => {
    if (actionButtonDisabled) {
      return;
    }

    const currentBankCodeError = validateBankCode(draft.bankCode);
    if (currentBankCodeError || !hasNonWhitespace(draft.bankName)) {
      navigate("/settings/payout-account/bank", { replace: true });
      return;
    }

    const currentBranchCodeError = validateBranchCode(draft.branchCode);
    if (currentBranchCodeError || !hasNonWhitespace(draft.branchName)) {
      navigate("/settings/payout-account/branch", { replace: true });
      return;
    }

    const currentAccountNumberError = validateAccountNumber(accountNumber);
    if (currentAccountNumberError || !hasNonWhitespace(accountHolderName)) {
      return;
    }

    setAccountDetails({
      accountType,
      accountNumber,
      accountHolderName,
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

        <div className="payout-bank-account-page__form">
          <fieldset className="payout-bank-account-page__field payout-bank-account-page__fieldset">
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
              onChange={(event) => setAccountNumber(event.target.value)}
              placeholder="1234567"
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
                7桁の口座番号を入力してください。
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