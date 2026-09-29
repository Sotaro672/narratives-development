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
import TokenCommentReplySection from "../features/token-commnet/components/TokenCommentReplySection";

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
    commentCard: page.commentCard,
    loading: page.loading,
    tokenBlueprintId: page.contents.tokenBlueprintId,
  });

  const isReplySectionOpen =
    page.isMobilePortrait &&
    isReplying &&
    Boolean(page.commentCard.replyingCommentId);

  const {
    swipeDismissRef,
    handleDismissStart,
    handleDismiss,
    handleDismissButtonClick,
  } = useContentsSwipeDismiss();

  const mainContent = (
    <Layout
      title="AMOL"
      mode="mypage"
      showHeader={!page.isMobilePortrait}
      showFooter={!page.isMobilePortrait}
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
            onClick={handleDismissButtonClick}
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
            loading={page.loading}
            error={page.error}
            metadataUri={page.contents.metadataUri}
            moderationHidden={page.moderationHidden}
            hasMediaItems={page.hasMediaItems}
            mediaItems={page.mediaItems}
            activeFileIndex={page.activeFileIndex}
            tokenName={page.tokenName}
            onPrevFile={page.handlePrevFile}
            onNextFile={page.handleNextFile}
            onSelectFile={page.setActiveFileIndex}
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
            contents={page.contents}
            tokenName={page.tokenName}
            tokenIconUrl={page.tokenIconUrl}
            loading={page.loading}
            isMobilePortrait={page.isMobilePortrait}
            commentCard={page.commentCard}
            resaleDisabled={page.resaleButtonDisabled}
            resaleLabel={page.resaleButtonLabel}
            onProductNameClick={page.handleProductNameClick}
            onBrandNameClick={page.handleBrandNameClick}
            onResaleClick={page.handleOpenResalePage}
          />
        </div>
      </section>
    </Layout>
  );

  const mobileContent = (
    <>
      <div
        className={[
          "contents-page-mobile-stage",
          isReplySectionOpen
            ? "contents-page-mobile-stage--reply-open"
            : "",
        ]
          .filter(Boolean)
          .join(" ")}
      >
        <div
          className="contents-page-mobile-stage__main"
          aria-hidden={isReplySectionOpen || undefined}
        >
          {mainContent}
        </div>

        <div
          className="contents-page-mobile-stage__reply"
          aria-hidden={!isReplySectionOpen || undefined}
          data-mobile-swipe-dismiss-ignore={
            isReplySectionOpen
              ? "true"
              : undefined
          }
        >
          {page.commentCard.replyingCommentId ? (
            <TokenCommentReplySection
              commentId={page.commentCard.replyingCommentId}
              tokenBlueprintId={page.contents.tokenBlueprintId}
              commentTree={page.commentCard.commentTree}
              expandedIds={page.commentCard.expandedIds}
              replyBody={page.commentCard.replyBody}
              replyPosting={page.commentCard.replyPosting}
              commentsLoading={page.commentCard.commentsLoading}
              commentsError={page.commentCard.commentsError}
              onBack={page.commentCard.cancelReply}
              onReplyBodyChange={page.commentCard.setReplyBody}
              onSubmitReply={page.commentCard.submitReply}
              onToggleExpanded={page.commentCard.toggleExpanded}
              onLike={page.commentCard.likeComment}
              onDislike={page.commentCard.dislikeComment}
            />
          ) : null}
        </div>
      </div>

      <MobileComposerFooter
        {...composerProps}
      />
    </>
  );

  const content =
    page.isMobilePortrait
      ? mobileContent
      : mainContent;

  if (!isWalletOverlay) {
    return content;
  }

  return (
    <MobileSwipeDismissPage
      ref={swipeDismissRef}
      enabled={!isReplySectionOpen}
      onDismissStart={handleDismissStart}
      onDismiss={handleDismiss}
    >
      {content}
    </MobileSwipeDismissPage>
  );
}