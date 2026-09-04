// frontend/amol/src/features/shared/presentation/components/ChatComposerModal.tsx

import {
  useEffect,
  useId,
  type ReactNode,
} from "react";
import { createPortal } from "react-dom";

export type ChatComposerModalProps = {
  open: boolean;
  title: string;
  content: string;
  placeholder: string;
  error?: string | null;
  submitting: boolean;
  canSubmit: boolean;
  submitLabel: string;
  submittingLabel: string;
  onContentChange: (value: string) => void;
  onCancel: () => void;
  onSubmit: () => void;
  description?: ReactNode;
  afterInput?: ReactNode;
  cancelLabel?: string;
  inputAriaLabel?: string;
  rows?: number;
  maxLength?: number | null;
};

export default function ChatComposerModal({
  open,
  title,
  content,
  placeholder,
  error,
  submitting,
  canSubmit,
  submitLabel,
  submittingLabel,
  onContentChange,
  onCancel,
  onSubmit,
  description,
  afterInput,
  cancelLabel = "キャンセル",
  inputAriaLabel,
  rows = 6,
  maxLength = 500,
}: ChatComposerModalProps) {
  const titleId = useId();
  const descriptionId = useId();

  useEffect(() => {
    if (
      !open ||
      typeof document === "undefined"
    ) {
      return;
    }

    const previousOverflow =
      document.body.style.overflow;
    const previousTouchAction =
      document.body.style.touchAction;

    document.body.style.overflow = "hidden";
    document.body.style.touchAction = "none";

    return () => {
      document.body.style.overflow =
        previousOverflow;
      document.body.style.touchAction =
        previousTouchAction;
    };
  }, [open]);

  if (
    !open ||
    typeof document === "undefined"
  ) {
    return null;
  }

  return createPortal(
    <div className="chat-detail-page__modal-backdrop">
      <div
        className="chat-detail-page__modal"
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={
          description
            ? descriptionId
            : undefined
        }
      >
        <div className="chat-detail-page__modal-header">
          <h2 id={titleId}>
            {title}
          </h2>

          <button
            type="button"
            className="chat-detail-page__modal-close"
            onClick={onCancel}
            disabled={submitting}
            aria-label="閉じる"
          >
            ×
          </button>
        </div>

        {description ? (
          <div
            id={descriptionId}
            className="chat-detail-page__content"
          >
            {description}
          </div>
        ) : null}

        <textarea
          className="chat-detail-page__reply-input"
          value={content}
          onChange={(event) => {
            onContentChange(
              event.target.value,
            );
          }}
          placeholder={placeholder}
          aria-label={
            inputAriaLabel ??
            placeholder
          }
          rows={rows}
          maxLength={maxLength ?? undefined}
          disabled={submitting}
        />

        {afterInput}

        {error ? (
          <div
            className="chat-detail-page__modal-error"
            role="alert"
          >
            {error}
          </div>
        ) : null}

        <div className="chat-detail-page__modal-actions">
          <button
            type="button"
            onClick={onCancel}
            disabled={submitting}
          >
            {cancelLabel}
          </button>

          <button
            type="button"
            onClick={onSubmit}
            disabled={
              !canSubmit ||
              submitting
            }
          >
            {submitting
              ? submittingLabel
              : submitLabel}
          </button>
        </div>
      </div>
    </div>,
    document.body,
  );
}