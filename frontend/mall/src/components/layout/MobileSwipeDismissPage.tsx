// frontend/mall/src/components/layout/MobileSwipeDismissPage.tsx

import {
  forwardRef,
  useCallback,
  useEffect,
  useImperativeHandle,
  useRef,
  useState,
  type CSSProperties,
  type MouseEvent as ReactMouseEvent,
  type ReactNode,
} from "react";

import "./mobile-swipe-dismiss-page.css";

type MobileSwipeDismissPageProps = {
  children: ReactNode;
  enabled?: boolean;
  dismissGestureEnabled?: boolean;
  className?: string;
  onDismissStart?: () => void;
  onDismiss: () => void | Promise<void>;
};

export type MobileSwipeDismissPageHandle = {
  dismiss: () => void;
};

type GestureState = {
  active: boolean;
  dragging: boolean;
  blocked: boolean;
  startX: number;
  startY: number;
  startTime: number;
  lastY: number;
  lastTime: number;
  velocityY: number;
};

const DIRECTION_LOCK_DISTANCE = 8;
const DISMISS_DISTANCE = 120;
const MIN_FAST_DISMISS_DISTANCE = 32;
const DISMISS_VELOCITY = 0.65;
const RETURN_ANIMATION_MS = 220;
const DISMISS_ANIMATION_MS = 180;

const INITIAL_GESTURE: GestureState = {
  active: false,
  dragging: false,
  blocked: false,
  startX: 0,
  startY: 0,
  startTime: 0,
  lastY: 0,
  lastTime: 0,
  velocityY: 0,
};

function isScrollableElement(element: HTMLElement): boolean {
  const style = window.getComputedStyle(element);
  const overflowY = style.overflowY;

  if (
    overflowY !== "auto" &&
    overflowY !== "scroll" &&
    overflowY !== "overlay"
  ) {
    return false;
  }

  return element.scrollHeight > element.clientHeight + 1;
}

function hasScrolledAncestor(
  target: EventTarget | null,
  boundary: HTMLElement,
): boolean {
  let element = target instanceof HTMLElement ? target : null;

  while (element && element !== boundary) {
    if (isScrollableElement(element) && element.scrollTop > 0) {
      return true;
    }

    element = element.parentElement;
  }

  return boundary.scrollTop > 0;
}

function shouldIgnoreGesture(target: EventTarget | null): boolean {
  if (!(target instanceof Element)) {
    return false;
  }

  return Boolean(
    target.closest('[data-mobile-swipe-dismiss-ignore="true"]'),
  );
}

const MobileSwipeDismissPage = forwardRef<
  MobileSwipeDismissPageHandle,
  MobileSwipeDismissPageProps
