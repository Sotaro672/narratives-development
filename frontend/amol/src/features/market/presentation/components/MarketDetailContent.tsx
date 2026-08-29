// frontend/amol/src/features/market/presentation/components/MarketDetailContent.tsx

import type { UseMarketDetailPageResult } from "../hooks/useMarketDetailPage";

import ProductMediaGallery from "../../../shared/presentation/components/ProductMediaGallery";
import ProductDetailLayout from "../../../shared/presentation/components/ProductDetailLayout";
import ProductModelMeta from "../../../shared/presentation/components/ProductModelMeta";
import ProductIdentity from "../../../shared/presentation/components/ProductIdentity";
import TokenSummaryCard from "../../../shared/presentation/components/TokenSummaryCard";

import MarketReviewSection from "./MarketReviewSection";
import MarketSellerCard from "./MarketSellerCard";

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
};

export default function MarketDetailContent({
  detail,
  onOpenSeller,
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
              />

              <MarketSellerCard
                avatarId={sellerAvatarId}
                avatarName={avatarName}
                avatarIcon={avatarIcon}
                onOpen={onOpenSeller}
              />
            </>
          }
        >
          <ProductIdentity
            brandName={item.brandName}
            productName={item.productName}
            tokenName={item.tokenName}
          />

          <p className="product-detail__price">{priceLabel}</p>

          <ProductModelMeta
            conditionLabel={item.condition}
            model={model}
          />

          {item.description ? (
            <div className="product-detail__description">
              <h2>商品説明</h2>
              <p>{item.description}</p>
            </div>
          ) : null}

          <MarketReviewSection
            reviews={reviews}
            loading={loadingReviews}
            error={reviewsError}
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