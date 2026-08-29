// frontend/amol/src/features/market/presentation/components/MarketDetailContent.tsx

import type { UseMarketDetailPageResult } from "../hooks/useMarketDetailPage";

import AvatarSummaryCard from "../../../shared/presentation/components/AvatarSummaryCard";
import ProductDescription from "../../../shared/presentation/components/ProductDescription";
import ProductDetailLayout from "../../../shared/presentation/components/ProductDetailLayout";
import ProductIdentity from "../../../shared/presentation/components/ProductIdentity";
import ProductMediaGallery from "../../../shared/presentation/components/ProductMediaGallery";
import ProductModelMeta from "../../../shared/presentation/components/ProductModelMeta";
import ProductPrice from "../../../shared/presentation/components/ProductPrice";
import ProductReviewSection from "../../../shared/presentation/components/ProductReviewSection";
import TokenSummaryCard from "../../../shared/presentation/components/TokenSummaryCard";

import "../../../shared/styles/product-detail.css";

type MarketDetailContentState = Pick<
  UseMarketDetailPageResult,
  | "item"
  | "reviews"
  | "loading"
  | "loadingReviews"
  | "error"
  | "reviewsError"
  | "cartMessage"
  | "cartErrorMessage"
  | "priceLabel"
  | "model"
  | "tokenName"
  | "tokenIcon"
  | "tokenDescription"
  | "sellerAvatarId"
  | "avatarName"
  | "avatarIcon"
  | "galleryItems"
  | "safeActiveMediaIndex"
  | "handlePrevMedia"
  | "handleNextMedia"
  | "handleSelectMedia"
>;

type MarketDetailContentProps = {
  detail: MarketDetailContentState;
  onOpenSeller: () => void;
  onOpenResaleChat: () => void;
};

export default function MarketDetailContent({
  detail,
  onOpenSeller,
  onOpenResaleChat,
}: MarketDetailContentProps) {
  const {
    item,
    reviews,
    loading,
    loadingReviews,
    error,
    reviewsError,
    cartMessage,
    cartErrorMessage,
    priceLabel,
    model,
    tokenName,
    tokenIcon,
    tokenDescription,
    sellerAvatarId,
    avatarName,
    avatarIcon,
    galleryItems,
    safeActiveMediaIndex,
    handlePrevMedia,
    handleNextMedia,
    handleSelectMedia,
  } = detail;

  return (
    <div className="page-layout market-detail-page">
      {loading ? (
        <div className="market-detail-page__state">
          <p>読み込み中です...</p>
        </div>
      ) : null}

      {!loading && error ? (
        <div className="market-detail-page__state market-detail-page__state--error">
          <p>{error}</p>
        </div>
      ) : null}

      {!loading && !error && item ? (
        <ProductDetailLayout
          className="product-detail__layout--summary-after-content-mobile"
          media={
            <ProductMediaGallery
              items={galleryItems}
              activeIndex={safeActiveMediaIndex}
              altFallback={item.productName || item.tokenName || "出品画像"}
              placeholderText="No Image"
              onPrev={handlePrevMedia}
              onNext={handleNextMedia}
              onSelect={handleSelectMedia}
            />
          }
          mediaFooter={
            <>
              <TokenSummaryCard
                brandName={item.brandName}
                tokenName={tokenName}
                tokenIcon={tokenIcon}
                description={tokenDescription}
              />

              <AvatarSummaryCard
                avatarId={sellerAvatarId}
                avatarName={avatarName}
                avatarIcon={avatarIcon}
                onClick={onOpenSeller}
              />

              <ProductDescription
                description={item.description}
                className="product-detail__description--standalone"
              />
            </>
          }
        >
          <ProductIdentity
            brandName={item.brandName}
            productName={item.productName}
            tokenName={item.tokenName}
          />

          <ProductPrice priceLabel={priceLabel} />

          <ProductModelMeta conditionLabel={item.condition} model={model} />

          <div className="page-actions">
            <button
              type="button"
              className="page-button page-button--secondary"
              onClick={onOpenResaleChat}
            >
              コメントを見る
            </button>
          </div>

          <ProductReviewSection
            items={reviews?.items ?? []}
            loading={loadingReviews}
            errorMessage={reviewsError}
          />

          {cartMessage ? (
            <p className="market-detail-page__cart-message">{cartMessage}</p>
          ) : null}

          {cartErrorMessage ? (
            <p className="market-detail-page__cart-error" role="alert">
              {cartErrorMessage}
            </p>
          ) : null}
        </ProductDetailLayout>
      ) : null}
    </div>
  );
}