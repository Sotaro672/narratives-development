// frontend/amol/src/pages/ResaleDetailPage.tsx

import Layout from "../components/layout/Layout";
import SectionHeader from "../components/ui/SectionHeader";

import AvatarSummaryCard from "../features/shared/presentation/components/AvatarSummaryCard";
import ProductDescription from "../features/shared/presentation/components/ProductDescription";
import ProductDetailLayout from "../features/shared/presentation/components/ProductDetailLayout";
import ProductIdentity from "../features/shared/presentation/components/ProductIdentity";
import ProductMediaGallery from "../features/shared/presentation/components/ProductMediaGallery";
import ProductModelMeta from "../features/shared/presentation/components/ProductModelMeta";
import ProductPrice from "../features/shared/presentation/components/ProductPrice";
import TokenSummaryCard from "../features/shared/presentation/components/TokenSummaryCard";

import ResaleConditionMediaField from "../features/resale/presentation/components/ResaleConditionMediaField";
import ResaleDetailEditForm from "../features/resale/presentation/components/ResaleDetailEditForm";
import ResaleDetailReadonlyInfo from "../features/resale/presentation/components/ResaleDetailReadonlyInfo";
import { useResaleDetailPage } from "../features/resale/presentation/hooks/useResaleDetailPage";

import "../styles/page-layout.css";
import "../styles/resale-page.css";
import "../styles/resale-detail-page.css";
import "../features/shared/styles/product-detail.css";

export default function ResaleDetailPage() {
  const {
    title,
    footerProps,
    loading,
    item,
    isEditing,
    errorMessage,
    saveMessage,
    listingTarget,
    model,
    readonlyInfoProps,
    editFormProps,
    handleBack,
    handleReload,
    handleBackToWallet,
  } = useResaleDetailPage();

  const showLoadError = !loading && !item && Boolean(errorMessage);
  const showDetail = !loading && Boolean(item);

  const headerActionProps =
    footerProps?.variant === "action"
      ? {
          actionButtonLabel: footerProps.buttonLabel,
          onActionButtonClick: footerProps.onButtonClick,
          actionButtonDisabled: footerProps.disabled,
        }
      : footerProps?.variant === "tripleAction"
        ? {
            actionButtonLabel: footerProps.centerButtonLabel,
            onActionButtonClick: footerProps.onCenterButtonClick,
            actionButtonDisabled: footerProps.centerButtonDisabled,
            secondaryActionButtonLabel: footerProps.leftButtonLabel,
            onSecondaryActionButtonClick: footerProps.onLeftButtonClick,
            secondaryActionButtonDisabled: footerProps.leftButtonDisabled,
            tertiaryActionButtonLabel: footerProps.rightButtonLabel,
            onTertiaryActionButtonClick: footerProps.onRightButtonClick,
            tertiaryActionButtonDisabled: footerProps.rightButtonDisabled,
          }
        : {};

  return (
    <Layout
      title={title}
      titleClickable={false}
      showBackButton
      onBackButtonClick={handleBack}
      mode="mypage"
      hideAnnouncementButton
      hideSettingsButton
      showFooter={Boolean(footerProps)}
      footerProps={footerProps}
      {...headerActionProps}
    >
      <div className="page-layout resale-detail-page">
        {loading ? (
          <div className="resale-detail-page__state">
            <p>読み込み中です...</p>
          </div>
        ) : null}

        {showLoadError ? (
          <div className="resale-detail-page__state resale-detail-page__state--error">
            <SectionHeader title="出品情報を表示できません" titleAs="h2">
              <p>{errorMessage}</p>
            </SectionHeader>

            <div className="page-actions">
              <button
                type="button"
                className="page-button page-button--secondary"
                onClick={() => void handleReload()}
              >
                再読み込み
              </button>

              <button
                type="button"
                className="page-button page-button--primary"
                onClick={handleBackToWallet}
              >
                ウォレットへ戻る
              </button>
            </div>
          </div>
        ) : null}

        {showDetail ? (
          <ProductDetailLayout
            className="product-detail__layout--summary-after-content-mobile"
            media={
              isEditing ? (
                <div className="product-detail__image-wrap resale-detail-page__image-wrap--editing">
                  <ResaleConditionMediaField
                    items={editFormProps.conditionMediaItems}
                    currentIndex={editFormProps.conditionMediaCurrentIndex}
                    inputRef={editFormProps.conditionMediaInputRef}
                    carouselRef={editFormProps.conditionMediaCarouselRef}
                    disabled={editFormProps.saving}
                    selecting={editFormProps.saving}
                    onFilesSelected={editFormProps.onConditionMediaSelected}
                    onRemoveItem={editFormProps.onRemoveConditionMedia}
                    onCarouselScroll={editFormProps.onConditionMediaCarouselScroll}
                    onMoveToSlide={editFormProps.onMoveToConditionMediaSlide}
                  />
                </div>
              ) : (
                <ProductMediaGallery
                  items={readonlyInfoProps.galleryItems}
                  activeIndex={readonlyInfoProps.activeGalleryIndex}
                  altFallback="商品状態の写真"
                  placeholderText="商品状態の写真はありません。"
                  onPrev={readonlyInfoProps.onPrevGalleryItem}
                  onNext={readonlyInfoProps.onNextGalleryItem}
                  onSelect={readonlyInfoProps.onSelectGalleryItem}
                />
              )
            }
            mediaFooter={
              <>
                <TokenSummaryCard
                  brandName={listingTarget.brandName}
                  tokenName={listingTarget.tokenName}
                  tokenIcon={listingTarget.tokenIconUrl}
                  description={listingTarget.tokenDescription}
                />

                <AvatarSummaryCard
                  avatarId={item?.avatarId}
                  avatarName={item?.avatarName}
                  avatarIcon={item?.avatarIcon}
                />
              </>
            }
            contentFooter={
              !isEditing ? (
                <ResaleDetailReadonlyInfo
                  statusLabel={readonlyInfoProps.statusLabel}
                  createdAtLabel={readonlyInfoProps.createdAtLabel}
                  updatedAtLabel={readonlyInfoProps.updatedAtLabel}
                />
              ) : null
            }
          >
            <ProductIdentity
              brandName={listingTarget.brandName}
              productName={listingTarget.productName}
              tokenName={listingTarget.tokenName}
            />

            {isEditing ? (
              <>
                <ProductModelMeta model={model} />
                <ResaleDetailEditForm {...editFormProps} />
              </>
            ) : (
              <>
                <ProductPrice priceLabel={readonlyInfoProps.priceLabel} />
                <ProductModelMeta
                  conditionLabel={readonlyInfoProps.conditionLabel}
                  model={model}
                />
                <ProductDescription
                  description={readonlyInfoProps.description || "説明文はありません。"}
                />
              </>
            )}

            {errorMessage ? (
              <p
                className="resale-detail-page__message resale-detail-page__message--error"
                role="alert"
              >
                {errorMessage}
              </p>
            ) : null}

            {saveMessage ? (
              <p
                className="resale-detail-page__message resale-detail-page__message--success"
                role="status"
              >
                {saveMessage}
              </p>
            ) : null}
          </ProductDetailLayout>
        ) : null}
      </div>
    </Layout>
  );
}