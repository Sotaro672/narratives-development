// frontend/mall/src/features/shared/presentation/components/ProgressModal.tsx

import Alert from "../../../../components/ui/Alert";
import Badge from "../../../../components/ui/Badge";
import Button from "../../../../components/ui/Button";
import InfoList, { InfoRow } from "../../../../components/ui/InfoList";
import Modal, {
  ModalBody,
  ModalDescription,
  ModalFooter,
  ModalHeader,
  ModalTitle,
} from "../../../../components/ui/Modal";
import Progress from "../../../../components/ui/Progress";

import "../../styles/progress-modal.css";

export type ProgressModalPhase =
  | "idle"
  | "preparing"
  | "uploading"
  | "saving"
  | "failed";

export type ProgressModalProgress = {
  phase: ProgressModalPhase;
  title: string;
  message: string;
  percentage: number;
  transferredBytes: number;
  totalBytes: number;
  currentFileName: string;
  errorMessage: string;
  isBrowserDependent: boolean;
  isBlockingNavigation: boolean;
  completedUploadCount?: number;
  expectedUploadCount?: number;
};

export type ProgressModalProps = {
  open: boolean;
  progress: ProgressModalProgress;
  idPrefix: string;
  progressAriaLabel: string;
  savingNotice: string;
  onClose?: () => void;
  progressLabel?: string;
  currentFileLabel?: string;
  uploadCountLabel?: string;
  browserDependentMessage?: string;
  closeLabel?: string;
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

function getStatusLabel(phase: ProgressModalPhase): string {
  switch (phase) {
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

function getStatusVariant(
  phase: ProgressModalPhase,
): "neutral" | "info" | "purple" | "danger" {
  switch (phase) {
    case "idle":
    case "preparing":
      return "neutral";
    case "uploading":
      return "info";
    case "saving":
      return "purple";
    case "failed":
      return "danger";
  }
}

function shouldShowIndeterminate(phase: ProgressModalPhase): boolean {
  return phase === "preparing" || phase === "saving";
}

function shouldShowProgressBar(progress: ProgressModalProgress): boolean {
  return (
    progress.totalBytes > 0 &&
    (progress.phase === "uploading" || progress.phase === "saving")
  );
}

function shouldShowCurrentFile(progress: ProgressModalProgress): boolean {
  return progress.phase === "uploading" && Boolean(progress.currentFileName);
}

function shouldShowUploadCount(progress: ProgressModalProgress): boolean {
  return (
    (progress.expectedUploadCount ?? 0) > 0 &&
    (progress.phase === "uploading" || progress.phase === "saving")
  );
}

export default function ProgressModal({
  open,
  progress,
  idPrefix,
  progressAriaLabel,
  savingNotice,
  onClose,
  progressLabel = "画像転送",
  currentFileLabel = "画像",
  uploadCountLabel = "転送済み画像",
  browserDependentMessage = "画像転送が完了するまで、この画面を閉じたり別のページへ移動したりしないでください。",
  closeLabel = "進捗画面を閉じる",
}: ProgressModalProps) {
  const canClose = !progress.isBlockingNavigation && Boolean(onClose);
  const closeHandler = canClose ? onClose : undefined;
  const statusLabel = getStatusLabel(progress.phase);
  const progressPercentage = Math.min(100, Math.max(0, progress.percentage));
  const showCurrentFile = shouldShowCurrentFile(progress);
  const showUploadCount = shouldShowUploadCount(progress);
  const titleId = `${idPrefix}-title`;
  const descriptionId = `${idPrefix}-description`;

  return (
    <Modal
      open={open}
      onClose={closeHandler}
      size="md"
      mobilePosition="bottom"
      closeOnBackdrop={canClose}
      closeOnEscape={canClose}
      ariaLabelledBy={titleId}
      ariaDescribedBy={descriptionId}
      ariaBusy={
        progress.phase === "preparing" ||
        progress.phase === "uploading" ||
        progress.phase === "saving"
      }
    >
      <ModalHeader onClose={closeHandler} closeLabel={closeLabel}>
        <div className="progress-modal__heading">
          {statusLabel ? (
            <Badge variant={getStatusVariant(progress.phase)}>
              {statusLabel}
            </Badge>
          ) : null}

          <ModalTitle id={titleId}>{progress.title}</ModalTitle>
        </div>
      </ModalHeader>

      <ModalBody className="progress-modal__body">
        <ModalDescription id={descriptionId}>
          {progress.message}
        </ModalDescription>

        {shouldShowIndeterminate(progress.phase) ? (
          <Progress
            indeterminate
            variant="info"
            size="sm"
            aria-label={progress.phase === "saving" ? "保存中" : "準備中"}
          />
        ) : null}

        {shouldShowProgressBar(progress) ? (
          <div className="progress-modal__progress-section">
            <div className="progress-modal__progress-header">
              <span className="progress-modal__progress-label">
                {progressLabel}
              </span>
              <span className="progress-modal__progress-percentage">
                {progressPercentage}%
              </span>
            </div>

            <Progress
              value={progressPercentage}
              variant="info"
              size="md"
              aria-label={progressAriaLabel}
            />

            <div className="progress-modal__bytes">
              {formatBytes(progress.transferredBytes)}
              {" / "}
              {formatBytes(progress.totalBytes)}
            </div>
          </div>
        ) : null}

        {showCurrentFile || showUploadCount ? (
          <InfoList className="progress-modal__info">
            {showCurrentFile ? (
              <InfoRow label={currentFileLabel}>
                <span
                  className="progress-modal__file-name"
                  title={progress.currentFileName}
                >
                  {progress.currentFileName}
                </span>
              </InfoRow>
            ) : null}

            {showUploadCount ? (
              <InfoRow label={uploadCountLabel}>
                <span className="progress-modal__count">
                  {progress.completedUploadCount ?? 0}
                  {" / "}
                  {progress.expectedUploadCount ?? 0}
                </span>
              </InfoRow>
            ) : null}
          </InfoList>
        ) : null}

        {progress.phase === "uploading" && progress.isBrowserDependent ? (
          <Alert variant="warning">
            {browserDependentMessage}
          </Alert>
        ) : null}

        {progress.phase === "saving" ? (
          <Alert variant="info">{savingNotice}</Alert>
        ) : null}

        {progress.errorMessage ? (
          <Alert variant="error">{progress.errorMessage}</Alert>
        ) : null}
      </ModalBody>

      {progress.phase === "failed" && canClose ? (
        <ModalFooter>
          <Button size="md" onClick={onClose}>
            閉じる
          </Button>
        </ModalFooter>
      ) : null}
    </Modal>
  );
}