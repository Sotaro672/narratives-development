// frontend/amol/src/pages/PayoutBranchSelectPage.tsx

import { useEffect, useMemo, useState } from "react";
import { Check, Search } from "lucide-react";
import { useNavigate } from "react-router-dom";

import "../styles/page-layout.css";
import "../styles/settings-page.css";
import "../styles/payout-branch-select-page.css";

import Layout from "../components/layout/Layout";
import FooterNav from "../components/layout/FooterNav";
import { useContactViewport } from "../features/contact/hooks/useContactViewport";
import { usePayoutAccountRegistration } from "../features/payout/context/PayoutAccountRegistrationProvider";
import { usePayoutAccountRegistrationRules } from "../features/payout/hooks/usePayoutAccountRegistrationRules";

type BranchCandidate = {
  branchCode: string;
  branchName: string;
  searchKeywords: string;
};

const MOCK_BRANCH_CANDIDATES: BranchCandidate[] = [
  {
    branchCode: "001",
    branchName: "本店（開発用）",
    searchKeywords: "本店 ほんてん ホンテン",
  },
  {
    branchCode: "101",
    branchName: "東京支店（開発用）",
    searchKeywords: "東京 とうきょう トウキョウ",
  },
  {
    branchCode: "201",
    branchName: "大阪支店（開発用）",
    searchKeywords: "大阪 おおさか オオサカ",
  },
];

function normalizeSearchValue(value: string): string {
  return value.trim().toLowerCase().replace(/\s+/g, "");
}

