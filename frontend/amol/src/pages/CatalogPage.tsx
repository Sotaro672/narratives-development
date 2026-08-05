// frontend/amol/src/pages/CatalogPage.tsx

import {
  useNavigate,
} from "react-router-dom";

import Layout from "../components/layout/Layout";
import FooterNav from "../components/layout/FooterNav";

import CatalogSummary from "../features/catalog/presentation/components/CatalogSummary";
import CatalogImageGallery from "../features/catalog/presentation/components/CatalogImageGallery";
import MeasurementTable from "../features/catalog/presentation/components/MeasurementTable";
import ModelSelector from "../features/catalog/presentation/components/ModelSelector";
import ProductInfoCard from "../features/catalog/presentation/components/ProductInfoCard";
import ReviewSection from "../features/catalog/presentation/components/ReviewSection";
import TokenInfoCard from "../features/catalog/presentation/components/TokenInfoCard";

import {
  useCatalogPage,
} from "../features/catalog/presentation/hooks/useCatalogPage";

import {
  useAuthState,
} from "../features/shared/hooks/useAuthState";

import "../features/catalog/presentation/styles/catalog-page.css";

export default function CatalogPage() {
  const navigate = useNavigate();

  const {
    isLoggedIn,
  } = useAuthState();

  const {
    catalog,
    catalogKind,
    isAlcoholCatalog,
    isLoadingCatalog,
    isAddingToCart,
    errorMessage,
    reviewErrorMessage,
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
    handleAvatarClick,
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

  return (
    <Layout
      title={
        catalog?.productBlueprint
          .productName ||
        (
          isLoadingCatalog
            ? ""
            : "カタログ詳細"
        )
      }
      titleClickable={false}
      mode={
        isLoggedIn
          ? "mypage"
          : "landing"
      }
      showBackButton
      backTo="/lists"
      onBackButtonClick={
        handleBackButtonClick
      }
      showFooter={false}
      showHeader
      hideSettingsButton
      showCartButton={isLoggedIn}
      cartButtonLabel="カート"
      onCartButtonClick={
        isLoggedIn
          ? handleCartButtonClick
          : undefined
      }
      actionButtonLabel={
        !isLoggedIn ||
        isMobilePortrait
          ? undefined
          : isAddingToCart
            ? "追加中"
            : "カートに入れる"
      }
      onActionButtonClick={
        !isLoggedIn ||
        isMobilePortrait
          ? undefined
          : handleAddToCart
      }
      actionButtonDisabled={
        !isLoggedIn ||
        !canAddToCart
      }
    >
      <section className="split-page catalog-page-section">
        {isLoadingCatalog ? (
          <p className="catalog-page-state">
            カタログ詳細を読み込んでいます。
          </p>
        ) : null}

        {!isLoadingCatalog &&
        errorMessage ? (
          <p
            className="catalog-page-error"
            role="alert"
          >
            {errorMessage}
          </p>
        ) : null}

        {!isLoadingCatalog &&
        !errorMessage &&
        catalog ? (
          <div className="split-page-content catalog-page-content">
            <div className="split-page-left catalog-page-media">
              <CatalogImageGallery
                activeImage={
                  activeImage
                }
                activeImageIndex={
                  activeImageIndex
                }
                catalogImages={
                  catalogImages
                }
                productBlueprint={
                  catalog.productBlueprint
                }
                hasMultipleImages={
                  hasMultipleImages
                }
                onPrevImage={
                  handlePrevImage
                }
                onNextImage={
                  handleNextImage
                }
                onSelectImage={
                  setActiveImageIndex
                }
                onTouchStart={
                  handleImageTouchStart
                }
                onTouchEnd={
                  handleImageTouchEnd
                }
              />
            </div>

            <div className="split-page-right catalog-page-detail">
              <CatalogSummary
                title={
                  catalog.list.title
                }
                description={
                  catalog.list.description
                }
                price={
                  firstPrice?.price
                }
              />

              <ProductInfoCard
                productBlueprint={
                  catalog.productBlueprint
                }
                categoryKind={
                  catalogKind
                }
                onBrandClick={
                  handleBrandClick
                }
              />

              {shouldShowMeasurementTable ? (
                <MeasurementTable
                  measurementRows={
                    measurementRows
                  }
                  measurementKeys={
                    measurementKeys
                  }
                />
              ) : null}

              <ModelSelector
                colorOptions={
                  colorOptions
                }
                sizeOptions={
                  sizeOptions
                }
                selectedColorKey={
                  selectedColorKey
                }
                selectedSize={
                  selectedSize
                }
                selectedModel={
                  selectedModel
                }
                selectedModelPrice={
                  selectedModelPrice
                }
                selectedModelStock={
                  selectedModelStock
                }
                cartErrorMessage={
                  isLoggedIn
                    ? cartErrorMessage
                    : ""
                }
                isAlcoholCatalog={
                  isAlcoholCatalog
                }
                onSelectColor={
                  handleSelectColor
                }
                onSelectSize={
                  handleSelectSize
                }
              />

              <TokenInfoCard
                tokenBlueprint={
                  catalog.tokenBlueprint
                }
              />

              <ReviewSection
                reviewSummary={
                  reviewSummary
                }
                reviewItems={
                  reviewItems
                }
                reviewErrorMessage={
                  reviewErrorMessage
                }
                onAvatarClick={
                  handleAvatarClick
                }
              />
            </div>
          </div>
        ) : null}
      </section>

      {isLoggedIn &&
      isMobilePortrait ? (
        <FooterNav
          variant="action"
          buttonLabel={
            isAddingToCart
              ? "追加中"
              : "カートに入れる"
          }
          disabled={
            !canAddToCart
          }
          onButtonClick={
            handleAddToCart
          }
        />
      ) : null}
    </Layout>
  );
}