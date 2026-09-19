// frontend/console/shell/src/features/tokenBlueprint/presentation/components/tokenBlueprintProgressModal.tsx

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
import type {
  TokenBlueprintCreateProgress,
  TokenBlueprintProgress,
  TokenBlueprintProgressTarget,
} from "../model/tokenBlueprintProgress";

type TokenBlueprintUpdateProgressModalProps = {
  variant: "update";
  open: boolean;
  progress: TokenBlueprintProgress;
  onClose?: () => void;
};

type TokenBlueprintCreateProgressModalProps = {
  variant: "create";
  open: boolean;
  progress: TokenBlueprintCreateProgress;
  retrying?: boolean;
  onClose?: () => void;
  onRetry?: () => void;
};

export type TokenBlueprintProgressModalProps =
  | TokenBlueprintUpdateProgressModalProps
  | TokenBlueprintCreateProgressModalProps;

type ProgressModalViewModel = {
  title: string;
  message: string;
  percentage: number;
  transferredBytes: number;
  totalBytes: number;
  currentFileName?: string | null;
  currentTargetLabel: string;
  completedUploadCount: number;
  expectedUploadCount: number;
  errorMessage?: string | null;
  statusLabel: string;
  statusVariant: BadgeVariant;
  ariaBusy: boolean;
  canClose: boolean;
  showIndeterminate: boolean;
  indeterminateLabel: string;
  showProgressBar: boolean;
  showUploadCount: boolean;
  showFooter: boolean;
  warningMessage?: string;
  noticeMessage?: string;
  retryInfo?: {
    canRetry: boolean;
    retryCount: number;
    maxRetries: number;
  };
};

