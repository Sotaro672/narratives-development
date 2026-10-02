// frontend/mall/src/pages/ResaleCreatePage.tsx

import { useState } from "react";
import { useNavigate } from "react-router-dom";

import { useMobilePortrait } from "../components/hooks/useMobilePortrait";
import Layout from "../components/layout/Layout";
import MobileComposerFooter from "../components/layout/MobileComposerFooter";
import MobileSwipeDismissPage from "../components/layout/MobileSwipeDismissPage";
import Button from "../components/ui/Button";

import ResaleConditionMediaField from "../features/resale/presentation/components/ResaleConditionMediaField";
import ResaleCreateForm, {
  RESALE_DESCRIPTION_MAX_LENGTH,
} from "../features/resale/presentation/components/ResaleCreateForm";
import ResaleCreateMissingTarget from "../features/resale/presentation/components/ResaleCreateMissingTarget";
import ResaleCreateProgressModal from "../features/resale/presentation/components/ResaleCreateProgressModal";
import { useResaleCreatePage } from "../features/resale/presentation/hooks/useResaleCreatePage";
import ProductDetailLayout from "../features/shared/presentation/components/ProductDetailLayout";
import ProductIdentity from "../features/shared/presentation/components/ProductIdentity";
import TokenSummaryCard from "../features/shared/presentation/components/TokenSummaryCard";

import "../styles/page-layout.css";
import "../styles/resale-page.css";
import "../features/shared/styles/product-detail.css";

type MobileEditor = "price" | "description" | null;

export default function ResaleCreatePage() {
  const navigate = useNavigate();
  const isMobilePortrait = useMobilePortrait();
  const [mobileEditor, setMobileEditor] = useState<MobileEditor>(null);

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

  const isMobileComposerOpen =
    isMobilePortrait && mobileEditor !== null;

  const showMobileActionFooter =
    isMobilePortrait &&
    hasRequiredListingTarget &&
    !isMobileComposerOpen;

  const handleDismiss = () => {
    navigate(-1);
  };

  const handleCloseMobileEditor = () => {
    setMobileEditor(null);
  };

  const content = (
    <Layout
      title="AMOL"
      mode="mypage"
      showHeader={!isMobilePortrait}
      showFooter={isMobilePortrait ? showMobileActionFooter : true}
      mainClassName={
        isMobilePortrait
          ? [
              "resale-create-page-main--mobile",
              isMobileComposerOpen
                ? "resale-create-page-main--with-composer"
                : "",
            ]
              .filter(Boolean)
              .join(" ")
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
              mobileComposerEnabled={isMobilePortrait}
              onPriceFocus={() => setMobileEditor("price")}
              onDescriptionFocus={() => setMobileEditor("description")}
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
          dismissGestureEnabled={!isUploading && !isMobileComposerOpen}
          dismissDisabled={isUploading}
          onDismiss={handleDismiss}
        >
          {content}
        </MobileSwipeDismissPage>
      ) : (
        content
      )}

      {isMobileComposerOpen ? (
        <MobileComposerFooter
          key={mobileEditor}
          content={
            mobileEditor === "price"
              ? formattedPrice
              : description
          }
          placeholder={
            mobileEditor === "price"
              ? "販売価格を入力"
              : "購入時期、着用回数、保管状態などを入力"
          }
          inputMode={
            mobileEditor === "price"
              ? "numeric"
              : "text"
          }
          autoFocus
          maxLength={
            mobileEditor === "description"
              ? RESALE_DESCRIPTION_MAX_LENGTH
              : null
          }
          disabled={isSubmitting}
          canSubmit
          submitLabel="完了"
          beforeInput={
            <span>
              {mobileEditor === "price"
                ? "販売価格"
                : `説明文 ${description.length}/${RESALE_DESCRIPTION_MAX_LENGTH}`}
            </span>
          }
          onContentChange={
            mobileEditor === "price"
              ? handlePriceChange
              : handleDescriptionChange
          }
          onSubmit={handleCloseMobileEditor}
        />
      ) : null}

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