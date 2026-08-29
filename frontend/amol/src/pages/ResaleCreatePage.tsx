// frontend/amol/src/pages/ResaleCreatePage.tsx

import Layout from "../components/layout/Layout";

import ProductDetailLayout from "../features/shared/presentation/components/ProductDetailLayout";
import ProductIdentity from "../features/shared/presentation/components/ProductIdentity";
import TokenSummaryCard from "../features/shared/presentation/components/TokenSummaryCard";

import ResaleConditionMediaField from "../features/resale/presentation/components/ResaleConditionMediaField";
import ResaleCreateForm from "../features/resale/presentation/components/ResaleCreateForm";
import ResaleCreateMissingTarget from "../features/resale/presentation/components/ResaleCreateMissingTarget";
import ResaleCreateProgressModal from "../features/resale/presentation/components/ResaleCreateProgressModal";
import { useResaleCreatePage } from "../features/resale/presentation/hooks/useResaleCreatePage";

import "../styles/page-layout.css";
import "../styles/resale-page.css";
import "../features/shared/styles/product-detail.css";

export default function ResaleCreatePage() {
  const {
    target,
    formattedPrice,
    condition,
    description,
    conditionMediaItems,
    conditionMediaCurrentIndex,
    conditionMediaInputRef,
    conditionMediaCarouselRef,
    hasRequiredListingTarget,
    canSubmit,
    isSubmitting,
    errorMessage,
    progress,
    progressOpen,
    submitButtonLabel,
    handlePriceChange,
    handleConditionChange,
    handleDescriptionChange,
    handleConditionMediaSelected,
    handleRemoveConditionMedia,
    handleConditionMediaCarouselScroll,
    handleMoveToConditionMediaSlide,
    handleBackToWallet,
    handleCloseProgress,
    handleSubmit,
  } = useResaleCreatePage();

  return (
    <>
      <Layout
        title="出品"
        showBackButton
        mode="mypage"
        actionButtonLabel="出品"
        onActionButtonClick={handleSubmit}
        actionButtonDisabled={!canSubmit || isSubmitting}
        showFooter
        footerProps={{
          variant: "action",
          buttonLabel: submitButtonLabel,
          disabled: !canSubmit || isSubmitting,
          onButtonClick: handleSubmit,
        }}
      >
        <section className="page-section">
          {!hasRequiredListingTarget ? (
            <ResaleCreateMissingTarget onBackToWallet={handleBackToWallet} />
          ) : (
            <ProductDetailLayout
              media={
                <ResaleConditionMediaField
                  items={conditionMediaItems}
                  currentIndex={conditionMediaCurrentIndex}
                  inputRef={conditionMediaInputRef}
                  carouselRef={conditionMediaCarouselRef}
                  disabled={isSubmitting}
                  onFilesSelected={handleConditionMediaSelected}
                  onRemoveItem={handleRemoveConditionMedia}
                  onCarouselScroll={handleConditionMediaCarouselScroll}
                  onMoveToSlide={handleMoveToConditionMediaSlide}
                />
              }
              mediaFooter={
                <TokenSummaryCard
                  brandName={target.brandName}
                  tokenName={target.tokenName}
                  tokenIcon={target.tokenIconUrl}
                  description={target.tokenDescription}
                />
              }
            >
              <ProductIdentity
                brandName={target.brandName}
                productName={target.productName}
                tokenName={target.tokenName}
              />

              <ResaleCreateForm
                formattedPrice={formattedPrice}
                condition={condition}
                description={description}
                disabled={isSubmitting}
                onPriceChange={handlePriceChange}
                onConditionChange={handleConditionChange}
                onDescriptionChange={handleDescriptionChange}
              />

              {errorMessage ? (
                <p className="page-error" role="alert">
                  {errorMessage}
                </p>
              ) : null}
            </ProductDetailLayout>
          )}
        </section>
      </Layout>

      <ResaleCreateProgressModal
        open={progressOpen}
        progress={progress}
        onClose={
          progress.isBlockingNavigation
            ? undefined
            : handleCloseProgress
        }
      />
    </>
  );
}