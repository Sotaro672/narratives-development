// frontend/console/shell/src/features/announcement/presentation/components/announcementCreateProgressModal.tsx

import { Badge, type BadgeVariant } from "../../../../shared/ui/badge";
import { Modal, ModalCloseButton } from "../../../../shared/ui/modal";
import {
  Progress,
  ProgressCurrent,
  ProgressIndeterminate,
  ProgressMessage,
  ProgressMetric,
} from "../../../../shared/ui/progress";
import type { AnnouncementCreateProgress } from "../model/announcementCreateProgress";

export type AnnouncementCreateProgressModalProps = {
  open: boolean;
  progress: AnnouncementCreateProgress;
  onClose?: () => void;
};

function phaseStatusLabel(progress: AnnouncementCreateProgress): string {
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

function phaseStatusVariant(
  progress: AnnouncementCreateProgress,
): BadgeVariant {
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

function shouldShowIndeterminate(
  progress: AnnouncementCreateProgress,
): boolean {
  return (
    progress.phase === "preparing" ||
    (progress.phase === "saving" && progress.totalBytes <= 0)
  );
}

function shouldShowProgressBar(
  progress: AnnouncementCreateProgress,
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
  progress: AnnouncementCreateProgress,
): boolean {
  return (
    progress.expectedUploadCount > 0 &&
    (progress.phase === "uploading" ||
      progress.phase === "saving" ||
      progress.phase === "completed")
  );
}

export default function AnnouncementCreateProgressModal({
  open,
  progress,
  onClose,
}: AnnouncementCreateProgressModalProps) {
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
        <ProgressIndeterminate
          ariaLabel={progress.phase === "saving" ? "告知保存中" : "処理準備中"}
        />
      ) : null}

      {shouldShowProgressBar(progress) ? (
        <Progress
          value={progress.percentage}
          label="画像転送"
          ariaLabel="告知画像転送進捗"
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
          画像転送は完了しています。告知情報の保存処理を続けています。
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