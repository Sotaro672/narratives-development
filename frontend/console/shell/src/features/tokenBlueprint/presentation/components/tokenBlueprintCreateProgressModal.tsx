// frontend/console/shell/src/features/tokenBlueprint/presentation/components/tokenBlueprintCreateProgressModal.tsx

import { Badge, type BadgeVariant } from "../../../../shared/ui/badge";
import {
  Modal,
  ModalButton,
  ModalCloseButton,
} from "../../../../shared/ui/modal";
import {
  Progress,
  ProgressCurrent,
  ProgressIndeterminate,
  ProgressMessage,
  ProgressMetric,
} from "../../../../shared/ui/progress";
import type { TokenBlueprintCreateProgress } from "../model/tokenBlueprintCreateProgress";

export type TokenBlueprintCreateProgressModalProps = {
  open: boolean;
  progress: TokenBlueprintCreateProgress;
  retrying?: boolean;
  onClose?: () => void;
  onRetry?: () => void;
};

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
  const canClose = !progress.isBlockingNavigation && Boolean(onClose);

  const canRetry =
    progress.phase === "failed_retryable" &&
    progress.canRetry &&
    Boolean(onRetry);

  const statusLabel = phaseStatusLabel(progress);
  const targetLabel = uploadTargetLabel(progress.currentUploadTarget);

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
        <ProgressIndeterminate ariaLabel="作成準備中" />
      ) : null}

      {shouldShowProgressBar(progress) ? (
        <Progress
          value={progress.percentage}
          label="ファイル転送"
          ariaLabel="ファイル転送進捗"
          transferredBytes={progress.transferredBytes}
          totalBytes={progress.totalBytes}
        />
      ) : null}

      {progress.phase === "uploading" && progress.currentFileName ? (
        <ProgressCurrent
          label={targetLabel || "ファイル"}
          value={progress.currentFileName}
        />
      ) : null}

      {shouldShowUploadCount(progress) ? (
        <ProgressMetric
          label="転送済みファイル"
          value={`${progress.completedUploadCount} / ${progress.expectedUploadCount}`}
        />
      ) : null}

      {progress.phase === "uploading" ? (
        <ProgressMessage variant="warning">
          ファイル転送中は、この画面を閉じたり別のページへ移動したりしないでください。
        </ProgressMessage>
      ) : null}

      {progress.phase === "queued" || progress.phase === "processing" ? (
        <ProgressMessage variant="notice">
          ファイル転送は完了しています。ここからの処理はサーバー側で継続されます。
        </ProgressMessage>
      ) : null}

      {progress.errorMessage ? (
        <ProgressMessage variant="error">
          {progress.errorMessage}
        </ProgressMessage>
      ) : null}

      {progress.phase === "failed_retryable" ? (
        <ProgressMetric
          label="再試行回数"
          value={`${progress.retryCount} / ${progress.maxRetries}`}
        />
      ) : null}
    </Modal>
  );
}