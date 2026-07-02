// frontend/amol/src/features/contents/components/ContentsTokenSummaryCard.tsx
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
  onBrandNameClick: () => void;
  onResaleClick: () => void;
};

export default function ContentsTokenSummaryCard({
  contents,
  tokenName,
  tokenIconUrl,
  loadingAvatarId,
  currentAvatarId,
  onProductNameClick,
  onBrandNameClick,
  onResaleClick,
}: ContentsTokenSummaryCardProps) {
  const hasProductName = Boolean(contents.productName);
  const hasBrandName = Boolean(contents.brandName);

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

          {hasProductName || hasBrandName ? (
            <div className="contents-page-card__tag-list">
              {hasProductName ? (
                <Tab
                  className="contents-page-card__product-name"
                  onClick={onProductNameClick}
                  disabled={!contents.productId}
                >
                  {contents.productName}
                </Tab>
              ) : null}

              {hasBrandName ? (
                <Tab
                  className="contents-page-card__brand-name"
                  onClick={onBrandNameClick}
                  disabled={!contents.brandId}
                >
                  {contents.brandName}
                </Tab>
              ) : null}
            </div>
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
        onResaleClick={onResaleClick}
      />

      {loadingAvatarId ? (
        <p className="contents-page-card__message">
          アバター情報を確認しています...
        </p>
      ) : null}
    </div>
  );
}