// frontend/mall/src/pages/ResaleCreatePage.tsx

import { useMobilePortrait } from "../components/hooks/useMobilePortrait";
import Layout from "../components/layout/Layout";
import MobileSwipeDismissPage from "../components/layout/MobileSwipeDismissPage";
import Button from "../components/ui/Button";

import ResaleConditionMediaField from "../features/resale/presentation/components/ResaleConditionMediaField";
import ResaleCreateForm from "../features/resale/presentation/components/ResaleCreateForm";
import ResaleCreateMissingTarget from "../features/resale/presentation/components/ResaleCreateMissingTarget";
import ResaleCreateProgressModal from "../features/resale/presentation/components/ResaleCreateProgressModal";
import { useResaleCreatePage } from "../features/resale/presentation/hooks/useResaleCreatePage";
import ProductDetailLayout from "../features/shared/presentation/components/ProductDetailLayout";
import ProductIdentity from "../features/shared/presentation/components/ProductIdentity";
import TokenSummaryCard from "../features/shared/presentation/components/TokenSummaryCard";

import "../styles/page-layout.css";
import "../styles/resale-page.css";
import "../features/shared/styles/product-detail.css";

export default function ResaleCreatePage() {
  const isMobilePortrait = useMobilePortrait();

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
    isUploading,
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

  const showMobileActionFooter =
    isMobilePortrait && hasRequiredListingTarget;

  const content = (
    <Layout
      title="AMOL"
      mode="mypage"
      showHeader={!isMobilePortrait}
      showFooter={isMobilePortrait ? showMobileActionFooter : true}
      mainClassName={
        isMobilePortrait
          ? "resale-create-page-main--mobile"
          : undefined
      }
      footerProps={
        showMobileActionFooter
          ? {
              variant: "action",
              buttonLabel: submitButtonLabel,
              buttonWidth: "content",
              disabled: !canSubmit || isSubmitting,
              onButtonClick: handleSubmit,
            }
          : undefined
      }
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

            {!isMobilePortrait ? (
              <div className="page-actions">
                <Button
                  type="button"
                  variant="primary"
                  size="lg"
                  fullWidth
                  disabled={!canSubmit || isSubmitting}
                  onClick={() => void handleSubmit()}
                >
                  {submitButtonLabel}
                </Button>
              </div>
            ) : null}
          </ProductDetailLayout>
        )}
      </section>
    </Layout>
  );

  return (
    <>
      {isMobilePortrait ? (
        <MobileSwipeDismissPage
          dismissButtonAriaLabel="出品画面を閉じる"
          dismissGestureEnabled={!isUploading}
          dismissDisabled={isUploading}
          onDismiss={handleBackToWallet}
        >
          {content}
        </MobileSwipeDismissPage>
      ) : (
        content
      )}

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