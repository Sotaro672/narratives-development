//frontend\amol\src\features\contents\components\ContentsTokenSummaryCard.tsx
import Tab from "../../../components/ui/Tab";
import TokenReviewAggregateCard from "../../token-commnet/components/TokenReviewAggregateCard";
import type { ContentsSearchParams } from "../types";

type ContentsTokenSummaryCardProps = {
  contents: ContentsSearchParams;
  tokenName: string;
  tokenIconUrl: string;
  loadingAvatarId: boolean;
  currentAvatarId: string;
  onProductNameClick: () => void;
};

export default function ContentsTokenSummaryCard({
  contents,
  tokenName,
  tokenIconUrl,
  loadingAvatarId,
  currentAvatarId,
  onProductNameClick,
}: ContentsTokenSummaryCardProps) {
  return (
    <div className="contents-page-card">
      <div className="contents-page-card__header">
        <div className="contents-page-card__icon-wrap">
          {tokenIconUrl ? (
            <img
              src={tokenIconUrl}
              alt={tokenName || "トークンアイコン"}
              className="contents-page-card__icon"
            />
          ) : (
            <div className="contents-page-card__icon contents-page-card__icon--fallback">
              ◎
            </div>
          )}
        </div>

        <div className="contents-page-card__meta">
          <p className="contents-page-card__title">
            {tokenName || "名称未設定のトークン"}
          </p>

          {contents.productName ? (
            <Tab
              className="contents-page-card__product-name"
              onClick={onProductNameClick}
              disabled={!contents.productId}
            >
              {contents.productName}
            </Tab>
          ) : null}

          {contents.brandName ? (
            <p className="contents-page-card__brand-name">
              {contents.brandName}
            </p>
          ) : null}
        </div>
      </div>

      <TokenReviewAggregateCard
        tokenBlueprintId={contents.tokenBlueprintId}
        productId={contents.productId}
        currentAvatarId={currentAvatarId}
        shareTitle={tokenName || "トークン詳細"}
        shareText={contents.productName || contents.brandName || ""}
        shareUrl={window.location.href}
      />

      {loadingAvatarId ? (
        <p className="contents-page-card__message">
          アバター情報を確認しています...
        </p>
      ) : null}
    </div>
  );
}