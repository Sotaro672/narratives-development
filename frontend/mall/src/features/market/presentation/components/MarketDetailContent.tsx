// frontend/mall/src/features/market/presentation/components/MarketDetailContent.tsx

import { useEffect, useState } from "react";

import Alert from "../../../../components/ui/Alert";
import Button from "../../../../components/ui/Button";
import Card from "../../../../components/ui/Card";
import TextState from "../../../../components/ui/TextState";
import { getMyAvatar } from "../../../avatar/api/avatarApi";
import { useAuthState } from "../../../shared/hooks/useAuthState";
import AvatarSummaryCard from "../../../shared/presentation/components/AvatarSummaryCard";
import FavoriteHeartButton from "../../../shared/presentation/components/FavoriteHeartButton";
import ProductDescription from "../../../shared/presentation/components/ProductDescription";
import ProductDetailLayout from "../../../shared/presentation/components/ProductDetailLayout";
import ProductIdentity from "../../../shared/presentation/components/ProductIdentity";
import ProductMediaGallery from "../../../shared/presentation/components/ProductMediaGallery";
import ProductModelMeta from "../../../shared/presentation/components/ProductModelMeta";
import ProductPrice from "../../../shared/presentation/components/ProductPrice";
import ProductReviewSection from "../../../shared/presentation/components/ProductReviewSection";
import ReportFlagButton from "../../../shared/presentation/components/ReportFlagButton";
import ResaleCommentButton from "../../../shared/presentation/components/ResaleCommentButton";
import TokenSummaryCard from "../../../shared/presentation/components/TokenSummaryCard";
import type { UseMarketDetailPageResult } from "../hooks/useMarketDetailPage";

import "../../../shared/styles/product-detail.css";

type MarketDetailContentState = Pick<
  UseMarketDetailPageResult,
  | "item"
  | "reviews"
  | "commentCount"
  | "isLiked"
  | "loading"
  | "loadingLike"
  | "loadingReviews"
  | "addingToCart"
  | "updatingLike"
  | "error"
  | "reviewsError"
  | "likeErrorMessage"
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
  | "canAddToCart"
  | "handlePrevMedia"
  | "handleNextMedia"
  | "handleSelectMedia"
  | "handleToggleLike"
>;

type MarketDetailContentProps = {
  detail: MarketDetailContentState;
  onOpenSeller: () => void;
  onOpenResaleChat: () => void;
  onOpenResaleReport: () => void;
  onAddToCart: () => void | Promise<void>;
  reportSubmitting: boolean;
};

export default function MarketDetailContent({
  detail,
  onOpenSeller,
  onOpenResaleChat,
  onOpenResaleReport,
  onAddToCart,
  reportSubmitting,
}: MarketDetailContentProps) {
  const { authResolved, isLoggedIn } = useAuthState();
  const [currentAvatarId, setCurrentAvatarId] = useState("");

  const {
    item,
    reviews,
    commentCount,
    isLiked,
    loading,
    loadingLike,
    loadingReviews,
    addingToCart,
    updatingLike,
    error,
    reviewsError,
    likeErrorMessage,
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
    canAddToCart,
    handlePrevMedia,
    handleNextMedia,
    handleSelectMedia,
    handleToggleLike,
  } = detail;

  useEffect(() => {
    let cancelled = false;

    async function loadCurrentAvatar() {
      if (!authResolved || !isLoggedIn) {
        setCurrentAvatarId("");
        return;
      }

      try {
        const avatar = await getMyAvatar();

        if (!cancelled) {
          setCurrentAvatarId(avatar?.avatarId?.trim() ?? "");
        }
      } catch {
        if (!cancelled) {
          setCurrentAvatarId("");
        }
      }
    }

    void loadCurrentAvatar();

    return () => {
      cancelled = true;
    };
  }, [authResolved, isLoggedIn]);

  const normalizedResaleId = item?.id?.trim() ?? "";
  const normalizedSellerAvatarId = sellerAvatarId.trim();
  const canReportResale =
    authResolved &&
    isLoggedIn &&
    Boolean(currentAvatarId) &&
    Boolean(normalizedResaleId) &&
    Boolean(normalizedSellerAvatarId) &&
    normalizedSellerAvatarId !== currentAvatarId &&
    !reportSubmitting;

  return (
    <div className="page-layout market-detail-page">
      {loading ? (
        <Card padding="lg" className="market-detail-page__state-card">
          <TextState variant="loading" className="market-detail-page__state-text">
            読み込み中です...
          </TextState>
        </Card>
      ) : null}

      {!loading && error ? <Alert variant="error">{error}</Alert> : null}

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
          mediaAfter={
            <div className="market-detail-page__media-actions-area">
              <div className="market-detail-page__media-actions">
                <FavoriteHeartButton
                  isLiked={isLiked}
                  disabled={loadingLike || updatingLike}
                  onClick={handleToggleLike}
                />

                <ResaleCommentButton
                  commentCount={commentCount}
                  onClick={onOpenResaleChat}
                />

                <ReportFlagButton
                  disabled={!canReportResale}
                  onClick={onOpenResaleReport}
                />
              </div>

              {likeErrorMessage ? (
                <TextState variant="error" className="market-detail-page__like-error">
                  {likeErrorMessage}
                </TextState>
              ) : null}
            </div>
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
            </>
          }
          contentFooter={
            <>
              <Button
                variant="primary"
                size="lg"
                fullWidth
                disabled={!canAddToCart}
                onClick={onAddToCart}
              >
                {addingToCart ? "追加中" : "カートに入れる"}
              </Button>

              <ProductReviewSection
                items={reviews?.items ?? []}
                productBlueprintId={item.productBlueprintId}
                currentAvatarId={isLoggedIn ? currentAvatarId : ""}
                loading={loadingReviews}
                errorMessage={reviewsError}
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

          <ProductModelMeta
            conditionLabel={item.condition}
            model={model}
          />

          <ProductDescription
            description={item.description}
            className="product-detail__description--standalone"
          />

          {cartMessage ? (
            <Alert variant="success" className="market-detail-page__cart-message">
              {cartMessage}
            </Alert>
          ) : null}

          {cartErrorMessage ? (
            <Alert variant="error" className="market-detail-page__cart-error">
              {cartErrorMessage}
            </Alert>
          ) : null}
        </ProductDetailLayout>
      ) : null}
    </div>
  );
}