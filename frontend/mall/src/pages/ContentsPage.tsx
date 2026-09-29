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

const OUTSIDE_TAP_MOVE_THRESHOLD = 8;
const ANCHOR_CORRECTION_THRESHOLD = 0.5;
const REPLY_VISIBLE_TOP_MARGIN = 16;

function findReplyTarget(commentId: string): HTMLElement | null {
  if (
    typeof document === "undefined" ||
    !commentId
  ) {
    return null;
  }

  const elements =
    document.querySelectorAll<HTMLElement>(
      "[data-token-comment-id]",
    );

  return (
    Array.from(elements).find(
      (element) =>
        element.dataset.tokenCommentId === commentId,
    ) ?? null
  );
}

function findScrollContainer(element: HTMLElement): HTMLElement | null {
  if (typeof window === "undefined") {
    return null;
  }

  let current = element.parentElement;

  while (current) {
    const style = window.getComputedStyle(current);
    const overflowY = style.overflowY;

    if (
      (
        overflowY === "auto" ||
        overflowY === "scroll" ||
        overflowY === "overlay"
      ) &&
      current.scrollHeight > current.clientHeight + 1
    ) {
      return current;
    }

    current = current.parentElement;
  }

  return null;
}

function getComposerFooter(): HTMLElement | null {
  if (typeof document === "undefined") {
    return null;
  }

  return document.querySelector<HTMLElement>(
    ".mobile-composer-footer",
  );
}

function getComposerTop(footer: HTMLElement): number {
  if (typeof window === "undefined") {
    return footer.getBoundingClientRect().top;
  }

  const visualViewport = window.visualViewport;

  if (!visualViewport) {
    return footer.getBoundingClientRect().top;
  }

  const footerHeight =
    footer.getBoundingClientRect().height;

  return (
    visualViewport.offsetTop +
    visualViewport.height -
    footerHeight
  );
}

function getVisibleViewportTop(): number {
  if (typeof window === "undefined") {
    return 0;
  }

  return (
    window.visualViewport?.offsetTop ??
    0
  );
}

function getKeyboardInset(): number {
  if (
    typeof window === "undefined" ||
    !window.visualViewport
  ) {
    return 0;
  }

  const visualViewport =
    window.visualViewport;

  const visibleBottom =
    visualViewport.offsetTop +
    visualViewport.height;

  return Math.max(
    0,
    window.innerHeight -
      visibleBottom,
  );
}

