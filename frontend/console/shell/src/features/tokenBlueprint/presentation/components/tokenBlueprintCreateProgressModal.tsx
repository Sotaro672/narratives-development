// frontend/console/shell/src/features/tokenBlueprint/presentation/components/tokenBlueprintCreateProgressModal.tsx

import { Badge, type BadgeVariant } from "../../../../shared/ui/badge";
import {
  Modal,
  ModalButton,
  ModalCloseButton,
} from "../../../../shared/ui/modal";
import type {
  TokenBlueprintCreateProgress,
} from "../model/tokenBlueprintCreateProgress";

export type TokenBlueprintCreateProgressModalProps = {
  open: boolean;
  progress: TokenBlueprintCreateProgress;
  retrying?: boolean;
  onClose?: () => void;
  onRetry?: () => void;
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

function uploadTargetLabel(
  target: TokenBlueprintCreateProgress["currentUploadTarget"],
): string {
  switch (target) {
    case "icon":
      return "アイコン";
    case "content":
      return "コンテンツ";
    default:
      return "";
  }
}

function phaseStatusLabel(
  progress: TokenBlueprintCreateProgress,
): string {
  switch (progress.phase) {
    case "idle":
      return "";
    case "starting":
      return "準備中";
    case "uploading":
      return "転送中";
    case "queued":
      return "保存待機中";
    case "processing":
      return "保存中";
    case "completed":
      return "完了";
    case "failed_retryable":
      return "再試行可能";
    case "failed_fatal":
      return "失敗";
  }
}

function phaseStatusVariant(
  progress: TokenBlueprintCreateProgress,
): BadgeVariant {
  switch (progress.phase) {
    case "starting":
      return "secondary";
    case "uploading":
      return "info";
    case "queued":
      return "secondary";
    case "processing":
      return "warning";
    case "completed":
      return "success";
    case "failed_retryable":
      return "warning";
    case "failed_fatal":
      return "danger";
    case "idle":
    default:
      return "default";
  }
}

function shouldShowProgressBar(
  progress: TokenBlueprintCreateProgress,
): boolean {
  return (
    progress.phase === "uploading" ||
    progress.phase === "queued" ||
    progress.phase === "processing" ||
    progress.phase === "completed"
  );
}

function shouldShowUploadCount(
  progress: TokenBlueprintCreateProgress,
): boolean {
  return (
    progress.expectedUploadCount > 0 &&
    progress.phase !== "starting"
  );
}

export default function TokenBlueprintCreateProgressModal({
  open,
  progress,
  retrying = false,
  onClose,
  onRetry,
}: TokenBlueprintCreateProgressModalProps) {
  const canClose =
    !progress.isBlockingNavigation &&
    Boolean(onClose);

  const canRetry =
    progress.phase === "failed_retryable" &&
    progress.canRetry &&
    Boolean(onRetry);

  const statusLabel = phaseStatusLabel(progress);
  const targetLabel = uploadTargetLabel(
    progress.currentUploadTarget,
  );

  const progressPercentage = Math.min(
    100,
    Math.max(0, progress.percentage),
  );

  const ariaBusy =
    progress.phase === "starting" ||
    progress.phase === "uploading" ||
    progress.phase === "queued" ||
    progress.phase === "processing";

  const showActions =
    progress.phase === "failed_retryable" ||
    progress.phase === "failed_fatal" ||
    progress.phase === "completed";

  return (
    <Modal
      open={open}
      title={progress.title}
      description={progress.message}
      eyebrow={
        statusLabel ? (
          <Badge variant={phaseStatusVariant(progress)}>
            {statusLabel}
          </Badge>
        ) : undefined
      }
      onClose={canClose ? onClose : undefined}
      closeable={canClose}
      closeOnBackdrop={canClose}
      closeOnEscape={canClose}
      showCloseButton={canClose}
      closeLabel="進捗画面を閉じる"
      ariaBusy={ariaBusy}
      footer={
        showActions ? (
          <>
            {canRetry ? (
              <ModalButton
                variant="primary"
                disabled={retrying}
                onClick={onRetry}
              >
                {retrying ? "再試行中" : "再試行"}
              </ModalButton>
            ) : null}

            {canClose ? (
              <ModalCloseButton
                disabled={retrying}
                onClick={onClose}
              >
                閉じる
              </ModalCloseButton>
            ) : null}
          </>
        ) : undefined
      }
    >
      {progress.phase === "starting" ? (
        <div
          className="token-blueprint-create-progress-modal__indeterminate"
          aria-label="作成準備中"
        >
          <div className="token-blueprint-create-progress-modal__indeterminate-bar" />
        </div>
      ) : null}

      {shouldShowProgressBar(progress) ? (
        <div className="token-blueprint-create-progress-modal__progress-section">
          <div className="token-blueprint-create-progress-modal__progress-header">
            <span className="token-blueprint-create-progress-modal__progress-label">
              ファイル転送
            </span>

            <span className="token-blueprint-create-progress-modal__progress-percentage">
              {progressPercentage}%
            </span>
          </div>

          <div
            className="token-blueprint-create-progress-modal__progress"
            role="progressbar"
            aria-label="ファイル転送進捗"
            aria-valuemin={0}
            aria-valuemax={100}
            aria-valuenow={progressPercentage}
          >
            <div
              className="token-blueprint-create-progress-modal__progress-bar"
              style={{ width: `${progressPercentage}%` }}
            />
          </div>

          {progress.totalBytes > 0 ? (
            <div className="token-blueprint-create-progress-modal__bytes">
              {formatBytes(progress.transferredBytes)}
              {" / "}
              {formatBytes(progress.totalBytes)}
            </div>
          ) : null}
        </div>
      ) : null}

      {progress.phase === "uploading" && progress.currentFileName ? (
        <div className="token-blueprint-create-progress-modal__current">
          <span className="token-blueprint-create-progress-modal__current-label">
            {targetLabel || "ファイル"}
          </span>

          <span
            className="token-blueprint-create-progress-modal__current-file"
            title={progress.currentFileName}
          >
            {progress.currentFileName}
          </span>
        </div>
      ) : null}

      {shouldShowUploadCount(progress) ? (
        <div className="token-blueprint-create-progress-modal__count">
          <span>転送済みファイル</span>

          <strong>
            {progress.completedUploadCount}
            {" / "}
            {progress.expectedUploadCount}
          </strong>
        </div>
      ) : null}

      {progress.phase === "uploading" ? (
        <div
          className="token-blueprint-create-progress-modal__warning"
          role="alert"
        >
          ファイル転送中は、この画面を閉じたり別のページへ移動したりしないでください。
        </div>
      ) : null}

      {progress.phase === "queued" ||
      progress.phase === "processing" ? (
        <div className="token-blueprint-create-progress-modal__notice">
          ファイル転送は完了しています。ここからの処理はサーバー側で継続されます。
        </div>
      ) : null}

      {progress.errorMessage ? (
        <div
          className="token-blueprint-create-progress-modal__error"
          role="alert"
        >
          {progress.errorMessage}
        </div>
      ) : null}

      {progress.phase === "failed_retryable" ? (
        <div className="token-blueprint-create-progress-modal__retry-info">
          <span>再試行回数</span>

          <strong>
            {progress.retryCount}
            {" / "}
            {progress.maxRetries}
          </strong>
        </div>
      ) : null}
    </Modal>
  );
}