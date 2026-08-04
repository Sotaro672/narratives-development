// frontend/amol/src/features/resale/presentation/components/ResaleCreateMissingTarget.tsx

import SectionHeader from "../../../../components/ui/SectionHeader";

export type ResaleCreateMissingTargetProps = {
  onBackToWallet: () => void;
};

export default function ResaleCreateMissingTarget({
  onBackToWallet,
}: ResaleCreateMissingTargetProps) {
  return (
    <div className="page-card">
      <SectionHeader
        title="出品情報が見つかりません"
        titleAs="h2"
      >
        <p className="page-card__text">
          ウォレットまたはトークン詳細から、
          もう一度出品ボタンを押してください。
        </p>
      </SectionHeader>

      <button
        type="button"
        className="page-button page-button--primary"
        onClick={onBackToWallet}
      >
        ウォレットへ戻る
      </button>
    </div>
  );
}