export default function ContentsPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const page = useContentsPage();

  const swipeDismissRef =
    useRef<MobileSwipeDismissPageHandle | null>(null);

  const replyAnchorGapRef =
    useRef<number | null>(null);

  const replyAnchorActiveRef =
    useRef(false);

  const replyComposerFocusedRef =
    useRef(false);

  const replyAnchorScrollContainerRef =
    useRef<HTMLElement | null>(null);

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

  const resetReplyAnchor = useCallback(() => {
    replyAnchorActiveRef.current = false;
    replyAnchorGapRef.current = null;
    replyAnchorScrollContainerRef.current = null;
  }, []);

  useEffect(() => {
    resetReplyAnchor();

    if (
      !page.isMobilePortrait ||
      !page.commentCard.replyingCommentId ||
      typeof window === "undefined"
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
        const targetElement =
          findReplyTarget(replyingCommentId);

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
    resetReplyAnchor,
  ]);

  useEffect(() => {
    if (
      !page.isMobilePortrait ||
      !isReplying ||
      typeof window === "undefined" ||
      typeof document === "undefined"
    ) {
      return;
    }

    const visualViewport =
      window.visualViewport;

    const detailElement =
      document.querySelector<HTMLElement>(
        ".contents-page-detail",
      );

    const previousInlinePaddingBottom =
      detailElement?.style.paddingBottom ?? "";

    let animationFrameId: number | null =
      null;

    const updateReplyScrollSpace = () => {
      if (!detailElement) {
        return;
      }

      if (!replyComposerFocusedRef.current) {
        detailElement.style.paddingBottom =
          "calc(var(--mobile-composer-height, 56px) + 24px)";
        return;
      }

      const keyboardInset =
        Math.ceil(getKeyboardInset());

      detailElement.style.paddingBottom =
        `calc(var(--mobile-composer-height, 56px) + ${keyboardInset}px + 24px)`;
    };

    const maintainReplyAnchor = () => {
      if (
        !replyAnchorActiveRef.current ||
        replyAnchorGapRef.current === null
      ) {
        return;
      }

      const replyingCommentId =
        page.commentCard.replyingCommentId?.trim() ?? "";

      if (!replyingCommentId) {
        resetReplyAnchor();
        return;
      }

      const targetElement =
        findReplyTarget(replyingCommentId);

      const footer =
        getComposerFooter();

      if (
        !targetElement ||
        !footer
      ) {
        resetReplyAnchor();
        return;
      }

      const targetRect =
        targetElement.getBoundingClientRect();

      const composerTop =
        getComposerTop(footer);

      let desiredTargetBottom =
        composerTop -
        replyAnchorGapRef.current;

      const minimumTargetTop =
        getVisibleViewportTop() +
        REPLY_VISIBLE_TOP_MARGIN;

      const minimumTargetBottom =
        minimumTargetTop +
        targetRect.height;

      if (
        desiredTargetBottom <
        minimumTargetBottom
      ) {
        desiredTargetBottom =
          minimumTargetBottom;
      }

      const scrollDelta =
        targetRect.bottom -
        desiredTargetBottom;

      if (
        Math.abs(scrollDelta) <
        ANCHOR_CORRECTION_THRESHOLD
      ) {
        return;
      }

      const scrollContainer =
        replyAnchorScrollContainerRef.current;

      if (scrollContainer) {
        scrollContainer.scrollBy({
          top: scrollDelta,
          left: 0,
          behavior: "auto",
        });

        return;
      }

      window.scrollBy({
        top: scrollDelta,
        left: 0,
        behavior: "auto",
      });
    };

    const scheduleAnchorCorrection = () => {
      if (animationFrameId !== null) {
        window.cancelAnimationFrame(
          animationFrameId,
        );
      }

      animationFrameId =
        window.requestAnimationFrame(() => {
          animationFrameId = null;

          updateReplyScrollSpace();

          if (
            replyAnchorActiveRef.current
          ) {
            maintainReplyAnchor();
          }
        });
    };

    const handleFocusIn = (
      event: FocusEvent,
    ) => {
      const target = event.target;

      if (
        !(target instanceof HTMLElement) ||
        !target.classList.contains(
          "mobile-composer-footer__input",
        )
      ) {
        return;
      }

      const replyingCommentId =
        page.commentCard.replyingCommentId?.trim() ?? "";

      if (!replyingCommentId) {
        return;
      }

      const replyTarget =
        findReplyTarget(replyingCommentId);

      const footer =
        getComposerFooter();

      if (
        !replyTarget ||
        !footer
      ) {
        return;
      }

      replyComposerFocusedRef.current =
        true;

      replyAnchorScrollContainerRef.current =
        findScrollContainer(replyTarget);

      const targetRect =
        replyTarget.getBoundingClientRect();

      const composerTop =
        getComposerTop(footer);

      replyAnchorGapRef.current =
        Math.max(
          0,
          composerTop -
            targetRect.bottom,
        );

      replyAnchorActiveRef.current =
        true;

      scheduleAnchorCorrection();
    };

    const handleFocusOut = (
      event: FocusEvent,
    ) => {
      const target = event.target;

      if (
        !(target instanceof HTMLElement) ||
        !target.classList.contains(
          "mobile-composer-footer__input",
        )
      ) {
        return;
      }

      window.requestAnimationFrame(() => {
        const activeElement =
          document.activeElement;

        if (
          activeElement instanceof HTMLElement &&
          activeElement.classList.contains(
            "mobile-composer-footer__input",
          )
        ) {
          return;
        }

        replyComposerFocusedRef.current =
          false;

        resetReplyAnchor();
        updateReplyScrollSpace();
      });
    };

    document.addEventListener(
      "focusin",
      handleFocusIn,
    );

    document.addEventListener(
      "focusout",
      handleFocusOut,
    );

    visualViewport?.addEventListener(
      "resize",
      scheduleAnchorCorrection,
      {
        passive: true,
      },
    );

    visualViewport?.addEventListener(
      "scroll",
      scheduleAnchorCorrection,
      {
        passive: true,
      },
    );

    window.addEventListener(
      "resize",
      scheduleAnchorCorrection,
      {
        passive: true,
      },
    );

    updateReplyScrollSpace();

    return () => {
      if (animationFrameId !== null) {
        window.cancelAnimationFrame(
          animationFrameId,
        );
      }

      document.removeEventListener(
        "focusin",
        handleFocusIn,
      );

      document.removeEventListener(
        "focusout",
        handleFocusOut,
      );

      visualViewport?.removeEventListener(
        "resize",
        scheduleAnchorCorrection,
      );

      visualViewport?.removeEventListener(
        "scroll",
        scheduleAnchorCorrection,
      );

      window.removeEventListener(
        "resize",
        scheduleAnchorCorrection,
      );

      replyComposerFocusedRef.current =
        false;

      resetReplyAnchor();

      if (detailElement) {
        detailElement.style.paddingBottom =
          previousInlinePaddingBottom;
      }
    };
  }, [
    isReplying,
    page.commentCard.replyingCommentId,
    page.isMobilePortrait,
    resetReplyAnchor,
  ]);

  useEffect(() => {
    if (
      !page.isMobilePortrait ||
      !isReplying ||
      typeof document === "undefined"
    ) {
      return;
    }

    let gesture:
      | {
          pointerId: number;
          startX: number;
          startY: number;
          moved: boolean;
        }
      | null = null;

    const handlePointerDown = (
      event: PointerEvent,
    ) => {
      if (
        page.commentCard.replyPosting
      ) {
        return;
      }

      const target = event.target;

      if (!(target instanceof Element)) {
        return;
      }

      if (
        target.closest(
          ".mobile-composer-footer",
        )
      ) {
        gesture = null;
        return;
      }

      resetReplyAnchor();

      gesture = {
        pointerId: event.pointerId,
        startX: event.clientX,
        startY: event.clientY,
        moved: false,
      };
    };

    const handlePointerMove = (
      event: PointerEvent,
    ) => {
      if (
        !gesture ||
        gesture.pointerId !==
          event.pointerId
      ) {
        return;
      }

      const deltaX =
        event.clientX -
        gesture.startX;

      const deltaY =
        event.clientY -
        gesture.startY;

      if (
        Math.hypot(
          deltaX,
          deltaY,
        ) >=
        OUTSIDE_TAP_MOVE_THRESHOLD
      ) {
        gesture.moved = true;
      }
    };

    const handlePointerUp = (
      event: PointerEvent,
    ) => {
      if (
        !gesture ||
        gesture.pointerId !==
          event.pointerId
      ) {
        return;
      }

      const completedGesture =
        gesture;

      gesture = null;

      if (
        completedGesture.moved ||
        page.commentCard.replyPosting
      ) {
        return;
      }

      const target = event.target;

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

    const handlePointerCancel = (
      event: PointerEvent,
    ) => {
      if (
        !gesture ||
        gesture.pointerId !==
          event.pointerId
      ) {
        return;
      }

      gesture = null;
    };

    const handleWheel = (
      event: WheelEvent,
    ) => {
      const target = event.target;

      if (
        target instanceof Element &&
        target.closest(
          ".mobile-composer-footer",
        )
      ) {
        return;
      }

      resetReplyAnchor();
    };

    document.addEventListener(
      "pointerdown",
      handlePointerDown,
    );

    document.addEventListener(
      "pointermove",
      handlePointerMove,
      {
        passive: true,
      },
    );

    document.addEventListener(
      "pointerup",
      handlePointerUp,
    );

    document.addEventListener(
      "pointercancel",
      handlePointerCancel,
    );

    document.addEventListener(
      "wheel",
      handleWheel,
      {
        passive: true,
      },
    );

    return () => {
      document.removeEventListener(
        "pointerdown",
        handlePointerDown,
      );

      document.removeEventListener(
        "pointermove",
        handlePointerMove,
      );

      document.removeEventListener(
        "pointerup",
        handlePointerUp,
      );

      document.removeEventListener(
        "pointercancel",
        handlePointerCancel,
      );

      document.removeEventListener(
        "wheel",
        handleWheel,
      );
    };
  }, [
    isReplying,
    page.commentCard.cancelReply,
    page.commentCard.replyPosting,
    page.isMobilePortrait,
    resetReplyAnchor,
  ]);

  const handleSubmitComment =
    useCallback(() => {
      if (!canSubmitComment) {
        return;
      }

      void page.commentCard.postComment();
    }, [
      canSubmitComment,
      page.commentCard,
    ]);

  const handleSubmitReply =
    useCallback(() => {
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

  const handleDismissStart =
    useCallback(() => {
      resetReplyAnchor();

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
    }, [
      resetReplyAnchor,
    ]);

  const handleDismiss =
    useCallback(() => {
      resetReplyAnchor();

      navigate("/wallet", {
        replace: true,
      });
    }, [
      navigate,
      resetReplyAnchor,
    ]);

  const handleDismissButtonClick =
    useCallback(() => {
      swipeDismissRef.current?.dismiss();
    }, []);

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