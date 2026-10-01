// frontend/mall/src/components/layout/MobileSwipeRightDismissPage.tsx

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
import { ChevronLeft } from "lucide-react";
import { useNavigate } from "react-router-dom";

import IconButton from "../ui/IconButton";

import "./mobileSwipeRightDismissPage.css";

type MobileSwipeRightDismissPageProps = {
  children: ReactNode;
  title: string;
  enabled?: boolean;
  dismissGestureEnabled?: boolean;
  className?: string;
  onDismissStart?: () => void;
  onDismiss: () => void | Promise<void>;
};

export type MobileSwipeRightDismissPageHandle = {
  dismiss: () => void;
};

type GestureState = {
  active: boolean;
  dragging: boolean;
  blocked: boolean;
  startX: number;
  startY: number;
  startTime: number;
  lastX: number;
  lastTime: number;
  velocityX: number;
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
  lastX: 0,
  lastTime: 0,
  velocityX: 0,
};

function isHorizontallyScrollableElement(element: HTMLElement): boolean {
  const style = window.getComputedStyle(element);
  const overflowX = style.overflowX;

  if (
    overflowX !== "auto" &&
    overflowX !== "scroll" &&
    overflowX !== "overlay"
  ) {
    return false;
  }

  return element.scrollWidth > element.clientWidth + 1;
}

function hasHorizontallyScrolledAncestor(
  target: EventTarget | null,
  boundary: HTMLElement,
): boolean {
  let element = target instanceof HTMLElement ? target : null;

  while (element && element !== boundary) {
    if (
      isHorizontallyScrollableElement(element) &&
      element.scrollLeft > 0
    ) {
      return true;
    }

    element = element.parentElement;
  }

  return false;
}

function shouldIgnoreGesture(target: EventTarget | null): boolean {
  if (!(target instanceof Element)) {
    return false;
  }

  return Boolean(
    target.closest('[data-mobile-swipe-dismiss-ignore="true"]'),
  );
}

const MobileSwipeRightDismissPage = forwardRef<
  MobileSwipeRightDismissPageHandle,
  MobileSwipeRightDismissPageProps
