// frontend/mall/src/pages/InquiryPage.tsx

import { useCallback } from "react";
import { useNavigate } from "react-router-dom";

import { useMobilePortrait } from "../components/hooks/useMobilePortrait";
import Layout from "../components/layout/Layout";
import MobileComposerFooter from "../components/layout/MobileComposerFooter";
import MobileSwipeRightDismissPage from "../components/layout/MobileSwipeRightDismissPage";
import Alert from "../components/ui/Alert";
import StatePanel from "../components/ui/StatePanel";
import ContactUploadProgressModal from "../features/contact/components/ContactUploadProgressModal";
import { useSettingsInquiryPage } from "../features/contact/hooks/useSettingsInquiryPage";
import ChatInlineComposer from "../features/shared/presentation/components/ChatInlineComposer";

import "../styles/page-layout.css";
import "../features/shared/styles/chat-detail-page.css";
import "../styles/inquiry-page.css";

export default function InquiryPage() {
  const navigate = useNavigate();
  const isMobilePortrait = useMobilePortrait();

  const {
    loadingUser,
    userError,
    composer,
    uploadingAttachments,
    uploadProgress,
    uploadFileProgress,
    uploadFileIndex,
    uploadFileCount,
  } = useSettingsInquiryPage();

  const shouldShowMobileComposer =
    isMobilePortrait &&
    !loadingUser &&
    !userError;

  const shouldShowDesktopComposer =
    !isMobilePortrait &&
    !loadingUser &&
    !userError;

  const pageLayoutClassName = [
    "chat-detail-page-layout",
    "chat-detail-page-layout--inquiry",
    "settings-inquiry-page-layout",
    shouldShowMobileComposer
      ? "settings-inquiry-page-layout--with-mobile-composer"
      : "",
  ]
    .filter(Boolean)
    .join(" ");

  const handleDismiss = useCallback(() => {
    navigate(-1);
  }, [navigate]);

  const pageContent = (
    <Layout
      title="AMOL"
      mode="mypage"
      showHeader={!isMobilePortrait}
      showFooter={false}
      mainClassName="inquiry-page"
      disableFooterPaddingOnDesktop
    >
      <div
        className={pageLayoutClassName}
        data-chat-detail
      >
        <section className="product-detail-page-layout chat-detail-page">
          <div className="settings-inquiry-page">
            <div className="settings-inquiry-page__content">
              <header className="settings-inquiry-page__header">
                <h1 className="settings-inquiry-page__title">
                  問い合わせ
                </h1>

                <p className="settings-inquiry-page__description">
                  AMOLへのお問い合わせ内容を入力してください。
                </p>
              </header>

              {loadingUser ? (
                <StatePanel
                  variant="loading"
                  title="ユーザー情報を確認中..."
                />
              ) : null}

              {!loadingUser && userError ? (
                <Alert
                  variant="error"
                  className="settings-inquiry-page__error"
                >
                  {userError}
                </Alert>
              ) : null}

              {!loadingUser && !userError ? (
                <div className="settings-inquiry-page__notice">
                  <p className="settings-inquiry-page__description">
                    お問い合わせ内容によっては、ご回答までにお時間をいただく場合があります。
                  </p>
                </div>
              ) : null}
            </div>

            {shouldShowDesktopComposer ? (
              <ChatInlineComposer {...composer} />
            ) : null}
          </div>
        </section>
      </div>
    </Layout>
  );

  return (
    <>
      {isMobilePortrait ? (
        <MobileSwipeRightDismissPage
          title="問い合わせ"
          enabled
          dismissGestureEnabled
          onDismiss={handleDismiss}
        >
          {pageContent}
        </MobileSwipeRightDismissPage>
      ) : (
        pageContent
      )}

      {shouldShowMobileComposer ? (
        <MobileComposerFooter {...composer} />
      ) : null}

      <ContactUploadProgressModal
        open={uploadingAttachments}
        progress={uploadProgress}
        fileProgress={uploadFileProgress}
        fileIndex={uploadFileIndex}
        fileCount={uploadFileCount}
      />
    </>
  );
}