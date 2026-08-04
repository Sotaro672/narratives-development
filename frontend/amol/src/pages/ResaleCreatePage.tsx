// frontend/amol/src/pages/ResaleCreatePage.tsx

import Layout from "../components/layout/Layout";

import ResaleCreateForm from "../features/resale/presentation/components/ResaleCreateForm";
import ResaleCreateMissingTarget from "../features/resale/presentation/components/ResaleCreateMissingTarget";
import ResaleListingTargetCard from "../features/resale/presentation/components/ResaleListingTargetCard";

import {
  useResaleCreatePage,
} from "../features/resale/presentation/hooks/useResaleCreatePage";

import "../styles/page-layout.css";
import "../styles/resale-page.css";

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
    submitButtonLabel,

    handlePriceChange,
    handleConditionChange,
    handleDescriptionChange,

    handleConditionMediaSelected,
    handleRemoveConditionMedia,
    handleConditionMediaCarouselScroll,
    handleMoveToConditionMediaSlide,

    handleBackToWallet,
    handleSubmit,
  } = useResaleCreatePage();

  return (
    <Layout
      title="出品"
      showBackButton
      mode="mypage"
      showFooter
      footerProps={{
        variant: "action",
        buttonLabel:
          submitButtonLabel,
        disabled:
          !canSubmit ||
          isSubmitting,
        onButtonClick:
          handleSubmit,
      }}
    >
      <section className="page-section">
        {!hasRequiredListingTarget ? (
          <ResaleCreateMissingTarget
            onBackToWallet={
              handleBackToWallet
            }
          />
        ) : (
          <div className="page-stack">
            <ResaleListingTargetCard
              target={target}
            />

            <ResaleCreateForm
              formattedPrice={
                formattedPrice
              }
              condition={
                condition
              }
              description={
                description
              }
              conditionMediaItems={
                conditionMediaItems
              }
              conditionMediaCurrentIndex={
                conditionMediaCurrentIndex
              }
              conditionMediaInputRef={
                conditionMediaInputRef
              }
              conditionMediaCarouselRef={
                conditionMediaCarouselRef
              }
              disabled={
                isSubmitting
              }
              onPriceChange={
                handlePriceChange
              }
              onConditionChange={
                handleConditionChange
              }
              onDescriptionChange={
                handleDescriptionChange
              }
              onConditionMediaSelected={
                handleConditionMediaSelected
              }
              onRemoveConditionMedia={
                handleRemoveConditionMedia
              }
              onConditionMediaCarouselScroll={
                handleConditionMediaCarouselScroll
              }
              onMoveToConditionMediaSlide={
                handleMoveToConditionMediaSlide
              }
            />

            {errorMessage ? (
              <p
                className="page-error"
                role="alert"
              >
                {errorMessage}
              </p>
            ) : null}
          </div>
        )}
      </section>
    </Layout>
  );
}