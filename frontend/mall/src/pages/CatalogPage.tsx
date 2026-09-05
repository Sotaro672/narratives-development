// frontend/mall/src/pages/CatalogPage.tsx

import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

import FooterNav from "../components/layout/FooterNav";
import Layout from "../components/layout/Layout";
import { formatPrice } from "../components/utils/price";

import { getMyAvatar } from "../features/avatar/api/avatarApi";
import MeasurementTable from "../features/catalog/presentation/components/MeasurementTable";
import ModelSelector from "../features/catalog/presentation/components/ModelSelector";
import ProductInfoCard from "../features/catalog/presentation/components/ProductInfoCard";
import { useCatalogPage } from "../features/catalog/presentation/hooks/useCatalogPage";

import { useAuthState } from "../features/shared/hooks/useAuthState";
import FavoriteHeartButton from "../features/shared/presentation/components/FavoriteHeartButton";
import ProductDescription from "../features/shared/presentation/components/ProductDescription";
import ProductDetailLayout from "../features/shared/presentation/components/ProductDetailLayout";
import ProductIdentity from "../features/shared/presentation/components/ProductIdentity";
import ProductMediaGallery from "../features/shared/presentation/components/ProductMediaGallery";
import ProductPrice from "../features/shared/presentation/components/ProductPrice";
import ProductReviewSection from "../features/shared/presentation/components/ProductReviewSection";
import TokenSummaryCard from "../features/shared/presentation/components/TokenSummaryCard";

import "../features/shared/styles/product-detail.css";
import "../styles/catalog-page.css";

