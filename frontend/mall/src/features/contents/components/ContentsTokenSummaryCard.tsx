// frontend/mall/src/features/contents/components/ContentsTokenSummaryCard.tsx

import TextLink from "../../../components/ui/textLink";
import TokenReviewAggregateCard from "../../token-commnet/components/TokenReviewAggregateCard";
import type { ContentsSearchParams } from "../../shared/types/contents";

type ContentsTokenSummaryCardProps = {
  contents: ContentsSearchParams;
  tokenName: string;
  tokenIconUrl: string;
  resaleDisabled: boolean;
  resaleLabel: string;
  onProductNameClick: () => void;
  onBrandNameClick: () => void;
  onResaleClick: () => void;
};

export default function ContentsTokenSummaryCard({
  contents,
  tokenName,
  tokenIconUrl,
  resaleDisabled,
  resaleLabel,
  onProductNameClick,
  onBrandNameClick,
  onResaleClick,
}: ContentsTokenSummaryCardProps) {
  const hasProductName = Boolean(contents.productName);
  const hasBrandName = Boolean(contents.brandName);

  return (
    <section className="contents-page-token-summary">
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
                <TextLink
                  onClick={onProductNameClick}
                  disabled={!contents.productId}
                >
                  {contents.productName}
                </TextLink>
              ) : null}

              {hasBrandName ? (
                <TextLink
                  onClick={onBrandNameClick}
                  disabled={!contents.brandId}
                >
                  {contents.brandName}
                </TextLink>
              ) : null}
            </div>
          ) : null}
        </div>
      </div>

      <TokenReviewAggregateCard
        tokenBlueprintId={contents.tokenBlueprintId}
        productId={contents.productId}
        resaleDisabled={resaleDisabled}
        resaleLabel={resaleLabel}
        onResaleClick={onResaleClick}
      />
    </section>
  );
}