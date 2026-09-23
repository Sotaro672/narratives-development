// frontend/console/shell/src/shared/ui/preview.tsx

import * as React from "react";
import { createPortal } from "react-dom";
import {
  ChevronLeft,
  ChevronRight,
  X,
} from "lucide-react";

import { Button } from "./button";

import "./preview.css";

export type PreviewProps = {
  open: boolean;
  src?: string | null;
  alt?: string;
  onClose: () => void;
  onPrev?: () => void;
  onNext?: () => void;
};

export default function Preview({
  open,
  src,
  alt = "画像プレビュー",
  onClose,
  onPrev,
  onNext,
}: PreviewProps) {
  const previousActiveElementRef =
    React.useRef<HTMLElement | null>(null);
  const closeButtonRef =
    React.useRef<HTMLButtonElement | null>(null);

  const source = String(src ?? "").trim();

  React.useEffect(() => {
    if (!open || typeof document === "undefined") {
      return;
    }

    previousActiveElementRef.current =
      document.activeElement instanceof HTMLElement
        ? document.activeElement
        : null;

    const previousBodyOverflow =
      document.body.style.overflow;

    document.body.style.overflow = "hidden";

    const frame = window.requestAnimationFrame(() => {
      closeButtonRef.current?.focus();
    });

    return () => {
      window.cancelAnimationFrame(frame);
      document.body.style.overflow = previousBodyOverflow;

      previousActiveElementRef.current?.focus();
      previousActiveElementRef.current = null;
    };
  }, [open]);

  React.useEffect(() => {
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
        onPrev();
        return;
      }

      if (event.key === "ArrowRight" && onNext) {
        event.preventDefault();
        onNext();
      }
    };

    document.addEventListener("keydown", handleKeyDown);

    return () => {
      document.removeEventListener(
        "keydown",
        handleKeyDown,
      );
    };
  }, [
    open,
    onClose,
    onPrev,
    onNext,
  ]);

  if (
    !open ||
    !source ||
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
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) {
          onClose();
        }
      }}
    >
      <Button
        ref={closeButtonRef}
        type="button"
        variant="outline"
        size="icon"
        className="ui-preview__close"
        aria-label="プレビューを閉じる"
        onClick={onClose}
      >
        <X aria-hidden="true" />
      </Button>

      {onPrev ? (
        <Button
          type="button"
          variant="outline"
          size="icon"
          className="ui-preview__nav ui-preview__nav--prev"
          aria-label="前の画像"
          onClick={onPrev}
        >
          <ChevronLeft aria-hidden="true" />
        </Button>
      ) : null}

      <div
        className="ui-preview__content"
        onMouseDown={(event) => {
          event.stopPropagation();
        }}
      >
        <img
          src={source}
          alt={alt}
          className="ui-preview__image"
          draggable={false}
        />
      </div>

      {onNext ? (
        <Button
          type="button"
          variant="outline"
          size="icon"
          className="ui-preview__nav ui-preview__nav--next"
          aria-label="次の画像"
          onClick={onNext}
        >
          <ChevronRight aria-hidden="true" />
        </Button>
      ) : null}
    </div>,
    document.body,
  );
}