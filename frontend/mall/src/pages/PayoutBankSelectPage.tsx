// frontend/mall/src/pages/PayoutBankSelectPage.tsx

import { useMemo, useState } from "react";
import { Check, Search } from "lucide-react";
import { useNavigate } from "react-router-dom";

import FooterNav from "../components/layout/FooterNav";
import Layout from "../components/layout/Layout";
import Input from "../components/ui/Input";
import List, { ListRow } from "../components/ui/List";
import TextState from "../components/ui/TextState";
import { useContactViewport } from "../features/contact/hooks/useContactViewport";
import { usePayoutAccountRegistration } from "../features/payout/context/PayoutAccountRegistrationProvider";
import { usePayoutAccountRegistrationRules } from "../features/payout/hooks/usePayoutAccountRegistrationRules";

import "../styles/page-layout.css";
import "../styles/settings-page.css";
import "../styles/payout.css";

type BankCandidate = {
  bankCode: string;
  bankName: string;
  searchKeywords: string;
};

const BANK_CANDIDATES: BankCandidate[] = [
  {
    bankCode: "0001",
    bankName: "みずほ銀行",
    searchKeywords: "みずほ ミズホ mizuho",
  },
  {
    bankCode: "0005",
    bankName: "三菱UFJ銀行",
    searchKeywords: "三菱ufj みつびしufj ミツビシufj mufg",
  },
  {
    bankCode: "0009",
    bankName: "三井住友銀行",
    searchKeywords: "三井住友 みついすみとも ミツイスミトモ smbc",
  },
  {
    bankCode: "0010",
    bankName: "りそな銀行",
    searchKeywords: "りそな リソナ resona",
  },
  {
    bankCode: "0033",
    bankName: "PayPay銀行",
    searchKeywords: "paypay ペイペイ",
  },
  {
    bankCode: "0034",
    bankName: "セブン銀行",
    searchKeywords: "セブン せぶん seven",
  },
  {
    bankCode: "0035",
    bankName: "ソニー銀行",
    searchKeywords: "ソニー そにー sony",
  },
  {
    bankCode: "0036",
    bankName: "楽天銀行",
    searchKeywords: "楽天 らくてん ラクテン rakuten",
  },
  {
    bankCode: "0038",
    bankName: "住信SBIネット銀行",
    searchKeywords: "住信sbi すみしんsbi スミシンsbi sbi",
  },
  {
    bankCode: "9900",
    bankName: "ゆうちょ銀行",
    searchKeywords: "ゆうちょ ユウチョ japan post",
  },
];

function normalizeSearchValue(value: string): string {
  return value.trim().toLowerCase().replace(/\s+/g, "");
}

export default function PayoutBankSelectPage() {
  const navigate = useNavigate();
  const { isDesktop } = useContactViewport();
  const { draft, setBank } = usePayoutAccountRegistration();
  const { validateBankCode } = usePayoutAccountRegistrationRules();

  const [searchText, setSearchText] = useState("");
  const [selectedBankCode, setSelectedBankCode] = useState(draft.bankCode);

  const filteredBanks = useMemo(() => {
    const query = normalizeSearchValue(searchText);

    if (!query) {
      return BANK_CANDIDATES;
    }

    return BANK_CANDIDATES.filter((bank) => {
      const searchable = normalizeSearchValue(
        `${bank.bankCode} ${bank.bankName} ${bank.searchKeywords}`,
      );

      return searchable.includes(query);
    });
  }, [searchText]);

  const selectedBank = useMemo(
    () =>
      BANK_CANDIDATES.find((bank) => bank.bankCode === selectedBankCode) ??
      null,
    [selectedBankCode],
  );

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
      mode="default"
      hideHamburgerMenu
      hideSettingsButton
      actionButtonLabel={isDesktop ? "次へ" : undefined}
      onActionButtonClick={isDesktop ? handleNext : undefined}
      actionButtonDisabled={actionButtonDisabled}
    >
      <section className="page-section content-page-section settings-page payout-select-page payout-bank-select-page">
        <p className="content-page-description payout-select-page__description">
          売上の受取口座として使用する金融機関を選択してください。
        </p>

        <div className="payout-select__search">
          <Search
            className="payout-select__search-icon"
            size={20}
            aria-hidden="true"
          />

          <Input
            type="search"
            value={searchText}
            onChange={(event) => setSearchText(event.target.value)}
            placeholder="金融機関名・金融機関コードで検索"
            aria-label="金融機関を検索"
            autoComplete="off"
          />
        </div>

        {filteredBanks.length > 0 ? (
          <List className="payout-select__list">
            {filteredBanks.map((bank) => {
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
        ) : (
          <div className="payout-select__empty">
            <TextState variant="empty">
              該当する金融機関が見つかりません
            </TextState>

            <TextState variant="muted">
              金融機関名または金融機関コードを確認して、もう一度検索してください。
            </TextState>
          </div>
        )}

        <TextState variant="muted" className="payout-select__note">
          金融機関一覧は現在開発用データを使用しています。本番接続時は金融機関情報提供元のデータに切り替えます。
        </TextState>
      </section>

      {!isDesktop ? (
        <FooterNav
          variant="action"
          buttonLabel="次へ"
          disabled={actionButtonDisabled}
          onButtonClick={handleNext}
        />
      ) : null}
    </Layout>
  );
}