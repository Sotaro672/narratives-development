// frontend/mall/src/pages/PayoutBankSelectPage.tsx

import { useState } from "react";
import { Check } from "lucide-react";
import { useNavigate } from "react-router-dom";

import Layout from "../components/layout/Layout";
import Button from "../components/ui/Button";
import List, { ListRow } from "../components/ui/List";
import TextState from "../components/ui/TextState";
import { usePayoutAccountRegistration } from "../features/payout/context/PayoutAccountRegistrationProvider";
import { usePayoutAccountRegistrationRules } from "../features/payout/hooks/usePayoutAccountRegistrationRules";

import "../styles/page-layout.css";
import "../styles/settings-page.css";
import "../styles/payout.css";

type BankCandidate = {
  bankCode: string;
  bankName: string;
};

const BANK_CANDIDATES: BankCandidate[] = [
  {
    bankCode: "0001",
    bankName: "みずほ銀行",
  },
  {
    bankCode: "0005",
    bankName: "三菱UFJ銀行",
  },
  {
    bankCode: "0009",
    bankName: "三井住友銀行",
  },
  {
    bankCode: "0010",
    bankName: "りそな銀行",
  },
  {
    bankCode: "0033",
    bankName: "PayPay銀行",
  },
  {
    bankCode: "0034",
    bankName: "セブン銀行",
  },
  {
    bankCode: "0035",
    bankName: "ソニー銀行",
  },
  {
    bankCode: "0036",
    bankName: "楽天銀行",
  },
  {
    bankCode: "0038",
    bankName: "住信SBIネット銀行",
  },
  {
    bankCode: "9900",
    bankName: "ゆうちょ銀行",
  },
];

export default function PayoutBankSelectPage() {
  const navigate = useNavigate();
  const { draft, setBank } = usePayoutAccountRegistration();
  const { validateBankCode } = usePayoutAccountRegistrationRules();

  const [selectedBankCode, setSelectedBankCode] = useState(draft.bankCode);

  const selectedBank =
    BANK_CANDIDATES.find((bank) => bank.bankCode === selectedBankCode) ?? null;

  const handleSelectBank = (bank: BankCandidate) => {
    if (validateBankCode(bank.bankCode)) {
      return;
    }

    setSelectedBankCode(bank.bankCode);
  };

  const handleNext = () => {
    if (!selectedBank) {
      return;
    }

    if (validateBankCode(selectedBank.bankCode)) {
      return;
    }

    setBank({
      bankCode: selectedBank.bankCode,
      bankName: selectedBank.bankName,
    });

    navigate("/settings/payout-account/branch");
  };

  const actionButtonDisabled =
    !selectedBank ||
    Boolean(selectedBank && validateBankCode(selectedBank.bankCode));

  return (
    <Layout
      title="金融機関を選択"
      titleClickable={false}
      mode="mypage"
      showFooter
      hideHamburgerMenu
    >
      <section className="page-section content-page-section settings-page payout-select-page payout-bank-select-page">
        <p className="content-page-description payout-select-page__description">
          売上の受取口座として使用する金融機関を選択してください。
        </p>

        <List className="payout-select__list">
          {BANK_CANDIDATES.map((bank) => {
            const selected = bank.bankCode === selectedBankCode;

            return (
              <ListRow
                key={bank.bankCode}
                title={bank.bankName}
                subLabel={`金融機関コード ${bank.bankCode}`}
                selected={selected}
                meta={
                  <span
                    className={[
                      "payout-select__check",
                      selected ? "payout-select__check--selected" : "",
                    ]
                      .filter(Boolean)
                      .join(" ")}
                    aria-hidden="true"
                  >
                    {selected ? (
                      <Check size={18} strokeWidth={2.5} />
                    ) : null}
                  </span>
                }
                ariaLabel={`${bank.bankName} 金融機関コード ${bank.bankCode}${
                  selected ? " 選択中" : ""
                }`}
                onClick={() => handleSelectBank(bank)}
              />
            );
          })}
        </List>

        <TextState variant="muted" className="payout-select__note">
          金融機関一覧は現在開発用データを使用しています。本番接続時は金融機関情報提供元のデータに切り替えます。
        </TextState>

        <div className="page-actions">
          <Button
            type="button"
            variant="primary"
            size="lg"
            fullWidth
            disabled={actionButtonDisabled}
            onClick={handleNext}
          >
            次へ
          </Button>
        </div>
      </section>
    </Layout>
  );
}