// frontend/amol/src/features/inquiry/presentation/components/InquiryReplyModal.tsx

import {
  type ChangeEvent,
} from "react";
import { createPortal } from "react-dom";

type InquiryReplyModalProps = {
  open: boolean;
  content: string;
  files: File[];
  error?: string | null;
  submitting: boolean;
  canSubmit: boolean;
  onContentChange: (value: string) => void;
  onFilesChange: (
    event: ChangeEvent<HTMLInputElement>,
  ) => void;
  onRemoveFile: (index: number) => void;
  onCancel: () => void;
  onSubmit: () => void;
};

export default function InquiryReplyModal({
  open,
  content,
  files,
  error,
  submitting,
  canSubmit,
  onContentChange,
  onFilesChange,
  onRemoveFile,
  onCancel,
  onSubmit,
}: InquiryReplyModalProps) {
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
        aria-labelledby="chat-detail-reply-modal-title"
      >
        <div className="chat-detail-page__modal-header">
          <h2 id="chat-detail-reply-modal-title">
            返信する
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

        <textarea
          className="chat-detail-page__reply-input"
          value={content}
          onChange={(event) => {
            onContentChange(
              event.target.value,
            );
          }}
          placeholder="返信内容を入力"
          rows={6}
          disabled={submitting}
        />

        <label className="chat-detail-page__file-picker">
          <span>画像を追加</span>

          <input
            type="file"
            accept="image/*"
            multiple
            onChange={onFilesChange}
            disabled={submitting}
          />
        </label>

        {files.length > 0 ? (
          <div className="chat-detail-page__selected-files">
            {files.map(
              (file, index) => (
                <div
                  key={`${file.name}-${file.lastModified}-${index}`}
                  className="chat-detail-page__selected-file"
                >
                  <span>
                    {file.name}
                  </span>

                  <button
                    type="button"
                    onClick={() => {
                      onRemoveFile(index);
                    }}
                    disabled={submitting}
                  >
                    削除
                  </button>
                </div>
              ),
            )}
          </div>
        ) : null}

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
            キャンセル
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
              ? "送信中..."
              : "送信"}
          </button>
        </div>
      </div>
    </div>,
    document.body,
  );
}