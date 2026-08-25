// frontend/console/shell/src/features/list/presentation/components/listProgressModal.tsx

import * as React from "react";
import { createPortal } from "react-dom";

import type {
  ListProgress,
} from "../modal/listProgress";

export type ListProgressModalProps = {
  open: boolean;
  progress: ListProgress;
  onClose?: () => void;
};

function formatBytes(
  bytes: number,
): string {
  if (!Number.isFinite(bytes) || bytes <= 0) {
    return "0 B";
  }

  const units = [
    "B",
    "KB",
    "MB",
    "GB",
  ];

  let value = bytes;
  let unitIndex = 0;

  while (
    value >= 1024 &&
    unitIndex < units.length - 1
  ) {
    value /= 1024;
    unitIndex += 1;
  }

  const digits =
    unitIndex === 0
      ? 0
      : value >= 10
        ? 1
        : 2;

  return `${value.toFixed(digits)} ${units[unitIndex]}`;
}

function phaseStatusLabel(
  progress: ListProgress,
): string {
  switch (progress.phase) {
    case "idle":
      return "";

    case "preparing":
      return "準備中";

    case "uploading":
      return "転送中";

    case "saving":
      return "保存中";

    case "completed":
      return "完了";

    case "failed":
      return "失敗";
  }
}

function statusClassName(
  progress: ListProgress,
): string {
  switch (progress.phase) {
    case "completed":
      return "list-progress-modal__status--completed";

    case "failed":
      return "list-progress-modal__status--failed";

    default:
      return `list-progress-modal__status--${progress.phase}`;
  }
}

function shouldShowIndeterminate(
  progress: ListProgress,
): boolean {
  return (
    progress.phase === "preparing" ||
    (
      progress.phase === "saving" &&
      progress.totalBytes <= 0
    )
  );
}

function shouldShowProgressBar(
  progress: ListProgress,
): boolean {
  if (progress.totalBytes <= 0) {
    return false;
  }

  return (
    progress.phase === "uploading" ||
    progress.phase === "saving" ||
    progress.phase === "completed"
  );
}

function shouldShowUploadCount(
  progress: ListProgress,
): boolean {
  return (
    progress.expectedUploadCount > 0 &&
    (
      progress.phase === "uploading" ||
      progress.phase === "saving" ||
      progress.phase === "completed"
    )
  );
}

export default function ListProgressModal({
  open,
  progress,
  onClose,
}: ListProgressModalProps) {
  const canClose =
    !progress.isBlockingNavigation &&
    Boolean(onClose);

  React.useEffect(() => {
    if (!open || !canClose) {
      return;
    }

    const handleKeyDown = (
      event: KeyboardEvent,
    ) => {
      if (event.key !== "Escape") {
        return;
      }

      event.preventDefault();
      onClose?.();
    };

    document.addEventListener(
      "keydown",
      handleKeyDown,
    );

    return () => {
      document.removeEventListener(
        "keydown",
        handleKeyDown,
      );
    };
  }, [
    open,
    canClose,
    onClose,
  ]);

  if (!open) {
    return null;
  }

  const statusLabel =
    phaseStatusLabel(progress);

  const progressPercentage =
    Math.min(
      100,
      Math.max(
        0,
        progress.percentage,
      ),
    );

  const modal = (
    <div
      className="list-progress-modal"
      role="presentation"
      onMouseDown={(event) => {
        if (
          event.target === event.currentTarget &&
          canClose
        ) {
          onClose?.();
        }
      }}
    >
      <div
        className="list-progress-modal__panel"
        role="dialog"
        aria-modal="true"
        aria-labelledby="list-progress-modal-title"
        aria-describedby="list-progress-modal-description"
        aria-busy={
          progress.phase === "preparing" ||
          progress.phase === "uploading" ||
          progress.phase === "saving"
        }
      >
        <div className="list-progress-modal__header">
          <div className="list-progress-modal__heading">
            {statusLabel ? (
              <span
                className={[
                  "list-progress-modal__status",
                  statusClassName(progress),
                ].join(" ")}
              >
                {statusLabel}
              </span>
            ) : null}

            <h2
              id="list-progress-modal-title"
              className="list-progress-modal__title"
            >
              {progress.title}
            </h2>
          </div>

          {canClose ? (
            <button
              type="button"
              className="list-progress-modal__close"
              onClick={onClose}
              aria-label="進捗画面を閉じる"
            >
              ×
            </button>
          ) : null}
        </div>

        <div className="list-progress-modal__body">
          <p
            id="list-progress-modal-description"
            className="list-progress-modal__description"
          >
            {progress.message}
          </p>

          {shouldShowIndeterminate(progress) ? (
            <div
              className="list-progress-modal__indeterminate"
              aria-label={
                progress.phase === "saving"
                  ? "保存中"
                  : "準備中"
              }
            >
              <div className="list-progress-modal__indeterminate-bar" />
            </div>
          ) : null}

          {shouldShowProgressBar(progress) ? (
            <div className="list-progress-modal__progress-section">
              <div className="list-progress-modal__progress-header">
                <span className="list-progress-modal__progress-label">
                  画像転送
                </span>

                <span className="list-progress-modal__progress-percentage">
                  {progressPercentage}%
                </span>
              </div>

              <div
                className="list-progress-modal__progress"
                role="progressbar"
                aria-label="画像転送進捗"
                aria-valuemin={0}
                aria-valuemax={100}
                aria-valuenow={progressPercentage}
              >
                <div
                  className="list-progress-modal__progress-bar"
                  style={{
                    width: `${progressPercentage}%`,
                  }}
                />
              </div>

              <div className="list-progress-modal__bytes">
                {formatBytes(
                  progress.transferredBytes,
                )}
                {" / "}
                {formatBytes(
                  progress.totalBytes,
                )}
              </div>
            </div>
          ) : null}

          {progress.phase === "uploading" &&
          progress.currentFileName ? (
            <div className="list-progress-modal__current">
              <span className="list-progress-modal__current-label">
                画像
              </span>

              <span
                className="list-progress-modal__current-file"
                title={progress.currentFileName}
              >
                {progress.currentFileName}
              </span>
            </div>
          ) : null}

          {shouldShowUploadCount(progress) ? (
            <div className="list-progress-modal__count">
              <span>
                転送済み画像
              </span>

              <strong>
                {progress.completedUploadCount}
                {" / "}
                {progress.expectedUploadCount}
              </strong>
            </div>
          ) : null}

          {progress.isBlockingNavigation ? (
            <div
              className="list-progress-modal__warning"
              role="alert"
            >
              処理が完了するまで、この画面を閉じたり別のページへ移動したりしないでください。
            </div>
          ) : null}

          {progress.errorMessage ? (
            <div
              className="list-progress-modal__error"
              role="alert"
            >
              {progress.errorMessage}
            </div>
          ) : null}
        </div>

        {progress.phase === "completed" ||
        progress.phase === "failed" ? (
          <div className="list-progress-modal__actions">
            {canClose ? (
              <button
                type="button"
                className="list-progress-modal__button list-progress-modal__button--secondary"
                onClick={onClose}
              >
                閉じる
              </button>
            ) : null}
          </div>
        ) : null}
      </div>
    </div>
  );

  return createPortal(
    modal,
    document.body,
  );
}