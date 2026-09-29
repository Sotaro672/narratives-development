// frontend/mall/src/pages/ContentsPage.tsx

import {
  useCallback,
  useEffect,
  useRef,
} from "react";
import { ChevronDown } from "lucide-react";
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

  useEffect(() => {
    if (
      !page.isMobilePortrait ||
      !page.commentCard.replyingCommentId ||
      typeof document === "undefined"
    ) {
      return;
    }

    const replyingCommentId =
      page.commentCard.replyingCommentId.trim();

    if (!replyingCommentId) {
      return;
    }

    const animationFrameId =
      window.requestAnimationFrame(() => {
        const commentElements =
          document.querySelectorAll<HTMLElement>(
            "[data-token-comment-id]",
          );

        const targetElement =
          Array.from(commentElements).find(
            (element) =>
              element.dataset.tokenCommentId ===
              replyingCommentId,
          );

        targetElement?.scrollIntoView({
          behavior: "smooth",
          block: "center",
          inline: "nearest",
        });
      });

    return () => {
      window.cancelAnimationFrame(
        animationFrameId,
      );
    };
  }, [
    page.commentCard.replyingCommentId,
    page.isMobilePortrait,
  ]);

  useEffect(() => {
    if (
      !page.isMobilePortrait ||
      !isReplying ||
      typeof document === "undefined"
    ) {
      return;
    }

    const handlePointerDown = (
      event: PointerEvent,
    ) => {
      if (
        page.commentCard.replyPosting
      ) {
        return;
      }

      const target =
        event.target;

      if (!(target instanceof Element)) {
        return;
      }

      if (
        target.closest(
          ".mobile-composer-footer",
        )
      ) {
        return;
      }

      page.commentCard.cancelReply();
    };

    document.addEventListener(
      "pointerdown",
      handlePointerDown,
    );

    return () => {
      document.removeEventListener(
        "pointerdown",
        handlePointerDown,
      );
    };
  }, [
    isReplying,
    page.commentCard.cancelReply,
    page.commentCard.replyPosting,
    page.isMobilePortrait,
  ]);

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

  const handleDismissStart = useCallback(() => {
    if (
      typeof document === "undefined"
    ) {
      return;
    }

    document
      .querySelectorAll<HTMLVideoElement>(
        "video",
      )
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

  const handleDismissButtonClick =
    useCallback(() => {
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