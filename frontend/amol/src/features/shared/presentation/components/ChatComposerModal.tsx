// frontend/amol/src/features/shared/presentation/components/ChatComposerModal.tsx

import { useId, type ReactNode } from "react";
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
  cancelLabel?: string;
  inputAriaLabel?: string;
  rows?: number;
  maxLength?: number;
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
  cancelLabel = "キャンセル",
  inputAriaLabel,
  rows = 6,
  maxLength = 500,
}: ChatComposerModalProps) {
  const titleId = useId();
  const descriptionId = useId();

  if (!open || typeof document === "undefined") {
    return null;
  }

  return createPortal(
    <div className="chat-detail-page__modal-backdrop">
      <div
        className="chat-detail-page__modal"
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={description ? descriptionId : undefined}
      >
        <div className="chat-detail-page__modal-header">
          <h2 id={titleId}>{title}</h2>

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
          <div id={descriptionId} className="chat-detail-page__content">
            {description}
          </div>
        ) : null}

        <textarea
          className="chat-detail-page__reply-input"
          value={content}
          onChange={(event) => {
            onContentChange(event.target.value);
          }}
          placeholder={placeholder}
          aria-label={inputAriaLabel ?? placeholder}
          rows={rows}
          maxLength={maxLength}
          disabled={submitting}
        />

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
            disabled={!canSubmit || submitting}
          >
            {submitting ? submittingLabel : submitLabel}
          </button>
        </div>
      </div>
    </div>,
    document.body,
  );
}
