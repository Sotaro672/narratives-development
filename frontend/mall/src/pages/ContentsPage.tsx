// frontend/mall/src/pages/ContentsPage.tsx

import {
  useCallback,
  useRef,
} from "react";
import { ChevronDown, X } from "lucide-react";
import { useLocation, useNavigate } from "react-router-dom";

import "../styles/page-layout.css";
import "../styles/contents-page.css";

import Layout from "../components/layout/Layout";
import MobileComposerFooter from "../components/layout/MobileComposerFooter";
import MobileSwipeDismissPage, {
  type MobileSwipeDismissPageHandle,
} from "../components/layout/MobileSwipeDismissPage";
import IconButton from "../components/ui/IconButton";
import ContentsDetailPanel from "../features/contents/components/ContentsDetailPanel";
import ContentsMediaPanel from "../features/contents/components/ContentsMediaPanel";
import { useContentsPage } from "../features/contents/hooks/useContentsPage";

export default function ContentsPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const page = useContentsPage();
  const swipeDismissRef =
    useRef<MobileSwipeDismissPageHandle | null>(null);

  const isWalletOverlay =
    page.isMobilePortrait &&
    location.pathname === "/wallet/contents";

  const isReplying =
    Boolean(page.commentCard.replyingCommentId);

  const commentActionDisabled =
    page.commentCard.posting ||
    page.loading ||
    !page.contents.tokenBlueprintId;

  const replyActionDisabled =
    page.commentCard.replyPosting ||
    page.loading ||
    !page.contents.tokenBlueprintId;

  const canSubmitComment =
    !commentActionDisabled &&
    page.commentCard.commentBody.trim().length > 0;

  const canSubmitReply =
    !replyActionDisabled &&
    Boolean(page.commentCard.replyingCommentId) &&
    page.commentCard.replyBody.trim().length > 0;

  const handleSubmitComment = useCallback(() => {
    if (!canSubmitComment) {
      return;
    }

    void page.commentCard.postComment();
  }, [
    canSubmitComment,
    page.commentCard,
  ]);

  const handleSubmitReply = useCallback(() => {
    const replyingCommentId =
      page.commentCard.replyingCommentId;

    if (
      !replyingCommentId ||
      !canSubmitReply
    ) {
      return;
    }

    void page.commentCard.submitReply(
      replyingCommentId,
    );
  }, [
    canSubmitReply,
    page.commentCard,
  ]);

  const handleCancelReply = useCallback(() => {
    if (page.commentCard.replyPosting) {
      return;
    }

    page.commentCard.cancelReply();
  }, [page.commentCard]);

  const handleDismissStart = useCallback(() => {
    if (typeof document === "undefined") {
      return;
    }

    document
      .querySelectorAll<HTMLVideoElement>("video")
      .forEach((video) => {
        if (!video.paused) {
          video.pause();
        }
      });
  }, []);

  const handleDismiss = useCallback(() => {
    navigate("/wallet", {
      replace: true,
    });
  }, [navigate]);

  const handleDismissButtonClick = useCallback(() => {
    swipeDismissRef.current?.dismiss();
  }, []);

  const content = (
    <>
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

      {page.isMobilePortrait ? (
        <MobileComposerFooter
          content={
            isReplying
              ? page.commentCard.replyBody
              : page.commentCard.commentBody
          }
          placeholder={
            isReplying
              ? "返信を書く…"
              : "コメントを書く…"
          }
          error={
            page.commentCard.commentsError ||
            undefined
          }
          submitting={
            isReplying
              ? page.commentCard.replyPosting
              : page.commentCard.posting
          }
          canSubmit={
            isReplying
              ? canSubmitReply
              : canSubmitComment
          }
          disabled={
            isReplying
              ? replyActionDisabled
              : commentActionDisabled
          }
          submitLabel={
            isReplying
              ? "返信"
              : "投稿"
          }
          submittingLabel="投稿中..."
          beforeInput={
            isReplying ? (
              <div className="contents-page__reply-composer-header">
                <span className="contents-page__reply-composer-label">
                  返信中
                </span>

                <IconButton
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="contents-page__reply-composer-close"
                  aria-label="返信をキャンセル"
                  disabled={page.commentCard.replyPosting}
                  onClick={handleCancelReply}
                >
                  <X
                    size={18}
                    aria-hidden="true"
                  />
                </IconButton>
              </div>
            ) : undefined
          }
          onContentChange={
            isReplying
              ? page.commentCard.setReplyBody
              : page.commentCard.setCommentBody
          }
          onSubmit={
            isReplying
              ? handleSubmitReply
              : handleSubmitComment
          }
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
      onDismissStart={handleDismissStart}
      onDismiss={handleDismiss}
    >
      {content}
    </MobileSwipeDismissPage>
  );
}