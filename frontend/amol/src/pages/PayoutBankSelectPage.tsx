// frontend/amol/src/pages/PayoutBankSelectPage.tsx

import { useMemo, useState } from "react";
import { Check, Search } from "lucide-react";
import { useNavigate } from "react-router-dom";

import "../styles/page-layout.css";
import "../styles/settings-page.css";
import "../styles/payout-bank-select-page.css";

import Layout from "../components/layout/Layout";
import FooterNav from "../components/layout/FooterNav";
import { useContactViewport } from "../features/contact/hooks/useContactViewport";
import { usePayoutAccountRegistration } from "../features/payout/context/PayoutAccountRegistrationProvider";
import { usePayoutAccountRegistrationRules } from "../features/payout/hooks/usePayoutAccountRegistrationRules";

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
  const {
    isLoading,
    isReady,
    isTestMode,
    errorMessage,
    testBankCode,
    validateBankCode,
  } = usePayoutAccountRegistrationRules();

  const [searchText, setSearchText] = useState("");
  const [selectedBankCode, setSelectedBankCode] = useState(draft.bankCode);

  const bankCandidates = useMemo<BankCandidate[]>(() => {
    if (!isReady) {
      return [];
    }

    if (isTestMode) {
      return [
        {
          bankCode: testBankCode,
          bankName: "Stripeテスト銀行",
          searchKeywords: "stripe test ストライプ テスト",
        },
      ];
    }

    return BANK_CANDIDATES;
  }, [isReady, isTestMode, testBankCode]);

  const filteredBanks = useMemo(() => {
    const query = normalizeSearchValue(searchText);

    if (!query) {
      return bankCandidates;
    }

    return bankCandidates.filter((bank) => {
      const searchable = normalizeSearchValue(
        `${bank.bankCode} ${bank.bankName} ${bank.searchKeywords}`,
      );

      return searchable.includes(query);
    });
  }, [bankCandidates, searchText]);

  const selectedBank = useMemo(
    () =>
      bankCandidates.find((bank) => bank.bankCode === selectedBankCode) ??
      null,
    [bankCandidates, selectedBankCode],
  );

  const handleSelectBank = (bank: BankCandidate) => {
    if (!isReady || validateBankCode(bank.bankCode)) {
      return;
    }

    setSelectedBankCode(bank.bankCode);
  };

  const handleNext = () => {
    if (!isReady || !selectedBank) {
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
    isLoading ||
    !isReady ||
    !selectedBank ||
    Boolean(selectedBank && validateBankCode(selectedBank.bankCode));

  return (
    <Layout
      title="金融機関を選択"
      titleClickable={false}
      showBackButton
      mode="default"
      backTo="/settings/payout-account"
      hideHamburgerMenu
      hideSettingsButton
      actionButtonLabel={isDesktop ? "次へ" : undefined}
      onActionButtonClick={isDesktop ? handleNext : undefined}
      actionButtonDisabled={actionButtonDisabled}
    >
      <section className="page-section content-page-section settings-page payout-bank-select-page">
        <p className="content-page-description payout-bank-select-page__description">
          売上の受取口座として使用する金融機関を選択してください。
        </p>

        <div className="payout-bank-select-page__search">
          <Search
            className="payout-bank-select-page__search-icon"
            size={20}
            aria-hidden="true"
          />

          <input
            type="search"
            value={searchText}
            onChange={(event) => setSearchText(event.target.value)}
            placeholder="金融機関名・金融機関コードで検索"
            className="payout-bank-select-page__search-input"
            aria-label="金融機関を検索"
            autoComplete="off"
            disabled={!isReady}
          />
        </div>

        {isLoading ? (
          <p className="content-page-description payout-bank-select-page__description">
            Stripe設定を確認しています...
          </p>
        ) : null}

        {!isLoading && errorMessage ? (
          <div className="payout-bank-select-page__empty">
            <p className="payout-bank-select-page__empty-title">
              金融機関一覧を準備できませんでした
            </p>
            <p className="payout-bank-select-page__empty-description">
              {errorMessage}
            </p>
          </div>
        ) : null}

        {!isLoading && isReady ? (
          <div className="payout-bank-select-page__list" role="list">
            {filteredBanks.map((bank) => {
              const selected = bank.bankCode === selectedBankCode;

              return (
                <button
                  key={bank.bankCode}
                  type="button"
                  className={[
                    "payout-bank-select-page__bank",
                    selected
                      ? "payout-bank-select-page__bank--selected"
                      : "",
                  ]
                    .filter(Boolean)
                    .join(" ")}
                  onClick={() => handleSelectBank(bank)}
                  aria-pressed={selected}
                  role="listitem"
                >
                  <span className="payout-bank-select-page__bank-content">
                    <strong className="payout-bank-select-page__bank-name">
                      {bank.bankName}
                    </strong>
                    <span className="payout-bank-select-page__bank-code">
                      金融機関コード {bank.bankCode}
                    </span>
                  </span>

                  <span
                    className={[
                      "payout-bank-select-page__check",
                      selected
                        ? "payout-bank-select-page__check--selected"
                        : "",
                    ]
                      .filter(Boolean)
                      .join(" ")}
                    aria-hidden="true"
                  >
                    {selected ? <Check size={18} strokeWidth={2.5} /> : null}
                  </span>
                </button>
              );
            })}

            {filteredBanks.length === 0 ? (
              <div className="payout-bank-select-page__empty">
                <p className="payout-bank-select-page__empty-title">
                  該当する金融機関が見つかりません
                </p>
                <p className="payout-bank-select-page__empty-description">
                  金融機関名または金融機関コードを確認して、もう一度検索してください。
                </p>
              </div>
            ) : null}
          </div>
        ) : null}

        {!isLoading && isReady ? (
          <p className="payout-bank-select-page__note">
            {isTestMode
              ? `開発環境ではStripeのテスト銀行（金融機関コード ${testBankCode}）のみ選択できます。`
              : "金融機関一覧は現在開発用データを使用しています。本番接続時は金融機関情報提供元のデータに切り替えます。"}
          </p>
        ) : null}
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