function uploadTargetLabel(
  target?: TokenBlueprintProgressTarget,
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

function updatePhaseStatusLabel(
  progress: TokenBlueprintProgress,
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

function updatePhaseStatusVariant(
  progress: TokenBlueprintProgress,
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

function createPhaseStatusLabel(
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

function createPhaseStatusVariant(
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

function buildUpdateViewModel(
  progress: TokenBlueprintProgress,
  onClose?: () => void,
): ProgressModalViewModel {
  const canClose = !progress.isBlockingNavigation && Boolean(onClose);

  const showProgressBar =
    progress.totalBytes > 0 &&
    (progress.phase === "uploading" ||
      progress.phase === "saving" ||
      progress.phase === "completed");

  const showUploadCount =
    progress.expectedUploadCount > 0 &&
    (progress.phase === "uploading" ||
      progress.phase === "saving" ||
      progress.phase === "completed");

  return {
    title: progress.title,
    message: progress.message,
    percentage: progress.percentage,
    transferredBytes: progress.transferredBytes,
    totalBytes: progress.totalBytes,
    currentFileName: progress.currentFileName,
    currentTargetLabel:
      uploadTargetLabel(progress.currentUploadTarget) || "ファイル",
    completedUploadCount: progress.completedUploadCount,
    expectedUploadCount: progress.expectedUploadCount,
    errorMessage: progress.errorMessage,
    statusLabel: updatePhaseStatusLabel(progress),
    statusVariant: updatePhaseStatusVariant(progress),
    ariaBusy:
      progress.phase === "preparing" ||
      progress.phase === "uploading" ||
      progress.phase === "saving",
    canClose,
    showIndeterminate:
      progress.phase === "preparing" ||
      (progress.phase === "saving" && progress.totalBytes <= 0),
    indeterminateLabel:
      progress.phase === "saving" ? "保存中" : "準備中",
    showProgressBar,
    showUploadCount,
    showFooter:
      canClose &&
      (progress.phase === "completed" || progress.phase === "failed"),
    warningMessage: progress.isBlockingNavigation
      ? "処理が完了するまで、この画面を閉じたり別のページへ移動したりしないでください。"
      : undefined,
  };
}

function buildCreateViewModel(
  progress: TokenBlueprintCreateProgress,
  onClose?: () => void,
): ProgressModalViewModel {
  const canClose = !progress.isBlockingNavigation && Boolean(onClose);

  return {
    title: progress.title,
    message: progress.message,
    percentage: progress.percentage,
    transferredBytes: progress.transferredBytes,
    totalBytes: progress.totalBytes,
    currentFileName: progress.currentFileName,
    currentTargetLabel:
      uploadTargetLabel(progress.currentUploadTarget) || "ファイル",
    completedUploadCount: progress.completedUploadCount,
    expectedUploadCount: progress.expectedUploadCount,
    errorMessage: progress.errorMessage,
    statusLabel: createPhaseStatusLabel(progress),
    statusVariant: createPhaseStatusVariant(progress),
    ariaBusy:
      progress.phase === "starting" ||
      progress.phase === "uploading" ||
      progress.phase === "queued" ||
      progress.phase === "processing",
    canClose,
    showIndeterminate: progress.phase === "starting",
    indeterminateLabel: "作成準備中",
    showProgressBar:
      progress.phase === "uploading" ||
      progress.phase === "queued" ||
      progress.phase === "processing" ||
      progress.phase === "completed",
    showUploadCount:
      progress.expectedUploadCount > 0 &&
      progress.phase !== "starting",
    showFooter:
      progress.phase === "failed_retryable" ||
      progress.phase === "failed_fatal" ||
      progress.phase === "completed",
    warningMessage:
      progress.phase === "uploading"
        ? "ファイル転送中は、この画面を閉じたり別のページへ移動したりしないでください。"
        : undefined,
    noticeMessage:
      progress.phase === "queued" || progress.phase === "processing"
        ? "ファイル転送は完了しています。ここからの処理はサーバー側で継続されます。"
        : undefined,
    retryInfo:
      progress.phase === "failed_retryable"
        ? {
            canRetry: progress.canRetry,
            retryCount: progress.retryCount,
            maxRetries: progress.maxRetries,
          }
        : undefined,
  };
}

export default function TokenBlueprintProgressModal(
  props: TokenBlueprintProgressModalProps,
) {
  const vm =
    props.variant === "create"
      ? buildCreateViewModel(props.progress, props.onClose)
      : buildUpdateViewModel(props.progress, props.onClose);

  const retrying =
    props.variant === "create"
      ? props.retrying ?? false
      : false;

  const canRetry =
    props.variant === "create" &&
    Boolean(vm.retryInfo?.canRetry) &&
    Boolean(props.onRetry);

  return (
    <Modal
      open={props.open}
      title={vm.title}
      description={vm.message}
      eyebrow={
        vm.statusLabel ? (
          <Badge variant={vm.statusVariant}>
            {vm.statusLabel}
          </Badge>
        ) : undefined
      }
      onClose={vm.canClose ? props.onClose : undefined}
      closeable={vm.canClose}
      closeOnBackdrop={vm.canClose}
      closeOnEscape={vm.canClose}
      showCloseButton={vm.canClose}
      closeLabel="進捗画面を閉じる"
      ariaBusy={vm.ariaBusy}
      footer={
        vm.showFooter ? (
          <>
            {props.variant === "create" && canRetry ? (
              <ModalButton
                variant="primary"
                disabled={retrying}
                onClick={props.onRetry}
              >
                {retrying ? "再試行中" : "再試行"}
              </ModalButton>
            ) : null}

            {vm.canClose ? (
              <ModalCloseButton
                disabled={retrying}
                onClick={props.onClose}
              >
                閉じる
              </ModalCloseButton>
            ) : null}
          </>
        ) : undefined
      }
    >
      {vm.showIndeterminate ? (
        <ProgressIndeterminate ariaLabel={vm.indeterminateLabel} />
      ) : null}

      {vm.showProgressBar ? (
        <Progress
          value={vm.percentage}
          label="ファイル転送"
          ariaLabel="ファイル転送進捗"
          transferredBytes={vm.transferredBytes}
          totalBytes={vm.totalBytes}
        />
      ) : null}

      {props.progress.phase === "uploading" && vm.currentFileName ? (
        <ProgressCurrent
          label={vm.currentTargetLabel}
          value={vm.currentFileName}
        />
      ) : null}

      {vm.showUploadCount ? (
        <ProgressMetric
          label="転送済みファイル"
          value={`${vm.completedUploadCount} / ${vm.expectedUploadCount}`}
        />
      ) : null}

      {vm.warningMessage ? (
        <ProgressMessage variant="warning">
          {vm.warningMessage}
        </ProgressMessage>
      ) : null}

      {vm.noticeMessage ? (
        <ProgressMessage variant="notice">
          {vm.noticeMessage}
        </ProgressMessage>
      ) : null}

      {vm.errorMessage ? (
        <ProgressMessage variant="error">
          {vm.errorMessage}
        </ProgressMessage>
      ) : null}

      {vm.retryInfo ? (
        <ProgressMetric
          label="再試行回数"
          value={`${vm.retryInfo.retryCount} / ${vm.retryInfo.maxRetries}`}
        />
      ) : null}
    </Modal>
  );
}