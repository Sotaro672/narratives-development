//frontend\console\inquiry\presentation\components\replyModal.tsx
import * as React from "react";
import { createPortal } from "react-dom";

const MAX_REPLY_IMAGES = 10;

export type ReplyUploadImage = {
  id: string;
  file: File;
  previewUrl: string;
};

type ReplyModalProps = {
  open: boolean;
  content: string;
  images: ReplyUploadImage[];
  submitting: boolean;
  errorMessage: string | null;
  onClose: () => void;
  onChangeContent: (value: string) => void;
  onChangeImages: React.ChangeEventHandler<HTMLInputElement>;
  onRemoveImage: (id: string) => void;
  onSubmit: () => void;
};

export default function ReplyModal({
  open,
  content,
  images,
  submitting,
  errorMessage,
  onClose,
  onChangeContent,
  onChangeImages,
  onRemoveImage,
  onSubmit,
}: ReplyModalProps) {
  if (!open) {
    return null;
  }

  const modal = (
    <div
      className="inq-reply-modal"
      role="presentation"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) {
          onClose();
        }
      }}
    >
      <div
        className="inq-reply-modal__panel"
        role="dialog"
        aria-modal="true"
        aria-labelledby="inquiry-reply-modal-title"
      >
        <div className="inq-reply-modal__header">
          <div>
            <h2
              id="inquiry-reply-modal-title"
              className="inq-reply-modal__title"
            >
              返信を入力
            </h2>
            <p className="inq-reply-modal__description">
              この問い合わせに対する返信内容を入力してください。
            </p>
          </div>

          <button
            type="button"
            className="inq-reply-modal__close"
            onClick={onClose}
            disabled={submitting}
            aria-label="返信モーダルを閉じる"
          >
            ×
          </button>
        </div>

        <div className="inq-reply-modal__body">
          {errorMessage ? (
            <div className="inq__empty">{errorMessage}</div>
          ) : null}

          <label
            className="inq-reply-modal__label"
            htmlFor="inquiry-reply-content"
          >
            返信内容
          </label>

          <textarea
            id="inquiry-reply-content"
            className="inq-reply-modal__textarea"
            value={content}
            placeholder="返信内容を入力してください"
            rows={8}
            maxLength={2000}
            disabled={submitting}
            onChange={(event) => onChangeContent(event.target.value)}
          />

          <div className="inq-reply-modal__counter">
            {content.length.toLocaleString()} / 2,000
          </div>

          <div className="inq-reply-modal__upload">
            <div className="inq-reply-modal__upload-header">
              <span className="inq-reply-modal__label">添付画像</span>
              <span className="inq-reply-modal__upload-count">
                {images.length} / {MAX_REPLY_IMAGES}
              </span>
            </div>

            <label className="inq-reply-modal__upload-box">
              <input
                type="file"
                accept="image/*"
                multiple
                className="inq-reply-modal__upload-input"
                disabled={submitting || images.length >= MAX_REPLY_IMAGES}
                onChange={onChangeImages}
              />
              <span className="inq-reply-modal__upload-main">画像を選択</span>
              <span className="inq-reply-modal__upload-sub">
                JPG / PNG / WebP / GIF、1枚20MBまで
              </span>
            </label>

            {images.length > 0 ? (
              <div className="inq-reply-modal__preview-grid">
                {images.map((image) => (
                  <div
                    key={image.id}
                    className="inq-reply-modal__preview-item"
                  >
                    <img
                      src={image.previewUrl}
                      alt={image.file.name}
                      className="inq-reply-modal__preview-image"
                    />

                    <button
                      type="button"
                      className="inq-reply-modal__preview-remove"
                      disabled={submitting}
                      onClick={() => onRemoveImage(image.id)}
                      aria-label={`${image.file.name}を削除`}
                    >
                      ×
                    </button>
                  </div>
                ))}
              </div>
            ) : null}
          </div>
        </div>

        <div className="inq-reply-modal__actions">
          <button
            type="button"
            className="inq-reply-modal__button inq-reply-modal__button--ghost"
            onClick={onClose}
            disabled={submitting}
          >
            キャンセル
          </button>

          <button
            type="button"
            className="inq-reply-modal__button"
            disabled={submitting || !content.trim()}
            onClick={onSubmit}
          >
            {submitting ? "送信中" : "送信"}
          </button>
        </div>
      </div>
    </div>
  );

  return createPortal(modal, document.body);
}