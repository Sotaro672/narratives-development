// frontend/console/shell/src/shared/ui/modal.tsx

import * as React from "react";
import { createPortal } from "react-dom";
import { X } from "lucide-react";

import { Button, type ButtonProps } from "./button";

import "./modal.css";

function cn(...classes: Array<string | undefined | false | null>): string {
  return classes.filter(Boolean).join(" ");
}

const FOCUSABLE_SELECTOR = [
  "a[href]",
  "button:not([disabled])",
  "input:not([disabled])",
  "select:not([disabled])",
  "textarea:not([disabled])",
  '[tabindex]:not([tabindex="-1"])',
].join(",");

export type ModalProps = {
  open: boolean;
  title: React.ReactNode;
  description?: React.ReactNode;
  eyebrow?: React.ReactNode;
  children?: React.ReactNode;
  footer?: React.ReactNode;
  onClose?: () => void;
  closeable?: boolean;
  closeOnBackdrop?: boolean;
  closeOnEscape?: boolean;
  showCloseButton?: boolean;
  closeLabel?: string;
  ariaBusy?: boolean;
  className?: string;
  panelClassName?: string;
  headerClassName?: string;
  bodyClassName?: string;
  footerClassName?: string;
};

export function Modal({
  open,
  title,
  description,
  eyebrow,
  children,
  footer,
  onClose,
  closeable = true,
  closeOnBackdrop = true,
  closeOnEscape = true,
  showCloseButton = true,
  closeLabel = "モーダルを閉じる",
  ariaBusy = false,
  className,
  panelClassName,
  headerClassName,
  bodyClassName,
  footerClassName,
}: ModalProps) {
  const titleId = React.useId();
  const descriptionId = React.useId();
  const panelRef = React.useRef<HTMLDivElement | null>(null);
  const previousActiveElementRef = React.useRef<HTMLElement | null>(null);

  const canClose = closeable && Boolean(onClose);

  const requestClose = React.useCallback(() => {
    if (!canClose) return;
    onClose?.();
  }, [canClose, onClose]);

  React.useEffect(() => {
    if (!open || typeof document === "undefined") return;

    previousActiveElementRef.current =
      document.activeElement instanceof HTMLElement
        ? document.activeElement
        : null;

    const previousBodyOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";

    const frame = window.requestAnimationFrame(() => {
      panelRef.current?.focus();
    });

    return () => {
      window.cancelAnimationFrame(frame);
      document.body.style.overflow = previousBodyOverflow;
      previousActiveElementRef.current?.focus();
      previousActiveElementRef.current = null;
    };
  }, [open]);

  React.useEffect(() => {
    if (!open || typeof document === "undefined") return;

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        if (!canClose || !closeOnEscape) return;

        event.preventDefault();
        requestClose();
        return;
      }

      if (event.key !== "Tab") return;

      const panel = panelRef.current;
      if (!panel) return;

      const focusableElements = Array.from(
        panel.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR),
      ).filter((element) => {
        return !element.hasAttribute("disabled") && element.tabIndex !== -1;
      });

      if (focusableElements.length === 0) {
        event.preventDefault();
        panel.focus();
        return;
      }

      const firstElement = focusableElements[0];
      const lastElement = focusableElements[focusableElements.length - 1];
      const activeElement = document.activeElement;

      if (event.shiftKey && activeElement === firstElement) {
        event.preventDefault();
        lastElement.focus();
        return;
      }

      if (!event.shiftKey && activeElement === lastElement) {
        event.preventDefault();
        firstElement.focus();
      }
    };

    document.addEventListener("keydown", handleKeyDown);

    return () => {
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [open, canClose, closeOnEscape, requestClose]);

  if (!open || typeof document === "undefined") {
    return null;
  }

  const modal = (
    <div
      className={cn("modal", className)}
      role="presentation"
      onMouseDown={(event) => {
        if (
          event.target === event.currentTarget &&
          canClose &&
          closeOnBackdrop
        ) {
          requestClose();
        }
      }}
    >
      <div
        ref={panelRef}
        className={cn("modal__panel", panelClassName)}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={description ? descriptionId : undefined}
        aria-busy={ariaBusy}
        tabIndex={-1}
      >
        <div className={cn("modal__header", headerClassName)}>
          <div className="modal__heading">
            {eyebrow ? (
              <div className="modal__eyebrow">
                {eyebrow}
              </div>
            ) : null}

            <h2 id={titleId} className="modal__title">
              {title}
            </h2>
          </div>

          {canClose && showCloseButton ? (
            <Button
              variant="outline"
              size="icon"
              onClick={requestClose}
              aria-label={closeLabel}
            >
              <X className="modal__close-icon" aria-hidden="true" />
            </Button>
          ) : null}
        </div>

        <div className={cn("modal__body", bodyClassName)}>
          {description ? (
            <p id={descriptionId} className="modal__description">
              {description}
            </p>
          ) : null}

          {children}
        </div>

        {footer ? (
          <div className={cn("modal__footer", footerClassName)}>
            {footer}
          </div>
        ) : null}
      </div>
    </div>
  );

  return createPortal(modal, document.body);
}

export type ModalButtonProps = Omit<
  ButtonProps,
  "variant" | "size" | "type"
> & {
  variant?: "primary" | "secondary";
};

function resolveModalButtonVariant(
  variant: NonNullable<ModalButtonProps["variant"]>,
): ButtonProps["variant"] {
  switch (variant) {
    case "primary":
      return "default";
    case "secondary":
    default:
      return "outline";
  }
}

export function ModalButton({
  variant = "secondary",
  children,
  ...props
}: ModalButtonProps) {
  return (
    <Button
      type="button"
      variant={resolveModalButtonVariant(variant)}
      size="lg"
      {...props}
    >
      {children}
    </Button>
  );
}

export type ModalCloseButtonProps = Omit<
  ModalButtonProps,
  "variant"
>;

export function ModalCloseButton({
  children = "閉じる",
  ...props
}: ModalCloseButtonProps) {
  return (
    <ModalButton
      variant="secondary"
      {...props}
    >
      {children}
    </ModalButton>
  );
}

export default Modal;