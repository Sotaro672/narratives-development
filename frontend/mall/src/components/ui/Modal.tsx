import {
  useEffect,
  type HTMLAttributes,
  type ReactNode,
} from "react";
import { createPortal } from "react-dom";

import IconButton from "./IconButton";

import "./Modal.css";

type ModalSize = "sm" | "md" | "lg";
type ModalMobilePosition = "center" | "bottom";

type ModalProps = {
  open: boolean;
  children: ReactNode;
  onClose?: () => void;
  size?: ModalSize;
  mobilePosition?: ModalMobilePosition;
  closeOnBackdrop?: boolean;
  closeOnEscape?: boolean;
  ariaLabel?: string;
  ariaLabelledBy?: string;
  ariaDescribedBy?: string;
  ariaBusy?: boolean;
  className?: string;
  panelClassName?: string;
};

type ModalSectionProps = HTMLAttributes<HTMLDivElement> & {
  children: ReactNode;
};

type ModalHeaderProps = ModalSectionProps & {
  onClose?: () => void;
  closeLabel?: string;
};

type ModalTitleProps = HTMLAttributes<HTMLHeadingElement> & {
  children: ReactNode;
};

type ModalDescriptionProps = HTMLAttributes<HTMLParagraphElement> & {
  children: ReactNode;
};

export default function Modal({
  open,
  children,
  onClose,
  size = "md",
  mobilePosition = "center",
  closeOnBackdrop = true,
  closeOnEscape = true,
  ariaLabel,
  ariaLabelledBy,
  ariaDescribedBy,
  ariaBusy,
  className = "",
  panelClassName = "",
}: ModalProps) {
  useEffect(() => {
    if (!open || !onClose || !closeOnEscape) {
      return;
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "Escape") {
        return;
      }

      event.preventDefault();
      onClose();
    };

    document.addEventListener("keydown", handleKeyDown);

    return () => {
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [open, onClose, closeOnEscape]);

  useEffect(() => {
    if (!open) {
      return;
    }

    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";

    return () => {
      document.body.style.overflow = previousOverflow;
    };
  }, [open]);

  if (!open) {
    return null;
  }

  const overlayClasses = [
    "ui-modal",
    `ui-modal--mobile-${mobilePosition}`,
    className,
  ]
    .filter(Boolean)
    .join(" ");

  const panelClasses = [
    "ui-modal__panel",
    `ui-modal__panel--${size}`,
    panelClassName,
  ]
    .filter(Boolean)
    .join(" ");

  const modal = (
    <div
      className={overlayClasses}
      role="presentation"
      onMouseDown={(event) => {
        if (
          event.target === event.currentTarget &&
          closeOnBackdrop &&
          onClose
        ) {
          onClose();
        }
      }}
    >
      <div
        className={panelClasses}
        role="dialog"
        aria-modal="true"
        aria-label={ariaLabel}
        aria-labelledby={ariaLabelledBy}
        aria-describedby={ariaDescribedBy}
        aria-busy={ariaBusy}
      >
        {children}
      </div>
    </div>
  );

  return createPortal(modal, document.body);
}

export function ModalHeader({
  children,
  onClose,
  closeLabel = "閉じる",
  className = "",
  ...props
}: ModalHeaderProps) {
  const classes = ["ui-modal__header", className]
    .filter(Boolean)
    .join(" ");

  return (
    <div className={classes} {...props}>
      <div className="ui-modal__header-content">{children}</div>

      {onClose ? (
        <IconButton
          variant="ghost"
          size="sm"
          className="ui-modal__close"
          onClick={onClose}
          aria-label={closeLabel}
        >
          ×
        </IconButton>
      ) : null}
    </div>
  );
}

export function ModalTitle({
  children,
  className = "",
  ...props
}: ModalTitleProps) {
  const classes = ["ui-modal__title", className]
    .filter(Boolean)
    .join(" ");

  return (
    <h2 className={classes} {...props}>
      {children}
    </h2>
  );
}

export function ModalDescription({
  children,
  className = "",
  ...props
}: ModalDescriptionProps) {
  const classes = ["ui-modal__description", className]
    .filter(Boolean)
    .join(" ");

  return (
    <p className={classes} {...props}>
      {children}
    </p>
  );
}

export function ModalBody({
  children,
  className = "",
  ...props
}: ModalSectionProps) {
  const classes = ["ui-modal__body", className]
    .filter(Boolean)
    .join(" ");

  return (
    <div className={classes} {...props}>
      {children}
    </div>
  );
}

export function ModalFooter({
  children,
  className = "",
  ...props
}: ModalSectionProps) {
  const classes = ["ui-modal__footer", className]
    .filter(Boolean)
    .join(" ");

  return (
    <div className={classes} {...props}>
      {children}
    </div>
  );
}