export default function PayoutBranchSelectPage() {
  const navigate = useNavigate();
  const { isDesktop } = useContactViewport();
  const { draft, setBranch } = usePayoutAccountRegistration();
  const {
    isLoading,
    isReady,
    isTestMode,
    errorMessage,
    testBranchCode,
    validateBankCode,
    validateBranchCode,
  } = usePayoutAccountRegistrationRules();

  const [searchText, setSearchText] = useState("");
  const [selectedBranchCode, setSelectedBranchCode] = useState(draft.branchCode);

  useEffect(() => {
    if (!draft.bankCode.trim() || !draft.bankName.trim()) {
      navigate("/settings/payout-account/bank", { replace: true });
    }
  }, [draft.bankCode, draft.bankName, navigate]);

  useEffect(() => {
    if (!isReady) {
      return;
    }

    if (validateBankCode(draft.bankCode)) {
      navigate("/settings/payout-account/bank", { replace: true });
    }
  }, [draft.bankCode, isReady, navigate, validateBankCode]);

  const branchCandidates = useMemo<BranchCandidate[]>(() => {
    if (!isReady) {
      return [];
    }

    if (isTestMode) {
      return [
        {
          branchCode: testBranchCode,
          branchName: "Stripeテスト支店",
          searchKeywords: "stripe test ストライプ テスト",
        },
      ];
    }

    return MOCK_BRANCH_CANDIDATES;
  }, [isReady, isTestMode, testBranchCode]);

  const filteredBranches = useMemo(() => {
    const query = normalizeSearchValue(searchText);

    if (!query) {
      return branchCandidates;
    }

    return branchCandidates.filter((branch) => {
      const searchable = normalizeSearchValue(
        `${branch.branchCode} ${branch.branchName} ${branch.searchKeywords}`,
      );
      return searchable.includes(query);
    });
  }, [branchCandidates, searchText]);

  const selectedBranch = useMemo(
    () => branchCandidates.find((branch) => branch.branchCode === selectedBranchCode) ?? null,
    [branchCandidates, selectedBranchCode],
  );

  const handleSelectBranch = (branch: BranchCandidate) => {
    if (!isReady || validateBranchCode(draft.bankCode, branch.branchCode)) {
      return;
    }

    setSelectedBranchCode(branch.branchCode);
  };

  const handleNext = () => {
    if (!isReady || !selectedBranch) {
      return;
    }

    if (validateBankCode(draft.bankCode)) {
      navigate("/settings/payout-account/bank", { replace: true });
      return;
    }

    if (validateBranchCode(draft.bankCode, selectedBranch.branchCode)) {
      return;
    }

    setBranch({
      branchCode: selectedBranch.branchCode,
      branchName: selectedBranch.branchName,
    });

    navigate("/settings/payout-account/account");
  };

  const selectedBranchError = selectedBranch
    ? validateBranchCode(draft.bankCode, selectedBranch.branchCode)
    : "";

  const actionButtonDisabled =
    isLoading ||
    !isReady ||
    !selectedBranch ||
    Boolean(selectedBranchError);

  return (
    <Layout
      title="支店を選択"
      titleClickable={false}
      showBackButton
      mode="default"
      backTo="/settings/payout-account/bank"
      hideHamburgerMenu
      hideSettingsButton
      actionButtonLabel={isDesktop ? "次へ" : undefined}
      onActionButtonClick={isDesktop ? handleNext : undefined}
      actionButtonDisabled={actionButtonDisabled}
    >
      <section className="page-section content-page-section settings-page payout-branch-select-page">
        <p className="content-page-description payout-branch-select-page__description">
          売上の受取口座として使用する支店を選択してください。
        </p>

        <div className="payout-branch-select-page__bank-summary">
          <span className="payout-branch-select-page__bank-summary-label">金融機関</span>
          <div className="payout-branch-select-page__bank-summary-content">
            <strong className="payout-branch-select-page__bank-summary-name">
              {draft.bankName}
            </strong>
            <span className="payout-branch-select-page__bank-summary-code">
              金融機関コード {draft.bankCode}
            </span>
          </div>
        </div>

        <div className="payout-branch-select-page__search">
          <Search
            className="payout-branch-select-page__search-icon"
            size={20}
            aria-hidden="true"
          />
          <input
            type="search"
            value={searchText}
            onChange={(event) => setSearchText(event.target.value)}
            placeholder="支店名・支店コードで検索"
            className="payout-branch-select-page__search-input"
            aria-label="支店を検索"
            autoComplete="off"
            disabled={!isReady}
          />
        </div>

        {isLoading ? (
          <p className="content-page-description payout-branch-select-page__description">
            Stripe設定を確認しています...
          </p>
        ) : null}

        {!isLoading && errorMessage ? (
          <div className="payout-branch-select-page__empty">
            <p className="payout-branch-select-page__empty-title">
              支店一覧を準備できませんでした
            </p>
            <p className="payout-branch-select-page__empty-description">
              {errorMessage}
            </p>
          </div>
        ) : null}

        {!isLoading && isReady ? (
          <div className="payout-branch-select-page__list" role="list">
            {filteredBranches.map((branch) => {
              const selected = branch.branchCode === selectedBranchCode;

              return (
                <button
                  key={branch.branchCode}
                  type="button"
                  className={[
                    "payout-branch-select-page__branch",
                    selected ? "payout-branch-select-page__branch--selected" : "",
                  ]
                    .filter(Boolean)
                    .join(" ")}
                  onClick={() => handleSelectBranch(branch)}
                  aria-pressed={selected}
                  role="listitem"
                >
                  <span className="payout-branch-select-page__branch-content">
                    <strong className="payout-branch-select-page__branch-name">
                      {branch.branchName}
                    </strong>
                    <span className="payout-branch-select-page__branch-code">
                      支店コード {branch.branchCode}
                    </span>
                  </span>

                  <span
                    className={[
                      "payout-branch-select-page__check",
                      selected ? "payout-branch-select-page__check--selected" : "",
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

            {filteredBranches.length === 0 ? (
              <div className="payout-branch-select-page__empty">
                <p className="payout-branch-select-page__empty-title">
                  該当する支店が見つかりません
                </p>
                <p className="payout-branch-select-page__empty-description">
                  支店名または支店コードを確認して、もう一度検索してください。
                </p>
              </div>
            ) : null}
          </div>
        ) : null}

        {!isLoading && isReady ? (
          <p className="payout-branch-select-page__note">
            {isTestMode
              ? `開発環境ではStripeのテスト支店（支店コード ${testBranchCode}）のみ選択できます。`
              : "支店一覧は現在開発用データを使用しています。本番接続時は金融機関情報提供元のデータに切り替えます。"}
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