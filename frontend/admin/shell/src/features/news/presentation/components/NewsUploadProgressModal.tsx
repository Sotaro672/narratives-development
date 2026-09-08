// frontend\admin\shell\src\features\news\presentation\components\NewsUploadProgressModal.tsx

import { useEffect } from "react";
import { createPortal } from "react-dom";

import "./NewsUploadProgressModal.css";

type NewsUploadProgressModalProps = {
  open: boolean;
  progress: number;
  fileName?: string | null;
};

function normalizeProgress(value: number): number {
  if (!Number.isFinite(value)) {
    return 0;
  }

  return Math.min(100, Math.max(0, Math.round(value)));
}

export default function NewsUploadProgressModal({
  open,
  progress,
  fileName,
}: NewsUploadProgressModalProps) {
  const normalizedProgress = normalizeProgress(progress);
  const transferCompleted = normalizedProgress >= 100;

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

  return createPortal(
    <div
      className="news-upload-progress-modal"
      role="presentation"
    >
      <div
        className="news-upload-progress-modal__panel"
        role="dialog"
        aria-modal="true"
        aria-labelledby="news-upload-progress-modal-title"
        aria-describedby="news-upload-progress-modal-description"
        aria-busy="true"
      >
        <div className="news-upload-progress-modal__header">
          <span className="news-upload-progress-modal__status">
            {transferCompleted ? "公開処理中" : "転送中"}
          </span>

          <h2
            id="news-upload-progress-modal-title"
            className="news-upload-progress-modal__title"
          >
            {transferCompleted
              ? "画像の転送が完了しました"
              : "画像を送信しています"}
          </h2>
        </div>

        <div className="news-upload-progress-modal__body">
          <p
            id="news-upload-progress-modal-description"
            className="news-upload-progress-modal__description"
          >
            {transferCompleted
              ? "画像を保存して通知を公開しています。この画面を閉じずにお待ちください。"
              : "画像転送が完了するまで、この画面を閉じたり別のページへ移動したりしないでください。"}
          </p>

          {fileName ? (
            <div className="news-upload-progress-modal__file">
              <span className="news-upload-progress-modal__file-label">
                ファイル
              </span>
              <span className="news-upload-progress-modal__file-name">
                {fileName}
              </span>
            </div>
          ) : null}

          <div className="news-upload-progress-modal__progress-section">
            <div className="news-upload-progress-modal__progress-header">
              <span>転送進捗</span>
              <strong>{normalizedProgress}%</strong>
            </div>

            <div
              className="news-upload-progress-modal__progress"
              role="progressbar"
              aria-label="通知画像の転送進捗"
              aria-valuemin={0}
              aria-valuemax={100}
              aria-valuenow={normalizedProgress}
            >
              <div
                className="news-upload-progress-modal__progress-bar"
                style={{
                  width: `${normalizedProgress}%`,
                }}
              />
            </div>
          </div>

          {transferCompleted ? (
            <div
              className="news-upload-progress-modal__processing"
              aria-live="polite"
            >
              <span
                className="news-upload-progress-modal__spinner"
                aria-hidden="true"
              />
              <span>通知を公開しています...</span>
            </div>
          ) : null}
        </div>
      </div>
    </div>,
    document.body,
  );
}