>(function MobileSwipeRightDismissPage(
  {
    children,
    title,
    enabled = true,
    dismissGestureEnabled = true,
    className,
    onDismissStart,
    onDismiss,
  },
  ref,
) {
  const navigate = useNavigate();
  const pageRef = useRef<HTMLDivElement | null>(null);
  const gestureRef = useRef<GestureState>({ ...INITIAL_GESTURE });
  const dismissStartedRef = useRef(false);
  const dismissCompletedRef = useRef(false);
  const suppressClickRef = useRef(false);
  const animationTimerRef = useRef<number | null>(null);

  const [translateX, setTranslateX] = useState(0);
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
    setTranslateX(0);

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
      Math.max(page.clientWidth, window.innerWidth) + 32;

    setPhase("dismissing");
    setTranslateX(dismissDistance);

    animationTimerRef.current = window.setTimeout(() => {
      animationTimerRef.current = null;
      finishDismiss();
    }, DISMISS_ANIMATION_MS);
  }, [clearAnimationTimer, finishDismiss]);

  const requestDismiss = useCallback(() => {
    if (!enabled || phase === "dismissing") {
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
      setTranslateX(0);
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
    if (!enabled || dismissGestureEnabled) {
      return;
    }

    clearAnimationTimer();
    resetGesture();
    suppressClickRef.current = false;
    setTranslateX(0);
    setPhase("idle");
  }, [
    clearAnimationTimer,
    dismissGestureEnabled,
    enabled,
    resetGesture,
  ]);

  useEffect(() => {
    if (!enabled || typeof document === "undefined") {
      return;
    }

    const previousOverflow = document.body.style.overflow;
    const previousOverscrollBehavior =
      document.body.style.overscrollBehavior;

    document.body.style.overflow = "hidden";
    document.body.style.overscrollBehavior = "none";

    return () => {
      document.body.style.overflow = previousOverflow;
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

      const now = performance.now();

      gestureRef.current = {
        active: true,
        dragging: false,
        blocked: false,
        startX: touch.clientX,
        startY: touch.clientY,
        startTime: now,
        lastX: touch.clientX,
        lastTime: now,
        velocityX: 0,
      };

      dismissStartedRef.current = false;
      suppressClickRef.current = false;
    };

    const handleTouchMove = (event: TouchEvent) => {
      const gesture = gestureRef.current;

      if (
        !gesture.active ||
        gesture.blocked ||
        event.touches.length !== 1
      ) {
        return;
      }

      const touch = event.touches[0];

      if (!touch) {
        return;
      }

      const deltaX = touch.clientX - gesture.startX;
      const deltaY = touch.clientY - gesture.startY;

      if (!gesture.dragging) {
        const distance = Math.hypot(deltaX, deltaY);

        if (distance < DIRECTION_LOCK_DISTANCE) {
          return;
        }

        const isRightward = deltaX > 0;
        const isHorizontal =
          Math.abs(deltaX) > Math.abs(deltaY);

        if (!isRightward || !isHorizontal) {
          gesture.blocked = true;
          return;
        }

        if (
          hasHorizontallyScrolledAncestor(
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

      const now = performance.now();
      const elapsed = Math.max(
        1,
        now - gesture.lastTime,
      );

      const movement =
        touch.clientX - gesture.lastX;

      gesture.velocityX =
        movement / elapsed;

      gesture.lastX = touch.clientX;
      gesture.lastTime = now;

      setTranslateX(
        Math.max(
          0,
          deltaX,
        ),
      );
    };

    const handleTouchEnd = (event: TouchEvent) => {
      const gesture = gestureRef.current;

      if (
        !gesture.active ||
        !gesture.dragging
      ) {
        resetGesture();
        return;
      }

      const touch = event.changedTouches[0];
      const endX = touch?.clientX ?? gesture.lastX;
      const distance = Math.max(
        0,
        endX - gesture.startX,
      );

      const totalElapsed = Math.max(
        1,
        performance.now() - gesture.startTime,
      );

      const averageVelocity =
        distance / totalElapsed;

      const velocity = Math.max(
        gesture.velocityX,
        averageVelocity,
      );

      const shouldDismiss =
        distance >= DISMISS_DISTANCE ||
        (
          distance >= MIN_FAST_DISMISS_DISTANCE &&
          velocity >= DISMISS_VELOCITY
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
      const gesture = gestureRef.current;

      if (gesture.dragging) {
        returnToStart();
        return;
      }

      resetGesture();
    };

    page.addEventListener(
      "touchstart",
      handleTouchStart,
      { passive: true },
    );

    page.addEventListener(
      "touchmove",
      handleTouchMove,
      { passive: false },
    );

    page.addEventListener(
      "touchend",
      handleTouchEnd,
      { passive: true },
    );

    page.addEventListener(
      "touchcancel",
      handleTouchCancel,
      { passive: true },
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

  const handleBackClick = () => {
    navigate(-1);
  };

  const shouldTransform =
    enabled &&
    phase !== "idle";

  const style: CSSProperties | undefined =
    shouldTransform
      ? {
          transform: `translate3d(${translateX}px, 0, 0)`,
        }
      : undefined;

  const rootClassName = [
    "mobile-swipe-right-dismiss-page",
    enabled
      ? "mobile-swipe-right-dismiss-page--enabled"
      : "",
    phase === "dragging"
      ? "mobile-swipe-right-dismiss-page--dragging"
      : "",
    phase === "returning"
      ? "mobile-swipe-right-dismiss-page--returning"
      : "",
    phase === "dismissing"
      ? "mobile-swipe-right-dismiss-page--dismissing"
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
      <header
        className="mobile-swipe-right-dismiss-page__header"
        data-mobile-swipe-dismiss-ignore="true"
      >
        <IconButton
          type="button"
          variant="ghost"
          size="md"
          className="mobile-swipe-right-dismiss-page__back-button"
          aria-label="戻る"
          title="戻る"
          onClick={handleBackClick}
        >
          <ChevronLeft
            size={24}
            strokeWidth={2}
            aria-hidden="true"
          />
        </IconButton>

        <h1 className="mobile-swipe-right-dismiss-page__title">
          {title}
        </h1>
      </header>

      <div className="mobile-swipe-right-dismiss-page__content">
        {children}
      </div>
    </div>
  );
});

export default MobileSwipeRightDismissPage;