>(function MobileSwipeDismissPage(
  {
    children,
    enabled = true,
    dismissGestureEnabled = true,
    className,
    onDismissStart,
    onDismiss,
  },
  ref,
) {
  const pageRef = useRef<HTMLDivElement | null>(null);
  const gestureRef = useRef<GestureState>({ ...INITIAL_GESTURE });
  const dismissStartedRef = useRef(false);
  const dismissCompletedRef = useRef(false);
  const suppressClickRef = useRef(false);
  const animationTimerRef = useRef<number | null>(null);

  const [translateY, setTranslateY] = useState(0);
  const [phase, setPhase] = useState<
    "idle" | "dragging" | "returning" | "dismissing"
  >("idle");

  const clearAnimationTimer = useCallback(() => {
    if (animationTimerRef.current === null) {
      return;
    }

    window.clearTimeout(animationTimerRef.current);
    animationTimerRef.current = null;
  }, []);

  const resetGesture = useCallback(() => {
    gestureRef.current = { ...INITIAL_GESTURE };
    dismissStartedRef.current = false;
  }, []);

  const notifyDismissStart = useCallback(() => {
    if (dismissStartedRef.current) {
      return;
    }

    dismissStartedRef.current = true;
    onDismissStart?.();
  }, [onDismissStart]);

  const finishDismiss = useCallback(() => {
    if (dismissCompletedRef.current) {
      return;
    }

    dismissCompletedRef.current = true;
    void onDismiss();
  }, [onDismiss]);

  const returnToStart = useCallback(() => {
    clearAnimationTimer();
    setPhase("returning");
    setTranslateY(0);

    animationTimerRef.current = window.setTimeout(() => {
      animationTimerRef.current = null;
      setPhase("idle");
      resetGesture();
    }, RETURN_ANIMATION_MS);
  }, [clearAnimationTimer, resetGesture]);

  const dismissPage = useCallback(() => {
    const page = pageRef.current;

    if (!page) {
      finishDismiss();
      return;
    }

    clearAnimationTimer();

    gestureRef.current.active = false;
    gestureRef.current.dragging = false;

    const dismissDistance =
      Math.max(page.clientHeight, window.innerHeight) + 32;

    setPhase("dismissing");
    setTranslateY(dismissDistance);

    animationTimerRef.current = window.setTimeout(() => {
      animationTimerRef.current = null;
      finishDismiss();
    }, DISMISS_ANIMATION_MS);
  }, [
    clearAnimationTimer,
    finishDismiss,
  ]);

  const requestDismiss = useCallback(() => {
    if (
      !enabled ||
      phase === "dismissing"
    ) {
      return;
    }

    notifyDismissStart();
    dismissPage();
  }, [
    dismissPage,
    enabled,
    notifyDismissStart,
    phase,
  ]);

  useImperativeHandle(
    ref,
    () => ({
      dismiss: requestDismiss,
    }),
    [requestDismiss],
  );

  useEffect(() => {
    if (!enabled) {
      clearAnimationTimer();
      resetGesture();
      dismissCompletedRef.current = false;
      suppressClickRef.current = false;
      setTranslateY(0);
      setPhase("idle");
      return;
    }

    dismissCompletedRef.current = false;
  }, [
    clearAnimationTimer,
    enabled,
    resetGesture,
  ]);

  useEffect(() => {
    if (
      !enabled ||
      dismissGestureEnabled
    ) {
      return;
    }

    clearAnimationTimer();
    resetGesture();
    suppressClickRef.current = false;
    setTranslateY(0);
    setPhase("idle");
  }, [
    clearAnimationTimer,
    dismissGestureEnabled,
    enabled,
    resetGesture,
  ]);

  useEffect(() => {
    if (
      !enabled ||
      typeof document === "undefined"
    ) {
      return;
    }

    const previousOverflow =
      document.body.style.overflow;

    const previousOverscrollBehavior =
      document.body.style.overscrollBehavior;

    document.body.style.overflow = "hidden";
    document.body.style.overscrollBehavior = "none";

    return () => {
      document.body.style.overflow =
        previousOverflow;

      document.body.style.overscrollBehavior =
        previousOverscrollBehavior;
    };
  }, [enabled]);

  useEffect(() => {
    const page = pageRef.current;

    if (
      !page ||
      !enabled ||
      !dismissGestureEnabled
    ) {
      return;
    }

    const handleTouchStart = (event: TouchEvent) => {
      if (
        event.touches.length !== 1 ||
        phase === "dismissing" ||
        phase === "returning" ||
        shouldIgnoreGesture(event.target)
      ) {
        resetGesture();
        return;
      }

      const touch = event.touches[0];

      if (!touch) {
        resetGesture();
        return;
      }

      const blocked =
        hasScrolledAncestor(
          event.target,
          page,
        );

      const now =
        performance.now();

      gestureRef.current = {
        active: true,
        dragging: false,
        blocked,
        startX: touch.clientX,
        startY: touch.clientY,
        startTime: now,
        lastY: touch.clientY,
        lastTime: now,
        velocityY: 0,
      };

      dismissStartedRef.current = false;
      suppressClickRef.current = false;
    };

    const handleTouchMove = (event: TouchEvent) => {
      const gesture =
        gestureRef.current;

      if (
        !gesture.active ||
        gesture.blocked ||
        event.touches.length !== 1
      ) {
        return;
      }

      const touch =
        event.touches[0];

      if (!touch) {
        return;
      }

      const deltaX =
        touch.clientX -
        gesture.startX;

      const deltaY =
        touch.clientY -
        gesture.startY;

      if (!gesture.dragging) {
        const distance =
          Math.hypot(
            deltaX,
            deltaY,
          );

        if (
          distance <
          DIRECTION_LOCK_DISTANCE
        ) {
          return;
        }

        const isDownward =
          deltaY > 0;

        const isVertical =
          Math.abs(deltaY) >
          Math.abs(deltaX);

        if (
          !isDownward ||
          !isVertical
        ) {
          gesture.blocked = true;
          return;
        }

        if (
          hasScrolledAncestor(
            event.target,
            page,
          )
        ) {
          gesture.blocked = true;
          return;
        }

        gesture.dragging = true;
        suppressClickRef.current = true;

        notifyDismissStart();
        setPhase("dragging");
      }

      if (!gesture.dragging) {
        return;
      }

      event.preventDefault();

      const now =
        performance.now();

      const elapsed =
        Math.max(
          1,
          now - gesture.lastTime,
        );

      const movement =
        touch.clientY -
        gesture.lastY;

      gesture.velocityY =
        movement / elapsed;

      gesture.lastY =
        touch.clientY;

      gesture.lastTime =
        now;

      setTranslateY(
        Math.max(
          0,
          deltaY,
        ),
      );
    };

    const handleTouchEnd = (event: TouchEvent) => {
      const gesture =
        gestureRef.current;

      if (
        !gesture.active ||
        !gesture.dragging
      ) {
        resetGesture();
        return;
      }

      const touch =
        event.changedTouches[0];

      const endY =
        touch?.clientY ??
        gesture.lastY;

      const distance =
        Math.max(
          0,
          endY -
            gesture.startY,
        );

      const totalElapsed =
        Math.max(
          1,
          performance.now() -
            gesture.startTime,
        );

      const averageVelocity =
        distance /
        totalElapsed;

      const velocity =
        Math.max(
          gesture.velocityY,
          averageVelocity,
        );

      const shouldDismiss =
        distance >=
          DISMISS_DISTANCE ||
        (
          distance >=
            MIN_FAST_DISMISS_DISTANCE &&
          velocity >=
            DISMISS_VELOCITY
        );

      gesture.active = false;
      gesture.dragging = false;

      if (shouldDismiss) {
        dismissPage();
        return;
      }

      returnToStart();
    };

    const handleTouchCancel = () => {
      const gesture =
        gestureRef.current;

      if (gesture.dragging) {
        returnToStart();
        return;
      }

      resetGesture();
    };

    page.addEventListener(
      "touchstart",
      handleTouchStart,
      {
        passive: true,
      },
    );

    page.addEventListener(
      "touchmove",
      handleTouchMove,
      {
        passive: false,
      },
    );

    page.addEventListener(
      "touchend",
      handleTouchEnd,
      {
        passive: true,
      },
    );

    page.addEventListener(
      "touchcancel",
      handleTouchCancel,
      {
        passive: true,
      },
    );

    return () => {
      page.removeEventListener(
        "touchstart",
        handleTouchStart,
      );

      page.removeEventListener(
        "touchmove",
        handleTouchMove,
      );

      page.removeEventListener(
        "touchend",
        handleTouchEnd,
      );

      page.removeEventListener(
        "touchcancel",
        handleTouchCancel,
      );
    };
  }, [
    dismissGestureEnabled,
    dismissPage,
    enabled,
    notifyDismissStart,
    phase,
    resetGesture,
    returnToStart,
  ]);

  useEffect(() => {
    return () => {
      clearAnimationTimer();
    };
  }, [clearAnimationTimer]);

  const handleClickCapture = (
    event: ReactMouseEvent<HTMLDivElement>,
  ) => {
    if (!suppressClickRef.current) {
      return;
    }

    event.preventDefault();
    event.stopPropagation();
    suppressClickRef.current = false;
  };

  const style: CSSProperties | undefined =
    enabled
      ? {
          transform: `translate3d(0, ${translateY}px, 0)`,
        }
      : undefined;

  const rootClassName = [
    "mobile-swipe-dismiss-page",
    enabled
      ? "mobile-swipe-dismiss-page--enabled"
      : "",
    phase === "dragging"
      ? "mobile-swipe-dismiss-page--dragging"
      : "",
    phase === "returning"
      ? "mobile-swipe-dismiss-page--returning"
      : "",
    phase === "dismissing"
      ? "mobile-swipe-dismiss-page--dismissing"
      : "",
    className ?? "",
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div
      ref={pageRef}
      className={rootClassName}
      style={style}
      onClickCapture={handleClickCapture}
      data-swipe-dismiss-phase={phase}
      data-swipe-dismiss-gesture-enabled={
        dismissGestureEnabled
          ? "true"
          : "false"
      }
    >
      {children}
    </div>
  );
});

export default MobileSwipeDismissPage;