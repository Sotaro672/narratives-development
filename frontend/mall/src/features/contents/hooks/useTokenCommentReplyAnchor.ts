// frontend/mall/src/features/contents/hooks/useTokenCommentReplyAnchor.ts

import {
  useCallback,
  useEffect,
  useRef,
} from "react";

const OUTSIDE_TAP_MOVE_THRESHOLD = 8;
const ANCHOR_CORRECTION_THRESHOLD = 0.5;
const REPLY_VISIBLE_TOP_MARGIN = 16;

type UseTokenCommentReplyAnchorParams = {
  enabled: boolean;
  replyingCommentId: string | null;
  replyPosting: boolean;
  onCancelReply: () => void;
};

function findReplyTarget(
  commentId: string,
): HTMLElement | null {
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
        element.dataset.tokenCommentId ===
        commentId,
    ) ?? null
  );
}

function findScrollContainer(
  element: HTMLElement,
): HTMLElement | null {
  if (typeof window === "undefined") {
    return null;
  }

  let current = element.parentElement;

  while (current) {
    const style =
      window.getComputedStyle(current);

    const overflowY =
      style.overflowY;

    if (
      (
        overflowY === "auto" ||
        overflowY === "scroll" ||
        overflowY === "overlay"
      ) &&
      current.scrollHeight >
        current.clientHeight + 1
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

function getComposerTop(
  footer: HTMLElement,
): number {
  if (typeof window === "undefined") {
    return footer.getBoundingClientRect().top;
  }

  const visualViewport =
    window.visualViewport;

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

export function useTokenCommentReplyAnchor({
  enabled,
  replyingCommentId,
  replyPosting,
  onCancelReply,
}: UseTokenCommentReplyAnchorParams) {
  const anchorGapRef =
    useRef<number | null>(null);

  const anchorActiveRef =
    useRef(false);

  const composerFocusedRef =
    useRef(false);

  const scrollContainerRef =
    useRef<HTMLElement | null>(null);

  const resetReplyAnchor =
    useCallback(() => {
      anchorActiveRef.current = false;
      anchorGapRef.current = null;
      scrollContainerRef.current = null;
    }, []);

  useEffect(() => {
    resetReplyAnchor();

    if (
      !enabled ||
      !replyingCommentId ||
      typeof window === "undefined"
    ) {
      return;
    }

    const normalizedCommentId =
      replyingCommentId.trim();

    if (!normalizedCommentId) {
      return;
    }

    const animationFrameId =
      window.requestAnimationFrame(() => {
        const targetElement =
          findReplyTarget(
            normalizedCommentId,
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
    enabled,
    replyingCommentId,
    resetReplyAnchor,
  ]);

  useEffect(() => {
    if (
      !enabled ||
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

    const previousPaddingBottom =
      detailElement?.style.paddingBottom ??
      "";

    let animationFrameId:
      number | null = null;

    const updateScrollSpace = () => {
      if (!detailElement) {
        return;
      }

      if (!composerFocusedRef.current) {
        detailElement.style.paddingBottom =
          "calc(var(--mobile-composer-height, 56px) + 24px)";
        return;
      }

      const keyboardInset =
        Math.ceil(
          getKeyboardInset(),
        );

      detailElement.style.paddingBottom =
        `calc(var(--mobile-composer-height, 56px) + ${keyboardInset}px + 24px)`;
    };

    const maintainAnchor = () => {
      if (
        !anchorActiveRef.current ||
        anchorGapRef.current === null
      ) {
        return;
      }

      const normalizedCommentId =
        replyingCommentId?.trim() ?? "";

      if (!normalizedCommentId) {
        resetReplyAnchor();
        return;
      }

      const replyTarget =
        findReplyTarget(
          normalizedCommentId,
        );

      const footer =
        getComposerFooter();

      if (
        !replyTarget ||
        !footer
      ) {
        resetReplyAnchor();
        return;
      }

      const targetRect =
        replyTarget.getBoundingClientRect();

      const composerTop =
        getComposerTop(footer);

      let desiredTargetBottom =
        composerTop -
        anchorGapRef.current;

      const minimumTargetTop =
        getVisibleViewportTop() +
        REPLY_VISIBLE_TOP_MARGIN;

      const minimumTargetBottom =
        minimumTargetTop +
        targetRect.height;

      desiredTargetBottom =
        Math.max(
          desiredTargetBottom,
          minimumTargetBottom,
        );

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
        scrollContainerRef.current;

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

    const scheduleCorrection = () => {
      if (animationFrameId !== null) {
        window.cancelAnimationFrame(
          animationFrameId,
        );
      }

      animationFrameId =
        window.requestAnimationFrame(() => {
          animationFrameId = null;

          updateScrollSpace();

          if (
            anchorActiveRef.current
          ) {
            maintainAnchor();
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

      const normalizedCommentId =
        replyingCommentId?.trim() ?? "";

      if (!normalizedCommentId) {
        return;
      }

      const replyTarget =
        findReplyTarget(
          normalizedCommentId,
        );

      const footer =
        getComposerFooter();

      if (
        !replyTarget ||
        !footer
      ) {
        return;
      }

      composerFocusedRef.current =
        true;

      scrollContainerRef.current =
        findScrollContainer(
          replyTarget,
        );

      const targetRect =
        replyTarget.getBoundingClientRect();

      const composerTop =
        getComposerTop(footer);

      anchorGapRef.current =
        Math.max(
          0,
          composerTop -
            targetRect.bottom,
        );

      anchorActiveRef.current =
        true;

      scheduleCorrection();
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

        composerFocusedRef.current =
          false;

        resetReplyAnchor();
        updateScrollSpace();
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
      scheduleCorrection,
      {
        passive: true,
      },
    );

    visualViewport?.addEventListener(
      "scroll",
      scheduleCorrection,
      {
        passive: true,
      },
    );

    window.addEventListener(
      "resize",
      scheduleCorrection,
      {
        passive: true,
      },
    );

    updateScrollSpace();

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
        scheduleCorrection,
      );

      visualViewport?.removeEventListener(
        "scroll",
        scheduleCorrection,
      );

      window.removeEventListener(
        "resize",
        scheduleCorrection,
      );

      composerFocusedRef.current =
        false;

      resetReplyAnchor();

      if (detailElement) {
        detailElement.style.paddingBottom =
          previousPaddingBottom;
      }
    };
  }, [
    enabled,
    replyingCommentId,
    resetReplyAnchor,
  ]);

  useEffect(() => {
    if (
      !enabled ||
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
      if (replyPosting) {
        return;
      }

      const target =
        event.target;

      if (
        !(target instanceof Element)
      ) {
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
        pointerId:
          event.pointerId,
        startX:
          event.clientX,
        startY:
          event.clientY,
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
        replyPosting
      ) {
        return;
      }

      const target =
        event.target;

      if (
        !(target instanceof Element)
      ) {
        return;
      }

      if (
        target.closest(
          ".mobile-composer-footer",
        )
      ) {
        return;
      }

      onCancelReply();
    };

    const handlePointerCancel = (
      event: PointerEvent,
    ) => {
      if (
        gesture?.pointerId !==
        event.pointerId
      ) {
        return;
      }

      gesture = null;
    };

    const handleWheel = (
      event: WheelEvent,
    ) => {
      const target =
        event.target;

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
    enabled,
    onCancelReply,
    replyPosting,
    resetReplyAnchor,
  ]);

  return {
    resetReplyAnchor,
  };
}