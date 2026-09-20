// frontend/mall/src/features/avatar/components/AvatarCreateProgressModal.tsx

import Alert from "../../../components/ui/Alert";
import Badge from "../../../components/ui/Badge";
import Button from "../../../components/ui/Button";
import Modal, {
  ModalBody,
  ModalDescription,
  ModalFooter,
  ModalHeader,
  ModalTitle,
} from "../../../components/ui/Modal";
import Progress from "../../../components/ui/Progress";
import type { AvatarCreateProgress } from "../models/avatarCreateProgress";

import "../../../styles/avatar-create-progress.css";

export type AvatarCreateProgressModalProps = {
  open: boolean;
  progress: AvatarCreateProgress;
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

function getStatusLabel(progress: AvatarCreateProgress): string {
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

function getStatusVariant(
  progress: AvatarCreateProgress,
): "neutral" | "info" | "purple" | "danger" {
  switch (progress.phase) {
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

function shouldShowIndeterminate(progress: AvatarCreateProgress): boolean {
  return progress.phase === "preparing" || progress.phase === "saving";
}

function shouldShowProgressBar(progress: AvatarCreateProgress): boolean {
  return (
    progress.totalBytes > 0 &&
    (progress.phase === "uploading" || progress.phase === "saving")
  );
}

export default function AvatarCreateProgressModal({
  open,
  progress,
  onClose,
}: AvatarCreateProgressModalProps) {
  const canClose = !progress.isBlockingNavigation && Boolean(onClose);
  const closeHandler = canClose ? onClose : undefined;
  const statusLabel = getStatusLabel(progress);
  const progressPercentage = Math.min(100, Math.max(0, progress.percentage));

  return (
    <Modal
      open={open}
      onClose={closeHandler}
      size="md"
      mobilePosition="bottom"
      closeOnBackdrop={canClose}
      closeOnEscape={canClose}
      ariaLabelledBy="avatar-create-progress-modal-title"
      ariaDescribedBy="avatar-create-progress-modal-description"
      ariaBusy={
        progress.phase === "preparing" ||
        progress.phase === "uploading" ||
        progress.phase === "saving"
      }
    >
      <ModalHeader onClose={closeHandler} closeLabel="進捗画面を閉じる">
        <div className="avatar-create-progress-modal__heading">
          {statusLabel ? (
            <Badge variant={getStatusVariant(progress)}>
              {statusLabel}
            </Badge>
          ) : null}

          <ModalTitle id="avatar-create-progress-modal-title">
            {progress.title}
          </ModalTitle>
        </div>
      </ModalHeader>

      <ModalBody className="avatar-create-progress-modal__body">
        <ModalDescription id="avatar-create-progress-modal-description">
          {progress.message}
        </ModalDescription>

        {shouldShowIndeterminate(progress) ? (
          <Progress
            indeterminate
            variant="info"
            size="sm"
            aria-label={progress.phase === "saving" ? "保存中" : "準備中"}
          />
        ) : null}

        {shouldShowProgressBar(progress) ? (
          <div className="avatar-create-progress-modal__progress-section">
            <div className="avatar-create-progress-modal__progress-header">
              <span className="avatar-create-progress-modal__progress-label">
                画像転送
              </span>
              <span className="avatar-create-progress-modal__progress-percentage">
                {progressPercentage}%
              </span>
            </div>

            <Progress
              value={progressPercentage}
              variant="info"
              size="md"
              aria-label="アバターアイコンの転送進捗"
            />

            <div className="avatar-create-progress-modal__bytes">
              {formatBytes(progress.transferredBytes)}
              {" / "}
              {formatBytes(progress.totalBytes)}
            </div>
          </div>
        ) : null}

        {progress.phase === "uploading" && progress.currentFileName ? (
          <div className="avatar-create-progress-modal__current">
            <span className="avatar-create-progress-modal__current-label">
              画像
            </span>
            <span
              className="avatar-create-progress-modal__current-file"
              title={progress.currentFileName}
            >
              {progress.currentFileName}
            </span>
          </div>
        ) : null}

        {progress.phase === "uploading" && progress.isBrowserDependent ? (
          <Alert variant="warning">
            画像転送が完了するまで、この画面を閉じたり別のページへ移動したりしないでください。
          </Alert>
        ) : null}

        {progress.phase === "saving" ? (
          <Alert variant="info">
            画像転送は完了しています。アバター情報の保存処理を続けています。
          </Alert>
        ) : null}

        {progress.errorMessage ? (
          <Alert variant="error">{progress.errorMessage}</Alert>
        ) : null}
      </ModalBody>

      {progress.phase === "failed" && canClose ? (
        <ModalFooter className="avatar-create-progress-modal__actions">
          <Button
            size="md"
            className="avatar-create-progress-modal__button"
            onClick={onClose}
          >
            閉じる
          </Button>
        </ModalFooter>
      ) : null}
    </Modal>
  );
}