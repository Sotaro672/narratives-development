// frontend/console/shell/src/features/brand/presentation/components/brandCreateProgressModal.tsx

import { Badge, type BadgeVariant } from "../../../../shared/ui/badge";
import { Modal, ModalCloseButton } from "../../../../shared/ui/modal";
import {
  Progress,
  ProgressCurrent,
  ProgressIndeterminate,
  ProgressMessage,
  ProgressMetric,
} from "../../../../shared/ui/progress";
import type { BrandProgress } from "../model/brandProgress";

export type BrandProgressModalProps = {
  open: boolean;
  progress: BrandProgress;
  onClose?: () => void;
};

function phaseStatusLabel(progress: BrandProgress): string {
  switch (progress.phase) {
    case "idle":
      return "";
    case "preparing":
      return progress.variant === "update" ? "更新中" : "登録中";
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

function phaseStatusVariant(progress: BrandProgress): BadgeVariant {
  switch (progress.phase) {
    case "preparing":
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

function shouldShowIndeterminate(progress: BrandProgress): boolean {
  return (
    progress.phase === "preparing" ||
    (progress.phase === "saving" && progress.totalBytes <= 0)
  );
}

function shouldShowProgressBar(progress: BrandProgress): boolean {
  if (progress.totalBytes <= 0) {
    return false;
  }

  return (
    progress.phase === "uploading" ||
    progress.phase === "saving" ||
    progress.phase === "completed"
  );
}

function shouldShowUploadCount(progress: BrandProgress): boolean {
  return (
    progress.expectedUploadCount > 0 &&
    (progress.phase === "uploading" ||
      progress.phase === "saving" ||
      progress.phase === "completed")
  );
}

function indeterminateLabel(progress: BrandProgress): string {
  if (progress.phase === "preparing") {
    return progress.variant === "update"
      ? "ブランド更新中"
      : "ブランド登録中";
  }

  return "ブランド情報保存中";
}

export default function BrandProgressModal({
  open,
  progress,
  onClose,
}: BrandProgressModalProps) {
  const canClose = !progress.isBlockingNavigation && Boolean(onClose);
  const statusLabel = phaseStatusLabel(progress);

  const ariaBusy =
    progress.phase === "preparing" ||
    progress.phase === "uploading" ||
    progress.phase === "saving";

  const showFooter =
    canClose &&
    (progress.phase === "completed" || progress.phase === "failed");

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
        <ProgressIndeterminate ariaLabel={indeterminateLabel(progress)} />
      ) : null}

      {shouldShowProgressBar(progress) ? (
        <Progress
          value={progress.percentage}
          label="画像転送"
          ariaLabel="ブランド画像転送進捗"
          transferredBytes={progress.transferredBytes}
          totalBytes={progress.totalBytes}
        />
      ) : null}

      {progress.phase === "uploading" && progress.currentFileName ? (
        <ProgressCurrent
          label="画像"
          value={progress.currentFileName}
        />
      ) : null}

      {shouldShowUploadCount(progress) ? (
        <ProgressMetric
          label="転送済み画像"
          value={`${progress.completedUploadCount} / ${progress.expectedUploadCount}`}
        />
      ) : null}

      {progress.phase === "uploading" && progress.isBrowserDependent ? (
        <ProgressMessage variant="warning">
          画像転送が完了するまで、この画面を閉じたり別のページへ移動したりしないでください。
        </ProgressMessage>
      ) : null}

      {progress.phase === "saving" ? (
        <ProgressMessage variant="notice">
          画像転送は完了しています。ブランド情報の保存処理を続けています。
        </ProgressMessage>
      ) : null}

      {progress.errorMessage ? (
        <ProgressMessage variant="error">
          {progress.errorMessage}
        </ProgressMessage>
      ) : null}
    </Modal>
  );
}