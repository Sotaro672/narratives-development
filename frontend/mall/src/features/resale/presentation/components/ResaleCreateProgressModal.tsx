// frontend/amol/src/features/resale/presentation/components/ResaleCreateProgressModal.tsx

import { useEffect } from "react";
import { createPortal } from "react-dom";

import type { ResaleCreateProgress } from "../models/resaleCreateProgress";

import "../styles/resale-create-progress.css";

export type ResaleCreateProgressModalProps = {
  open: boolean;
  progress: ResaleCreateProgress;
  onClose?: () => void;
};

function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) {
    return "0 B";
  }

  const units = ["B", "KB", "MB", "GB"];
  let value = bytes;
  let unitIndex = 0;

  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }

  const digits = unitIndex === 0 ? 0 : value >= 10 ? 1 : 2;

  return `${value.toFixed(digits)} ${units[unitIndex]}`;
}

function getStatusLabel(progress: ResaleCreateProgress): string {
  switch (progress.phase) {
    case "idle":
      return "";
    case "preparing":
      return "準備中";
    case "uploading":
      return "転送中";
    case "saving":
      return "保存中";
    case "failed":
      return "失敗";
  }
}

function shouldShowIndeterminate(progress: ResaleCreateProgress): boolean {
  return progress.phase === "preparing" || progress.phase === "saving";
}

function shouldShowProgressBar(progress: ResaleCreateProgress): boolean {
  return (
    progress.totalBytes > 0 &&
    (progress.phase === "uploading" || progress.phase === "saving")
  );
}

function shouldShowUploadCount(progress: ResaleCreateProgress): boolean {
  return (
    progress.expectedUploadCount > 0 &&
    (progress.phase === "uploading" || progress.phase === "saving")
  );
}

export default function ResaleCreateProgressModal({
  open,
  progress,
  onClose,
}: ResaleCreateProgressModalProps) {
  const canClose = !progress.isBlockingNavigation && Boolean(onClose);

  useEffect(() => {
    if (!open || !canClose) {
      return;
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "Escape") {
        return;
      }

      event.preventDefault();
      onClose?.();
    };

    document.addEventListener("keydown", handleKeyDown);

    return () => {
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [open, canClose, onClose]);

  if (!open) {
    return null;
  }

  const statusLabel = getStatusLabel(progress);
  const progressPercentage = Math.min(
    100,
    Math.max(0, progress.percentage),
  );

  const modal = (
    <div
      className="resale-create-progress-modal"
      role="presentation"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget && canClose) {
          onClose?.();
        }
      }}
    >
      <div
        className="resale-create-progress-modal__panel"
        role="dialog"
        aria-modal="true"
        aria-labelledby="resale-create-progress-modal-title"
        aria-describedby="resale-create-progress-modal-description"
        aria-busy={
          progress.phase === "preparing" ||
          progress.phase === "uploading" ||
          progress.phase === "saving"
        }
      >
        <div className="resale-create-progress-modal__header">
          <div className="resale-create-progress-modal__heading">
            {statusLabel ? (
              <span
                className={[
                  "resale-create-progress-modal__status",
                  `resale-create-progress-modal__status--${progress.phase}`,
                ].join(" ")}
              >
                {statusLabel}
              </span>
            ) : null}

            <h2
              id="resale-create-progress-modal-title"
              className="resale-create-progress-modal__title"
            >
              {progress.title}
            </h2>
          </div>

          {canClose ? (
            <button
              type="button"
              className="resale-create-progress-modal__close"
              onClick={onClose}
              aria-label="進捗画面を閉じる"
            >
              ×
            </button>
          ) : null}
        </div>

        <div className="resale-create-progress-modal__body">
          <p
            id="resale-create-progress-modal-description"
            className="resale-create-progress-modal__description"
          >
            {progress.message}
          </p>

          {shouldShowIndeterminate(progress) ? (
            <div
              className="resale-create-progress-modal__indeterminate"
              aria-label={progress.phase === "saving" ? "保存中" : "準備中"}
            >
              <div className="resale-create-progress-modal__indeterminate-bar" />
            </div>
          ) : null}

          {shouldShowProgressBar(progress) ? (
            <div className="resale-create-progress-modal__progress-section">
              <div className="resale-create-progress-modal__progress-header">
                <span className="resale-create-progress-modal__progress-label">
                  画像転送
                </span>
                <span className="resale-create-progress-modal__progress-percentage">
                  {progressPercentage}%
                </span>
              </div>

              <div
                className="resale-create-progress-modal__progress"
                role="progressbar"
                aria-label="商品状態画像の転送進捗"
                aria-valuemin={0}
                aria-valuemax={100}
                aria-valuenow={progressPercentage}
              >
                <div
                  className="resale-create-progress-modal__progress-bar"
                  style={{
                    width: `${progressPercentage}%`,
                  }}
                />
              </div>

              <div className="resale-create-progress-modal__bytes">
                {formatBytes(progress.transferredBytes)}
                {" / "}
                {formatBytes(progress.totalBytes)}
              </div>
            </div>
          ) : null}

          {progress.phase === "uploading" && progress.currentFileName ? (
            <div className="resale-create-progress-modal__current">
              <span className="resale-create-progress-modal__current-label">
                画像
              </span>
              <span
                className="resale-create-progress-modal__current-file"
                title={progress.currentFileName}
              >
                {progress.currentFileName}
              </span>
            </div>
          ) : null}

          {shouldShowUploadCount(progress) ? (
            <div className="resale-create-progress-modal__count">
              <span>転送済み画像</span>
              <strong>
                {progress.completedUploadCount}
                {" / "}
                {progress.expectedUploadCount}
              </strong>
            </div>
          ) : null}

          {progress.phase === "uploading" && progress.isBrowserDependent ? (
            <div
              className="resale-create-progress-modal__warning"
              role="alert"
            >
              画像転送が完了するまで、この画面を閉じたり別のページへ移動したりしないでください。
            </div>
          ) : null}

          {progress.phase === "saving" ? (
            <div className="resale-create-progress-modal__notice">
              画像転送は完了しています。出品情報の保存処理を続けています。
            </div>
          ) : null}

          {progress.errorMessage ? (
            <div
              className="resale-create-progress-modal__error"
              role="alert"
            >
              {progress.errorMessage}
            </div>
          ) : null}
        </div>

        {progress.phase === "failed" && canClose ? (
          <div className="resale-create-progress-modal__actions">
            <button
              type="button"
              className="resale-create-progress-modal__button"
              onClick={onClose}
            >
              閉じる
            </button>
          </div>
        ) : null}
      </div>
    </div>
  );

  return createPortal(modal, document.body);
}