// frontend/mall/src/features/contact/components/ContactUploadProgressModal.tsx

import Badge from "../../../components/ui/Badge";
import Modal, { ModalBody, ModalDescription, ModalHeader, ModalTitle } from "../../../components/ui/Modal";
import Progress from "../../../components/ui/Progress";

type ContactUploadProgressModalProps = {
  open: boolean;
  progress: number;
  fileProgress: number;
  fileIndex: number;
  fileCount: number;
};

function clampProgress(value: number): number {
  if (!Number.isFinite(value)) {
    return 0;
  }

  return Math.min(100, Math.max(0, Math.round(value)));
}

export default function ContactUploadProgressModal({
  open,
  progress,
  fileProgress,
  fileIndex,
  fileCount,
}: ContactUploadProgressModalProps) {
  const totalProgress = clampProgress(progress);
  const currentFileProgress = clampProgress(fileProgress);
  const safeFileCount = Math.max(fileCount, 1);
  const currentFileIndex = Math.min(Math.max(fileIndex, 1), safeFileCount);

  return (
    <Modal
      open={open}
      size="sm"
      closeOnBackdrop={false}
      closeOnEscape={false}
      ariaLabelledBy="contact-upload-progress-modal-title"
      ariaDescribedBy="contact-upload-progress-modal-description"
      ariaBusy
      panelClassName="contact-upload-progress-modal__panel"
    >
      <ModalHeader className="contact-upload-progress-modal__header">
        <Badge variant="info" size="md">転送中</Badge>
        <ModalTitle id="contact-upload-progress-modal-title">添付画像を送信しています</ModalTitle>
      </ModalHeader>

      <ModalBody className="contact-upload-progress-modal__body">
        <ModalDescription id="contact-upload-progress-modal-description">
          画像転送が完了するまで、この画面を閉じたり別のページへ移動したりしないでください。
        </ModalDescription>

        <div className="contact-upload-progress-modal__progress-section">
          <div className="contact-upload-progress-modal__progress-header">
            <span>全体の進捗</span>
            <strong>{totalProgress}%</strong>
          </div>
          <Progress
            aria-label="添付画像全体の転送進捗"
            value={totalProgress}
            max={100}
            variant="info"
          />
        </div>

        <div className="contact-upload-progress-modal__file-section">
          <div className="contact-upload-progress-modal__file-header">
            <span>画像 {currentFileIndex} / {fileCount}</span>
            <strong>{currentFileProgress}%</strong>
          </div>
          <Progress
            aria-label={`${currentFileIndex}枚目の画像の転送進捗`}
            value={currentFileProgress}
            max={100}
            variant="info"
          />
        </div>
      </ModalBody>
    </Modal>
  );
}