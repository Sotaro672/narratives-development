// frontend/mall/src/components/ui/Preview.tsx

import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type PointerEvent as ReactPointerEvent,
} from "react";
import { createPortal } from "react-dom";
import { ChevronLeft, ChevronRight, X } from "lucide-react";
import { QRCodeSVG } from "qrcode.react";

import "./Preview.css";

export type PreviewProps = {
  open: boolean;
  src?: string | null;
  alt?: string;
  type?: string;
  qrValue?: string | null;
  onClose: () => void;
  onPrev?: () => void;
  onNext?: () => void;
};

type Point = {
  x: number;
  y: number;
};

type PreviewTransform = {
  scale: number;
  x: number;
  y: number;
};

type DragOrigin = {
  pointerId: number;
  pointerX: number;
  pointerY: number;
  x: number;
  y: number;
};

const MIN_SCALE = 1;
const MAX_SCALE = 4;
const DOUBLE_TAP_SCALE = 2;

const INITIAL_TRANSFORM: PreviewTransform = {
  scale: MIN_SCALE,
  x: 0,
  y: 0,
};

function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max);
}

function getDistance(first: Point, second: Point): number {
  return Math.hypot(second.x - first.x, second.y - first.y);
}

export default function Preview({
  open,
  src,
  alt = "メディアプレビュー",
  type,
  qrValue,
  onClose,
  onPrev,
  onNext,
}: PreviewProps) {
  const contentRef = useRef<HTMLDivElement | null>(null);
  const imageRef = useRef<HTMLImageElement | null>(null);
  const closeButtonRef = useRef<HTMLButtonElement | null>(null);
  const previousActiveElementRef = useRef<HTMLElement | null>(null);
  const pointersRef = useRef<Map<number, Point>>(new Map());
  const pinchStartDistanceRef = useRef<number | null>(null);
  const pinchStartScaleRef = useRef(MIN_SCALE);
  const dragOriginRef = useRef<DragOrigin | null>(null);
  const transformRef = useRef<PreviewTransform>(INITIAL_TRANSFORM);

  const [transform, setTransform] = useState<PreviewTransform>(INITIAL_TRANSFORM);

  const source = String(src ?? "").trim();
  const qrPayload = String(qrValue ?? "").trim();
  const normalizedType = String(type ?? "").trim().toLowerCase();

  const isVideo =
    normalizedType === "video" ||
    normalizedType.startsWith("video/");
  const isQr = normalizedType === "qr";

  const applyTransform = useCallback((nextTransform: PreviewTransform) => {
    transformRef.current = nextTransform;
    setTransform(nextTransform);
  }, []);

  const clearGesture = useCallback(() => {
    pointersRef.current.clear();
    pinchStartDistanceRef.current = null;
    pinchStartScaleRef.current = MIN_SCALE;
    dragOriginRef.current = null;
  }, []);

  const resetTransform = useCallback(() => {
    clearGesture();
    applyTransform(INITIAL_TRANSFORM);
  }, [applyTransform, clearGesture]);

  const clampTransform = useCallback(
    (nextTransform: PreviewTransform): PreviewTransform => {
      const nextScale = clamp(nextTransform.scale, MIN_SCALE, MAX_SCALE);

      if (nextScale <= MIN_SCALE) {
        return INITIAL_TRANSFORM;
      }

      const content = contentRef.current;
      const image = imageRef.current;

      if (!content || !image) {
        return {
          scale: nextScale,
          x: nextTransform.x,
          y: nextTransform.y,
        };
      }

      const baseWidth = image.clientWidth;
      const baseHeight = image.clientHeight;
      const viewportWidth = content.clientWidth;
      const viewportHeight = content.clientHeight;

      const maxX = Math.max(0, (baseWidth * nextScale - viewportWidth) / 2);
      const maxY = Math.max(0, (baseHeight * nextScale - viewportHeight) / 2);

      return {
        scale: nextScale,
        x: clamp(nextTransform.x, -maxX, maxX),
        y: clamp(nextTransform.y, -maxY, maxY),
      };
    },
    [],
  );

  useEffect(() => {
    if (!open || typeof document === "undefined") {
      return;
    }

    previousActiveElementRef.current =
      document.activeElement instanceof HTMLElement
        ? document.activeElement
        : null;

    const previousBodyOverflow = document.body.style.overflow;
    const previousBodyOverscrollBehavior = document.body.style.overscrollBehavior;

    document.body.style.overflow = "hidden";
    document.body.style.overscrollBehavior = "none";

    const frameId = window.requestAnimationFrame(() => {
      closeButtonRef.current?.focus();
    });

    return () => {
      window.cancelAnimationFrame(frameId);
      document.body.style.overflow = previousBodyOverflow;
      document.body.style.overscrollBehavior = previousBodyOverscrollBehavior;
      previousActiveElementRef.current?.focus();
      previousActiveElementRef.current = null;
    };
  }, [open]);

  useEffect(() => {
    if (!open) {
      return;
    }

    resetTransform();

    const handleResize = () => {
      resetTransform();
    };

    window.addEventListener("resize", handleResize);

    return () => {
      window.removeEventListener("resize", handleResize);
      clearGesture();
    };
  }, [open, source, qrPayload, clearGesture, resetTransform]);

  useEffect(() => {
    if (!open || typeof document === "undefined") {
      return;
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault();
        onClose();
        return;
      }

      if (event.key === "ArrowLeft" && onPrev) {
        event.preventDefault();
        resetTransform();
        onPrev();
        return;
      }

      if (event.key === "ArrowRight" && onNext) {
        event.preventDefault();
        resetTransform();
        onNext();
      }
    };

    document.addEventListener("keydown", handleKeyDown);

    return () => {
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [open, onClose, onPrev, onNext, resetTransform]);

  const handlePointerDown = (
    event: ReactPointerEvent<HTMLDivElement>,
  ): void => {
    if (isVideo || isQr) {
      return;
    }

    if (event.pointerType === "mouse" && event.button !== 0) {
      return;
    }

    event.preventDefault();

    pointersRef.current.set(event.pointerId, {
      x: event.clientX,
      y: event.clientY,
    });

    if (!event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.setPointerCapture(event.pointerId);
    }

    const pointers = Array.from(pointersRef.current.entries());

    if (pointers.length === 1) {
      const [pointerId, point] = pointers[0];

      dragOriginRef.current = {
        pointerId,
        pointerX: point.x,
        pointerY: point.y,
        x: transformRef.current.x,
        y: transformRef.current.y,
      };

      return;
    }

    if (pointers.length >= 2) {
      const first = pointers[0][1];
      const second = pointers[1][1];

      pinchStartDistanceRef.current = getDistance(first, second);
      pinchStartScaleRef.current = transformRef.current.scale;
      dragOriginRef.current = null;
    }
  };

  const handlePointerMove = (
    event: ReactPointerEvent<HTMLDivElement>,
  ): void => {
    if (
      isVideo ||
      isQr ||
      !pointersRef.current.has(event.pointerId)
    ) {
      return;
    }

    event.preventDefault();

    pointersRef.current.set(event.pointerId, {
      x: event.clientX,
      y: event.clientY,
    });

    const pointers = Array.from(pointersRef.current.entries());

    if (
      pointers.length >= 2 &&
      pinchStartDistanceRef.current &&
      pinchStartDistanceRef.current > 0
    ) {
      const first = pointers[0][1];
      const second = pointers[1][1];
      const currentDistance = getDistance(first, second);
      const ratio = currentDistance / pinchStartDistanceRef.current;

      const nextScale = clamp(
        pinchStartScaleRef.current * ratio,
        MIN_SCALE,
        MAX_SCALE,
      );

      applyTransform(
        clampTransform({
          scale: nextScale,
          x: transformRef.current.x,
          y: transformRef.current.y,
        }),
      );

      return;
    }

    const dragOrigin = dragOriginRef.current;

    if (
      pointers.length !== 1 ||
      !dragOrigin ||
      dragOrigin.pointerId !== event.pointerId ||
      transformRef.current.scale <= MIN_SCALE
    ) {
      return;
    }

    applyTransform(
      clampTransform({
        scale: transformRef.current.scale,
        x: dragOrigin.x + event.clientX - dragOrigin.pointerX,
        y: dragOrigin.y + event.clientY - dragOrigin.pointerY,
      }),
    );
  };

  const handlePointerEnd = (
    event: ReactPointerEvent<HTMLDivElement>,
  ): void => {
    if (isVideo || isQr) {
      return;
    }

    pointersRef.current.delete(event.pointerId);

    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId);
    }

    const pointers = Array.from(pointersRef.current.entries());

    if (pointers.length === 0) {
      pinchStartDistanceRef.current = null;
      dragOriginRef.current = null;

      applyTransform(clampTransform(transformRef.current));
      return;
    }

    if (pointers.length === 1) {
      const [pointerId, point] = pointers[0];

      pinchStartDistanceRef.current = null;
      dragOriginRef.current = {
        pointerId,
        pointerX: point.x,
        pointerY: point.y,
        x: transformRef.current.x,
        y: transformRef.current.y,
      };

      return;
    }

    const first = pointers[0][1];
    const second = pointers[1][1];

    pinchStartDistanceRef.current = getDistance(first, second);
    pinchStartScaleRef.current = transformRef.current.scale;
    dragOriginRef.current = null;
  };

  const handleDoubleClick = (): void => {
    if (isVideo || isQr) {
      return;
    }

    if (transformRef.current.scale > MIN_SCALE) {
      resetTransform();
      return;
    }

    applyTransform(
      clampTransform({
        scale: DOUBLE_TAP_SCALE,
        x: 0,
        y: 0,
      }),
    );
  };

  const handlePrev = (): void => {
    resetTransform();
    onPrev?.();
  };

  const handleNext = (): void => {
    resetTransform();
    onNext?.();
  };

  const hasContent = isQr ? Boolean(qrPayload) : Boolean(source);

  if (
    !open ||
    !hasContent ||
    typeof document === "undefined"
  ) {
    return null;
  }

  return createPortal(
    <div
      className="ui-preview"
      role="dialog"
      aria-modal="true"
      aria-label={alt}
    >
      <button
        ref={closeButtonRef}
        type="button"
        className="ui-preview__control ui-preview__close"
        aria-label="プレビューを閉じる"
        onClick={onClose}
      >
        <X aria-hidden="true" />
      </button>

      {onPrev ? (
        <button
          type="button"
          className="ui-preview__control ui-preview__nav ui-preview__nav--prev"
          aria-label="前のメディアを表示"
          onClick={handlePrev}
        >
          <ChevronLeft aria-hidden="true" />
        </button>
      ) : null}

      <div
        ref={contentRef}
        className={[
          "ui-preview__content",
          isQr
            ? "ui-preview__content--qr"
            : isVideo
              ? "ui-preview__content--video"
              : "ui-preview__content--image",
        ].join(" ")}
        onPointerDown={(event) => {
          if (event.target === event.currentTarget) {
            onClose();
            return;
          }

          handlePointerDown(event);
        }}
        onPointerMove={handlePointerMove}
        onPointerUp={handlePointerEnd}
        onPointerCancel={handlePointerEnd}
        onDoubleClick={handleDoubleClick}
      >
        {isQr ? (
          <div
            className="ui-preview__qr-frame"
            role="img"
            aria-label={alt}
          >
            <QRCodeSVG
              className="ui-preview__qr"
              value={qrPayload}
              size={320}
              level="M"
              includeMargin
            />

            <span className="ui-preview__qr-label">
              PUDO MOCK
            </span>
          </div>
        ) : isVideo ? (
          <video
            src={source}
            className="ui-preview__video"
            controls
            playsInline
            autoPlay
            preload="metadata"
            aria-label={alt}
          />
        ) : (
          <img
            ref={imageRef}
            src={source}
            alt={alt}
            className={[
              "ui-preview__image",
              transform.scale > MIN_SCALE
                ? "ui-preview__image--zoomed"
                : "",
            ]
              .filter(Boolean)
              .join(" ")}
            style={{
              transform: `translate3d(${transform.x}px, ${transform.y}px, 0) scale(${transform.scale})`,
            }}
            draggable={false}
            onLoad={resetTransform}
          />
        )}
      </div>

      {onNext ? (
        <button
          type="button"
          className="ui-preview__control ui-preview__nav ui-preview__nav--next"
          aria-label="次のメディアを表示"
          onClick={handleNext}
        >
          <ChevronRight aria-hidden="true" />
        </button>
      ) : null}
    </div>,
    document.body,
  );
}