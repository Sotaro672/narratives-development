// frontend/console/shell/src/features/brand/presentation/components/brandCreateProgressModal.tsx

import { Badge, type BadgeVariant } from "../../../../shared/ui/badge";
import { Modal, ModalCloseButton } from "../../../../shared/ui/modal";
import type { BrandCreateProgress } from "../model/brandCreateProgress";

import "../../../../styles/brandCreateProgress.css";

export type BrandCreateProgressModalProps = {
  open: boolean;
  progress: BrandCreateProgress;
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

function phaseStatusLabel(progress: BrandCreateProgress): string {
  switch (progress.phase) {
    case "idle":
      return "";
    case "creating":
      return "登録中";
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

function phaseStatusVariant(progress: BrandCreateProgress): BadgeVariant {
  switch (progress.phase) {
    case "creating":
      return "secondary";
    case "uploading":
      return "info";
    case "saving":
      return "warning";
    case "completed":
      return "success";
    case "failed":
      return "danger";
    case "idle":
    default:
      return "default";
  }
}

function shouldShowIndeterminate(progress: BrandCreateProgress): boolean {
  return (
    progress.phase === "creating" ||
    (progress.phase === "saving" && progress.totalBytes <= 0)
  );
}

function shouldShowProgressBar(progress: BrandCreateProgress): boolean {
  if (progress.totalBytes <= 0) {
    return false;
  }

  return (
    progress.phase === "uploading" ||
    progress.phase === "saving" ||
    progress.phase === "completed"
  );
}

function shouldShowUploadCount(progress: BrandCreateProgress): boolean {
  return (
    progress.expectedUploadCount > 0 &&
    (
      progress.phase === "uploading" ||
      progress.phase === "saving" ||
      progress.phase === "completed"
    )
  );
}

export default function BrandCreateProgressModal({
  open,
  progress,
  onClose,
}: BrandCreateProgressModalProps) {
  const canClose =
    !progress.isBlockingNavigation &&
    Boolean(onClose);

  const statusLabel = phaseStatusLabel(progress);
  const progressPercentage = Math.min(
    100,
    Math.max(0, progress.percentage),
  );

  const ariaBusy =
    progress.phase === "creating" ||
    progress.phase === "uploading" ||
    progress.phase === "saving";

  const showFooter =
    canClose &&
    (
      progress.phase === "completed" ||
      progress.phase === "failed"
    );

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
        showFooter ? (
          <ModalCloseButton onClick={onClose}>
            閉じる
          </ModalCloseButton>
        ) : undefined
      }
    >
      {shouldShowIndeterminate(progress) ? (
        <div
          className="brand-create-progress-modal__indeterminate"
          aria-label={
            progress.phase === "creating"
              ? "ブランド登録中"
              : "ブランド情報保存中"
          }
        >
          <div className="brand-create-progress-modal__indeterminate-bar" />
        </div>
      ) : null}

      {shouldShowProgressBar(progress) ? (
        <div className="brand-create-progress-modal__progress-section">
          <div className="brand-create-progress-modal__progress-header">
            <span className="brand-create-progress-modal__progress-label">
              画像転送
            </span>

            <span className="brand-create-progress-modal__progress-percentage">
              {progressPercentage}%
            </span>
          </div>

          <div
            className="brand-create-progress-modal__progress"
            role="progressbar"
            aria-label="ブランド画像転送進捗"
            aria-valuemin={0}
            aria-valuemax={100}
            aria-valuenow={progressPercentage}
          >
            <div
              className="brand-create-progress-modal__progress-bar"
              style={{ width: `${progressPercentage}%` }}
            />
          </div>

          <div className="brand-create-progress-modal__bytes">
            {formatBytes(progress.transferredBytes)}
            {" / "}
            {formatBytes(progress.totalBytes)}
          </div>
        </div>
      ) : null}

      {progress.phase === "uploading" && progress.currentFileName ? (
        <div className="brand-create-progress-modal__current">
          <span className="brand-create-progress-modal__current-label">
            画像
          </span>

          <span
            className="brand-create-progress-modal__current-file"
            title={progress.currentFileName}
          >
            {progress.currentFileName}
          </span>
        </div>
      ) : null}

      {shouldShowUploadCount(progress) ? (
        <div className="brand-create-progress-modal__count">
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
          className="brand-create-progress-modal__warning"
          role="alert"
        >
          画像転送が完了するまで、この画面を閉じたり別のページへ移動したりしないでください。
        </div>
      ) : null}

      {progress.phase === "saving" ? (
        <div className="brand-create-progress-modal__notice">
          画像転送は完了しています。ブランド情報の保存処理を続けています。
        </div>
      ) : null}

      {progress.errorMessage ? (
        <div
          className="brand-create-progress-modal__error"
          role="alert"
        >
          {progress.errorMessage}
        </div>
      ) : null}
    </Modal>
  );
}