// frontend/mall/src/pages/ContentsPage.tsx

import { ChevronDown } from "lucide-react";
import { useLocation } from "react-router-dom";

import "../styles/page-layout.css";
import "../styles/contents-page.css";

import Layout from "../components/layout/Layout";
import MobileComposerFooter from "../components/layout/MobileComposerFooter";
import MobileSwipeDismissPage from "../components/layout/MobileSwipeDismissPage";
import IconButton from "../components/ui/IconButton";
import ContentsDetailPanel from "../features/contents/components/ContentsDetailPanel";
import ContentsMediaPanel from "../features/contents/components/ContentsMediaPanel";
import { useContentsCommentComposer } from "../features/contents/hooks/useContentsCommentComposer";
import { useContentsPage } from "../features/contents/hooks/useContentsPage";
import { useContentsSwipeDismiss } from "../features/contents/hooks/useContentsSwipeDismiss";
import { useTokenCommentReplyAnchor } from "../features/contents/hooks/useTokenCommentReplyAnchor";

export default function ContentsPage() {
  const location = useLocation();
  const page = useContentsPage();

  const isWalletOverlay =
    page.isMobilePortrait &&
    location.pathname === "/wallet/contents";

  const {
    isReplying,
    composerProps,
  } = useContentsCommentComposer({
    commentCard:
      page.commentCard,
    loading:
      page.loading,
    tokenBlueprintId:
      page.contents.tokenBlueprintId,
  });

  const {
    resetReplyAnchor,
  } = useTokenCommentReplyAnchor({
    enabled:
      page.isMobilePortrait &&
      isReplying,
    replyingCommentId:
      page.commentCard.replyingCommentId,
    replyPosting:
      page.commentCard.replyPosting,
    onCancelReply:
      page.commentCard.cancelReply,
  });

  const {
    swipeDismissRef,
    handleDismissStart,
    handleDismiss,
    handleDismissButtonClick,
  } = useContentsSwipeDismiss({
    onBeforeDismiss:
      resetReplyAnchor,
  });

  const content = (
    <>
      <Layout
        title="AMOL"
        mode="mypage"
        showHeader={
          !page.isMobilePortrait
        }
        showFooter={
          !page.isMobilePortrait
        }
        disableFooterPaddingOnDesktop
      >
        {isWalletOverlay ? (
          <div
            className="contents-page__dismiss-control"
            data-mobile-swipe-dismiss-ignore="true"
          >
            <IconButton
              type="button"
              variant="secondary"
              size="md"
              className="contents-page__dismiss-button"
              aria-label="コンテンツを閉じる"
              onClick={
                handleDismissButtonClick
              }
            >
              <ChevronDown
                size={24}
                strokeWidth={2.25}
                aria-hidden="true"
              />
            </IconButton>
          </div>
        ) : null}

        <section className="split-page contents-page">
          <div className="split-page-content contents-page-content">
            <ContentsMediaPanel
              loading={
                page.loading
              }
              error={
                page.error
              }
              metadataUri={
                page.contents.metadataUri
              }
              moderationHidden={
                page.moderationHidden
              }
              hasMediaItems={
                page.hasMediaItems
              }
              mediaItems={
                page.mediaItems
              }
              activeFileIndex={
                page.activeFileIndex
              }
              tokenName={
                page.tokenName
              }
              onPrevFile={
                page.handlePrevFile
              }
              onNextFile={
                page.handleNextFile
              }
              onSelectFile={
                page.setActiveFileIndex
              }
              onPageDismissStart={
                isWalletOverlay
                  ? handleDismissStart
                  : undefined
              }
              onPageDismiss={
                isWalletOverlay
                  ? handleDismiss
                  : undefined
              }
            />

            <ContentsDetailPanel
              contents={
                page.contents
              }
              tokenName={
                page.tokenName
              }
              tokenIconUrl={
                page.tokenIconUrl
              }
              loading={
                page.loading
              }
              isMobilePortrait={
                page.isMobilePortrait
              }
              commentCard={
                page.commentCard
              }
              resaleDisabled={
                page.resaleButtonDisabled
              }
              resaleLabel={
                page.resaleButtonLabel
              }
              onProductNameClick={
                page.handleProductNameClick
              }
              onBrandNameClick={
                page.handleBrandNameClick
              }
              onResaleClick={
                page.handleOpenResalePage
              }
            />
          </div>
        </section>
      </Layout>

      {page.isMobilePortrait ? (
        <MobileComposerFooter
          {...composerProps}
        />
      ) : null}
    </>
  );

  if (!isWalletOverlay) {
    return content;
  }

  return (
    <MobileSwipeDismissPage
      ref={swipeDismissRef}
      onDismissStart={
        handleDismissStart
      }
      onDismiss={
        handleDismiss
      }
    >
      {content}
    </MobileSwipeDismissPage>
  );
}