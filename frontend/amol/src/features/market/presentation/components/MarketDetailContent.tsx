// frontend/amol/src/features/market/presentation/components/MarketDetailContent.tsx

import type {
  UseMarketDetailPageResult,
} from "../hooks/useMarketDetailPage";

import MarketProductMeta from "./MarketProductMeta";
import MarketResaleGallery from "./MarketResaleGallery";
import MarketReviewSection from "./MarketReviewSection";
import MarketSellerCard from "./MarketSellerCard";
import MarketTokenSummary from "./MarketTokenSummary";

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
    | "modelKind"
    | "modelKindLabel"
    | "modelNumber"
    | "modelSize"
    | "modelColorName"
    | "modelColorCssValue"
    | "hasColorInfo"
    | "modelVolumeLabel"
    | "measurementsLabel"
    | "hasModelInfo"
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
    modelKind,
    modelKindLabel,
    modelNumber,
    modelSize,
    modelColorName,
    modelColorCssValue,
    hasColorInfo,
    modelVolumeLabel,
    measurementsLabel,
    hasModelInfo,
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
        <section className="market-detail-page__card">
          <div className="market-detail-page__media-column">
            <MarketResaleGallery
              items={galleryItems}
              activeIndex={safeActiveMediaIndex}
              altFallback={
                item.productName ||
                item.tokenName ||
                "出品画像"
              }
              onPrev={handlePrevMedia}
              onNext={handleNextMedia}
              onSelect={handleSelectMedia}
            />

              <MarketTokenSummary
                tokenName={tokenName}
                tokenIcon={tokenIcon}
                brandName={item.brandName || ""}
              />

            <MarketSellerCard
              avatarId={sellerAvatarId}
              avatarName={avatarName}
              avatarIcon={avatarIcon}
              onOpen={onOpenSeller}
            />
          </div>

          <div className="market-detail-page__content">
            <p className="market-detail-page__brand">
              {item.brandName || "ブランド名未設定"}
            </p>

            <h1 className="market-detail-page__title">
              {item.productName ||
                item.tokenName ||
                "商品名未設定"}
            </h1>

            <p className="market-detail-page__price">
              {priceLabel}
            </p>

            <MarketProductMeta
              condition={item.condition}
              hasModelInfo={hasModelInfo}
              modelKind={modelKind}
              modelKindLabel={modelKindLabel}
              modelNumber={modelNumber}
              modelSize={modelSize}
              hasColorInfo={hasColorInfo}
              modelColorName={modelColorName}
              modelColorCssValue={modelColorCssValue}
              measurementsLabel={measurementsLabel}
              modelVolumeLabel={modelVolumeLabel}
            />

            {item.description ? (
              <div className="market-detail-page__description">
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
          </div>
        </section>
      ) : null}
    </div>
  );
}