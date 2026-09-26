// frontend/mall/src/pages/PayoutBranchSelectPage.tsx

import { useEffect, useState } from "react";
import { Check } from "lucide-react";
import { useNavigate } from "react-router-dom";

import Layout from "../components/layout/Layout";
import Button from "../components/ui/Button";
import Card from "../components/ui/Card";
import List, { ListRow } from "../components/ui/List";
import TextState from "../components/ui/TextState";
import { usePayoutAccountRegistration } from "../features/payout/context/PayoutAccountRegistrationProvider";
import { usePayoutAccountRegistrationRules } from "../features/payout/hooks/usePayoutAccountRegistrationRules";

import "../styles/page-layout.css";
import "../styles/settings-page.css";
import "../styles/payout.css";
import "../styles/payout-branch-select-page.css";

type BranchCandidate = {
  branchCode: string;
  branchName: string;
};

const MOCK_BRANCH_CANDIDATES: BranchCandidate[] = [
  {
    branchCode: "001",
    branchName: "本店（開発用）",
  },
  {
    branchCode: "101",
    branchName: "東京支店（開発用）",
  },
  {
    branchCode: "201",
    branchName: "大阪支店（開発用）",
  },
];

function hasNonWhitespace(value: string): boolean {
  return /\S/.test(value);
}

export default function PayoutBranchSelectPage() {
  const navigate = useNavigate();
  const { draft, setBranch } = usePayoutAccountRegistration();
  const { validateBankCode, validateBranchCode } =
    usePayoutAccountRegistrationRules();

  const [selectedBranchCode, setSelectedBranchCode] = useState(draft.branchCode);

  useEffect(() => {
    if (
      validateBankCode(draft.bankCode) ||
      !hasNonWhitespace(draft.bankName)
    ) {
      navigate("/settings/payout-account/bank", { replace: true });
    }
  }, [
    draft.bankCode,
    draft.bankName,
    navigate,
    validateBankCode,
  ]);

  const selectedBranch =
    MOCK_BRANCH_CANDIDATES.find(
      (branch) => branch.branchCode === selectedBranchCode,
    ) ?? null;

  const handleSelectBranch = (branch: BranchCandidate) => {
    if (validateBranchCode(branch.branchCode)) {
      return;
    }

    setSelectedBranchCode(branch.branchCode);
  };

  const handleNext = () => {
    if (!selectedBranch) {
      return;
    }

    if (
      validateBankCode(draft.bankCode) ||
      !hasNonWhitespace(draft.bankName)
    ) {
      navigate("/settings/payout-account/bank", { replace: true });
      return;
    }

    if (validateBranchCode(selectedBranch.branchCode)) {
      return;
    }

    setBranch({
      branchCode: selectedBranch.branchCode,
      branchName: selectedBranch.branchName,
    });

    navigate("/settings/payout-account/account");
  };

  const selectedBranchError = selectedBranch
    ? validateBranchCode(selectedBranch.branchCode)
    : "";

  const actionButtonDisabled =
    !selectedBranch ||
    Boolean(selectedBranchError);

  return (
    <Layout
      title="支店を選択"
      titleClickable={false}
      mode="mypage"
      showFooter
      hideHamburgerMenu
    >
      <section className="page-section content-page-section settings-page payout-select-page payout-branch-select-page">
        <p className="content-page-description payout-select-page__description">
          売上の受取口座として使用する支店を選択してください。
        </p>

        <Card
          padding="md"
          className="payout-branch-select-page__bank-summary"
        >
          <span className="payout-branch-select-page__bank-summary-label">
            金融機関
          </span>

          <div className="payout-branch-select-page__bank-summary-content">
            <strong className="payout-branch-select-page__bank-summary-name">
              {draft.bankName}
            </strong>

            <span className="payout-branch-select-page__bank-summary-code">
              金融機関コード {draft.bankCode}
            </span>
          </div>
        </Card>

        <List className="payout-select__list">
          {MOCK_BRANCH_CANDIDATES.map((branch) => {
            const selected = branch.branchCode === selectedBranchCode;

            return (
              <ListRow
                key={branch.branchCode}
                title={branch.branchName}
                subLabel={`支店コード ${branch.branchCode}`}
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
                ariaLabel={`${branch.branchName} 支店コード ${branch.branchCode}${
                  selected ? " 選択中" : ""
                }`}
                onClick={() => handleSelectBranch(branch)}
              />
            );
          })}
        </List>

        <TextState variant="muted" className="payout-select__note">
          支店一覧は現在開発用データを使用しています。本番接続時は金融機関情報提供元のデータに切り替えます。
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