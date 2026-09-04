// frontend/mall/src/features/contact/components/ContactUploadProgressModal.tsx
import { createPortal } from "react-dom";

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
  if (!open) {
    return null;
  }

  const totalProgress = clampProgress(progress);
  const currentFileProgress = clampProgress(fileProgress);
  const currentFileIndex = Math.min(
    Math.max(fileIndex, 1),
    Math.max(fileCount, 1),
  );

  const modal = (
    <div
      className="contact-upload-progress-modal"
      role="presentation"
    >
      <div
        className="contact-upload-progress-modal__panel"
        role="dialog"
        aria-modal="true"
        aria-labelledby="contact-upload-progress-modal-title"
        aria-describedby="contact-upload-progress-modal-description"
        aria-busy="true"
      >
        <div className="contact-upload-progress-modal__header">
          <span className="contact-upload-progress-modal__status">
            転送中
          </span>

          <h2
            id="contact-upload-progress-modal-title"
            className="contact-upload-progress-modal__title"
          >
            添付画像を送信しています
          </h2>
        </div>

        <div className="contact-upload-progress-modal__body">
          <p
            id="contact-upload-progress-modal-description"
            className="contact-upload-progress-modal__description"
          >
            画像転送が完了するまで、この画面を閉じたり別のページへ移動したりしないでください。
          </p>

          <div className="contact-upload-progress-modal__progress-section">
            <div className="contact-upload-progress-modal__progress-header">
              <span>全体の進捗</span>
              <strong>{totalProgress}%</strong>
            </div>

            <div
              className="contact-upload-progress-modal__progress"
              role="progressbar"
              aria-label="添付画像全体の転送進捗"
              aria-valuemin={0}
              aria-valuemax={100}
              aria-valuenow={totalProgress}
            >
              <div
                className="contact-upload-progress-modal__progress-bar"
                style={{ width: `${totalProgress}%` }}
              />
            </div>
          </div>

          <div className="contact-upload-progress-modal__file-section">
            <div className="contact-upload-progress-modal__file-header">
              <span>
                画像 {currentFileIndex} / {fileCount}
              </span>
              <strong>{currentFileProgress}%</strong>
            </div>

            <div
              className="contact-upload-progress-modal__file-progress"
              role="progressbar"
              aria-label={`${currentFileIndex}枚目の画像の転送進捗`}
              aria-valuemin={0}
              aria-valuemax={100}
              aria-valuenow={currentFileProgress}
            >
              <div
                className="contact-upload-progress-modal__file-progress-bar"
                style={{ width: `${currentFileProgress}%` }}
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  );

  return createPortal(modal, document.body);
}