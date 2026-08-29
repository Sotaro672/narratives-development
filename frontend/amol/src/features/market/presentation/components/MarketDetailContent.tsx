// frontend/amol/src/features/market/presentation/components/MarketDetailContent.tsx

import type {
  UseMarketDetailPageResult,
} from "../hooks/useMarketDetailPage";

import ResaleConditionGallery from "../../../shared/presentation/components/ProductMediaGallery";
import ResaleDetailLayout from "../../../shared/presentation/components/ProductDetailLayout";
import ResaleModelMeta from "../../../shared/presentation/components/ProductModelMeta";
import ResaleProductIdentity from "../../../shared/presentation/components/ProductIdentity";
import ResaleTokenCard from "../../../shared/presentation/components/TokenSummaryCard";

import MarketReviewSection from "./MarketReviewSection";
import MarketSellerCard from "./MarketSellerCard";

import "../../../shared/styles/resale-product-detail.css";

type MarketDetailContentState =
  Pick<
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
        <ResaleDetailLayout
          media={
            <ResaleConditionGallery
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
              <ResaleTokenCard
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
          <ResaleProductIdentity
            brandName={item.brandName}
            productName={item.productName}
            tokenName={item.tokenName}
          />

          <p className="resale-product-detail__price">
            {priceLabel}
          </p>

          <ResaleModelMeta
            conditionLabel={item.condition}
            model={model}
          />

          {item.description ? (
            <div className="resale-product-detail__description">
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
            <p className="market-detail-page__cart-message">
              {cartMessage}
            </p>
          ) : null}

          {cartErrorMessage ? (
            <p
              className="market-detail-page__cart-error"
              role="alert"
            >
              {cartErrorMessage}
            </p>
          ) : null}
        </ResaleDetailLayout>
      ) : null}
    </div>
  );
}