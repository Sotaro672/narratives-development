// frontend/amol/src/pages/CatalogPage.tsx
import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { onAuthStateChanged, type User } from "firebase/auth";

import "../styles/catalog-page.css";

import Layout from "../components/layout/Layout";
import FooterNav from "../components/layout/FooterNav";
import CatalogImageGallery from "../features/catalog/presentation/components/CatalogImageGallery";
import MeasurementTable from "../features/catalog/presentation/components/MeasurementTable";
import ModelSelector from "../features/catalog/presentation/components/ModelSelector";
import ProductInfoCard from "../features/catalog/presentation/components/ProductInfoCard";
import ReviewSection from "../features/catalog/presentation/components/ReviewSection";
import TokenInfoCard from "../features/catalog/presentation/components/TokenInfoCard";
import { useCatalogPage } from "../features/catalog/presentation/hooks/useCatalogPage";
import { formatPrice } from "../features/catalog/utils/format";
import { auth } from "../lib/firebase";

export default function CatalogPage() {
  const navigate = useNavigate();
  const [user, setUser] = useState<User | null | undefined>(undefined);

  useEffect(() => {
    const unsubscribe = onAuthStateChanged(auth, (nextUser) => {
      setUser(nextUser);
    });

    return unsubscribe;
  }, []);

  const isLoggedIn = Boolean(user);

  const {
    catalog,
    catalogKind,
    isAlcoholCatalog,
    isLoadingCatalog,
    isLoadingReviews,
    isAddingToCart,
    errorMessage,
    reviewErrorMessage,
    cartMessage,
    cartErrorMessage,
    activeImage,
    activeImageIndex,
    catalogImages,
    hasMultipleImages,
    firstPrice,
    reviewSummary,
    reviewItems,
    measurementRows,
    measurementKeys,
    shouldShowMeasurementTable,
    colorOptions,
    sizeOptions,
    selectedColorKey,
    selectedSize,
    selectedModel,
    selectedModelPrice,
    selectedModelStock,
    canAddToCart,
    isMobilePortrait,
    setActiveImageIndex,
    handlePrevImage,
    handleNextImage,
    handleImageTouchStart,
    handleImageTouchEnd,
    handleSelectColor,
    handleSelectSize,
    handleBrandClick,
    handleAddToCart,
    handleCartButtonClick,
  } = useCatalogPage();

  const handleBackButtonClick = () => {
    if (isLoggedIn) {
      navigate("/lists");
      return;
    }

    navigate(-1);
  };

  const handleAvatarClick = (avatarId: string) => {
    if (!avatarId) {
      return;
    }

    navigate(`/avatars/${encodeURIComponent(avatarId)}`);
  };

  return (
    <Layout
      title={
        catalog?.productBlueprint.productName ||
        (isLoadingCatalog ? "" : "カタログ詳細")
      }
      mode={isLoggedIn ? "mypage" : "landing"}
      showBackButton
      backTo="/lists"
      onBackButtonClick={handleBackButtonClick}
      showFooter={false}
      showHeader
      hideSettingsButton
      showCartButton={isLoggedIn}
      cartButtonLabel="カート"
      onCartButtonClick={isLoggedIn ? handleCartButtonClick : undefined}
      actionButtonLabel={
        !isLoggedIn || isMobilePortrait
          ? undefined
          : isAddingToCart
            ? "追加中"
            : "カートに入れる"
      }
      onActionButtonClick={
        !isLoggedIn || isMobilePortrait ? undefined : handleAddToCart
      }
      actionButtonDisabled={!isLoggedIn || !canAddToCart}
    >
      <section className="split-page catalog-page-section">
        {isLoadingCatalog ? (
          <p className="catalog-page-state">カタログ詳細を読み込んでいます。</p>
        ) : null}

        {!isLoadingCatalog && errorMessage ? (
          <p className="catalog-page-error" role="alert">
            {errorMessage}
          </p>
        ) : null}

        {!isLoadingCatalog && !errorMessage && catalog ? (
          <div className="split-page-content catalog-page-content">
            <div className="split-page-left catalog-page-media">
              <CatalogImageGallery
                activeImage={activeImage}
                activeImageIndex={activeImageIndex}
                catalogImages={catalogImages}
                productBlueprint={catalog.productBlueprint}
                hasMultipleImages={hasMultipleImages}
                onPrevImage={handlePrevImage}
                onNextImage={handleNextImage}
                onSelectImage={setActiveImageIndex}
                onTouchStart={handleImageTouchStart}
                onTouchEnd={handleImageTouchEnd}
              />
            </div>

            <div className="split-page-right catalog-page-detail">
              <div className="catalog-page-summary">
                <h1 className="catalog-page-title">{catalog.list.title}</h1>

                {catalog.list.description ? (
                  <p className="catalog-page-description">
                    {catalog.list.description}
                  </p>
                ) : null}

                <p className="catalog-page-price">
                  {firstPrice ? formatPrice(firstPrice.price) : "価格未設定"}
                </p>
              </div>

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
                colorOptions={colorOptions}
                sizeOptions={sizeOptions}
                selectedColorKey={selectedColorKey}
                selectedSize={selectedSize}
                selectedModel={selectedModel}
                selectedModelPrice={selectedModelPrice}
                selectedModelStock={selectedModelStock}
                cartMessage={isLoggedIn ? cartMessage : ""}
                cartErrorMessage={isLoggedIn ? cartErrorMessage : ""}
                isAlcoholCatalog={isAlcoholCatalog}
                onSelectColor={handleSelectColor}
                onSelectSize={handleSelectSize}
              />

              <TokenInfoCard tokenBlueprint={catalog.tokenBlueprint} />

              <ReviewSection
                reviewSummary={reviewSummary}
                reviewItems={reviewItems}
                isLoadingReviews={isLoadingReviews}
                reviewErrorMessage={reviewErrorMessage}
                onAvatarClick={handleAvatarClick}
              />
            </div>
          </div>
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