export default function CatalogPage() {
  const navigate = useNavigate();
  const { authResolved, isLoggedIn } = useAuthState();
  const [currentAvatarId, setCurrentAvatarId] = useState("");

  const {
    catalog,
    catalogKind,
    isAlcoholCatalog,
    isLoadingCatalog,
    isAddingToCart,
    isLiked,
    isLoadingLike,
    isUpdatingLike,
    errorMessage,
    reviewErrorMessage,
    cartErrorMessage,
    likeErrorMessage,
    activeImageIndex,
    galleryItems,
    firstPrice,
    reviewSummary,
    reviewItems,
    measurementRows,
    measurementKeys,
    shouldShowMeasurementTable,
    alcoholOptions,
    colorOptions,
    sizeOptions,
    selectedColorKey,
    selectedSize,
    selectedModelId,
    selectedModel,
    selectedModelPrice,
    selectedModelStock,
    canAddToCart,
    isMobilePortrait,
    handlePrevImage,
    handleNextImage,
    handleSelectImage,
    handleSelectColor,
    handleSelectSize,
    handleSelectModel,
    handleBrandClick,
    handleAvatarClick,
    handleToggleLike,
    handleAddToCart,
  } = useCatalogPage();

  useEffect(() => {
    let cancelled = false;

    async function loadCurrentAvatar() {
      if (!authResolved || !isLoggedIn) {
        setCurrentAvatarId("");
        return;
      }

      try {
        const avatar = await getMyAvatar();

        if (cancelled) {
          return;
        }

        setCurrentAvatarId(avatar?.avatarId?.trim() ?? "");
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

  const handleBackButtonClick = () => {
    if (isLoggedIn) {
      navigate("/lists");
      return;
    }

    navigate(-1);
  };

  return (
    <Layout
      title={
        catalog?.productBlueprint.productName ||
        (isLoadingCatalog ? "" : "カタログ詳細")
      }
      titleClickable={false}
      mode={isLoggedIn ? "mypage" : "landing"}
      showBackButton
      backTo="/lists"
      onBackButtonClick={handleBackButtonClick}
      showFooter={false}
      showHeader
      hideSettingsButton
      showCartButton={isLoggedIn}
      cartButtonLabel="カート"
      onCartButtonClick={isLoggedIn ? () => navigate("/cart") : undefined}
      actionButtonLabel={
        !isLoggedIn || isMobilePortrait
          ? undefined
          : isAddingToCart
            ? "追加中"
            : "カートに入れる"
      }
      onActionButtonClick={
        !isLoggedIn || isMobilePortrait
          ? undefined
          : handleAddToCart
      }
      actionButtonDisabled={!isLoggedIn || !canAddToCart}
    >
      <section className="catalog-page-section">
        {isLoadingCatalog ? (
          <p className="catalog-page-state">
            カタログ詳細を読み込んでいます。
          </p>
        ) : null}

        {!isLoadingCatalog && errorMessage ? (
          <p className="catalog-page-error" role="alert">
            {errorMessage}
          </p>
        ) : null}

        {!isLoadingCatalog && !errorMessage && catalog ? (
          <ProductDetailLayout
            media={
              <ProductMediaGallery
                items={galleryItems}
                activeIndex={activeImageIndex}
                altFallback={
                  catalog.productBlueprint.productName ||
                  catalog.list.title ||
                  "商品画像"
                }
                placeholderText="No Image"
                onPrev={handlePrevImage}
                onNext={handleNextImage}
                onSelect={handleSelectImage}
              />
            }
            mediaFooter={
              <TokenSummaryCard
                brandName={catalog.productBlueprint.brandName}
                tokenName={catalog.tokenBlueprint.tokenName}
                tokenIcon={catalog.tokenBlueprint.tokenIcon}
                symbol={catalog.tokenBlueprint.symbol}
                description={catalog.tokenBlueprint.description}
              />
            }
            mediaColumnClassName="catalog-page-media"
          >
            <ProductIdentity
              brandName={catalog.productBlueprint.brandName}
              productName={catalog.list.title}
              tokenName={catalog.tokenBlueprint.tokenName}
            />

            <ProductDescription description={catalog.list.description} />
            <ProductPrice priceLabel={formatPrice(firstPrice?.price)} />

            {isLoggedIn ? (
              <>
                <FavoriteHeartButton
                  isLiked={isLiked}
                  disabled={isLoadingLike || isUpdatingLike}
                  onClick={handleToggleLike}
                />

                {likeErrorMessage ? (
                  <p className="catalog-page-error" role="alert">
                    {likeErrorMessage}
                  </p>
                ) : null}
              </>
            ) : null}

            <ProductInfoCard
              productBlueprint={catalog.productBlueprint}
              categoryKind={catalogKind}
              onBrandClick={handleBrandClick}
            />

            {shouldShowMeasurementTable ? (
              <MeasurementTable
                measurementRows={measurementRows}
                measurementKeys={measurementKeys}
              />
            ) : null}

            <ModelSelector
              alcoholOptions={alcoholOptions}
              colorOptions={colorOptions}
              sizeOptions={sizeOptions}
              selectedColorKey={selectedColorKey}
              selectedSize={selectedSize}
              selectedModelId={selectedModelId}
              selectedModel={selectedModel}
              selectedModelPrice={selectedModelPrice}
              selectedModelStock={selectedModelStock}
              cartErrorMessage={isLoggedIn ? cartErrorMessage : ""}
              isAlcoholCatalog={isAlcoholCatalog}
              onSelectColor={handleSelectColor}
              onSelectSize={handleSelectSize}
              onSelectModel={handleSelectModel}
            />

            <ProductReviewSection
              items={reviewItems}
              productBlueprintId={catalog.productBlueprint.id}
              currentAvatarId={isLoggedIn ? currentAvatarId : ""}
              averageRating={reviewSummary?.averageRating}
              totalCount={reviewSummary?.totalCount}
              errorMessage={reviewErrorMessage}
              showHelpfulVotes
              onAvatarClick={handleAvatarClick}
            />
          </ProductDetailLayout>
        ) : null}
      </section>

      {isLoggedIn && isMobilePortrait ? (
        <FooterNav
          variant="action"
          buttonLabel={isAddingToCart ? "追加中" : "カートに入れる"}
          disabled={!canAddToCart}
          onButtonClick={handleAddToCart}
        />
      ) : null}
    </Layout>
  );
}