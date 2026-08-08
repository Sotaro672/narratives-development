// frontend/amol/src/pages/ResaleDetailPage.tsx

import Layout from "../components/layout/Layout";
import SectionHeader from "../components/ui/SectionHeader";

import ResaleDetailEditForm from "../features/resale/presentation/components/ResaleDetailEditForm";
import ResaleDetailModelInfo from "../features/resale/presentation/components/ResaleDetailModelInfo";
import ResaleDetailReadonlyInfo from "../features/resale/presentation/components/ResaleDetailReadonlyInfo";
import ResaleListingTargetCard from "../features/resale/presentation/components/ResaleListingTargetCard";
import { useResaleDetailPage } from "../features/resale/presentation/hooks/useResaleDetailPage";

import "../styles/page-layout.css";
import "../styles/resale-page.css";
import "../styles/resale-detail-page.css";

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
    modelInfoProps,
    readonlyInfoProps,
    editFormProps,
    handleBack,
    handleReload,
    handleBackToWallet,
  } = useResaleDetailPage();

  const showLoadError =
    !loading &&
    !item &&
    Boolean(errorMessage);

  const showDetail =
    !loading &&
    Boolean(item);

  const headerActionProps =
    footerProps?.variant === "action"
      ? {
          actionButtonLabel:
            footerProps.buttonLabel,
          onActionButtonClick:
            footerProps.onButtonClick,
          actionButtonDisabled:
            footerProps.disabled,
        }
      : footerProps?.variant === "tripleAction"
        ? {
            actionButtonLabel:
              footerProps.centerButtonLabel,
            onActionButtonClick:
              footerProps.onCenterButtonClick,
            actionButtonDisabled:
              footerProps.centerButtonDisabled,

            secondaryActionButtonLabel:
              footerProps.leftButtonLabel,
            onSecondaryActionButtonClick:
              footerProps.onLeftButtonClick,
            secondaryActionButtonDisabled:
              footerProps.leftButtonDisabled,

            tertiaryActionButtonLabel:
              footerProps.rightButtonLabel,
            onTertiaryActionButtonClick:
              footerProps.onRightButtonClick,
            tertiaryActionButtonDisabled:
              footerProps.rightButtonDisabled,
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
      <section className="page-section resale-detail-page">
        {loading ? (
          <div className="page-card">
            <p className="page-card__text">
              読み込み中です...
            </p>
          </div>
        ) : null}

        {showLoadError ? (
          <div className="page-card">
            <SectionHeader
              title="出品情報を表示できません"
              titleAs="h2"
            >
              <p className="page-card__text">
                {errorMessage}
              </p>
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
          <div className="page-stack">
            <ResaleListingTargetCard
              target={listingTarget}
            />

            <ResaleDetailModelInfo
              {...modelInfoProps}
            />

            {isEditing ? (
              <ResaleDetailEditForm
                {...editFormProps}
              />
            ) : (
              <ResaleDetailReadonlyInfo
                {...readonlyInfoProps}
              />
            )}

            {errorMessage ? (
              <p
                className="page-error"
                role="alert"
              >
                {errorMessage}
              </p>
            ) : null}

            {saveMessage ? (
              <p
                className="page-card__text"
                role="status"
              >
                {saveMessage}
              </p>
            ) : null}

            <div className="resale-detail-page__footer-spacer" />
          </div>
        ) : null}
      </section>
    </Layout>
  );
}