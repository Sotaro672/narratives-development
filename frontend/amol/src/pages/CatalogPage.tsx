// frontend/src/pages/CatalogPage.tsx
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

export default function CatalogPage() {
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
    handleAddToCart,
    handleCartButtonClick,
  } = useCatalogPage();

  return (
    <Layout
      title={
        catalog?.productBlueprint.productName ||
        (isLoadingCatalog ? "" : "カタログ詳細")
      }
      mode="mypage"
      showBackButton
      backTo="/lists"
      showFooter={false}
      showHeader
      hideSettingsButton
      showCartButton
      cartButtonLabel="カート"
      onCartButtonClick={handleCartButtonClick}
      actionButtonLabel={
        isMobilePortrait
          ? undefined
          : isAddingToCart
            ? "追加中"
            : "カートに入れる"
      }
      onActionButtonClick={isMobilePortrait ? undefined : handleAddToCart}
      actionButtonDisabled={!canAddToCart}
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
                cartMessage={cartMessage}
                cartErrorMessage={cartErrorMessage}
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
              />
            </div>
          </div>
        ) : null}
      </section>

      {isMobilePortrait ? (
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