// frontend/console/shell/src/features/inquiry/presentation/components/replyModal.tsx

import * as React from "react";

import DeleteButton from "../../../../shared/ui/delete";
import { ErrorMessage } from "../../../../shared/ui/error";
import { Label } from "../../../../shared/ui/label";
import {
  Modal,
  ModalButton,
  ModalCloseButton,
} from "../../../../shared/ui/modal";
import Text from "../../../../shared/ui/text";
import Textarea from "../../../../shared/ui/textarea";

import {
  MAX_REPLY_IMAGES,
  MAX_REPLY_IMAGE_SIZE_MB,
} from "../../constants/inquiryReply";

export type ReplyUploadImage = {
  id: string;
  file: File;
  previewUrl: string;
};

type ReplyModalProps = {
  open: boolean;
  content: string;
  images: ReplyUploadImage[];
  submitting: boolean;
  errorMessage: string | null;
  onClose: () => void;
  onChangeContent: (value: string) => void;
  onChangeImages: React.ChangeEventHandler<HTMLInputElement>;
  onRemoveImage: (id: string) => void;
  onSubmit: () => void;
};

export default function ReplyModal({
  open,
  content,
  images,
  submitting,
  errorMessage,
  onClose,
  onChangeContent,
  onChangeImages,
  onRemoveImage,
  onSubmit,
}: ReplyModalProps) {
  return (
    <Modal
      open={open}
      title="返信を入力"
      description="この問い合わせに対する返信内容を入力してください。"
      onClose={onClose}
      closeable={!submitting}
      closeLabel="返信モーダルを閉じる"
      ariaBusy={submitting}
      panelClassName="inq-reply-modal__panel"
      footer={
        <>
          <ModalCloseButton
            onClick={onClose}
            disabled={submitting}
          >
            キャンセル
          </ModalCloseButton>

          <ModalButton
            variant="primary"
            disabled={submitting || !content.trim()}
            onClick={onSubmit}
          >
            {submitting ? "送信中" : "送信"}
          </ModalButton>
        </>
      }
    >
      {errorMessage ? (
        <ErrorMessage>
          {errorMessage}
        </ErrorMessage>
      ) : null}

      <Label
        className="inq-reply-modal__label"
        htmlFor="inquiry-reply-content"
      >
        返信内容
      </Label>

      <Textarea
        id="inquiry-reply-content"
        size="medium"
        value={content}
        placeholder="返信内容を入力してください"
        rows={8}
        maxLength={2000}
        disabled={submitting}
        onChange={(event) => onChangeContent(event.target.value)}
      />

      <Text
        as="div"
        size="xs"
        tone="muted"
        className="inq-reply-modal__counter"
      >
        {content.length.toLocaleString()} / 2,000
      </Text>

      <div className="inq-reply-modal__upload">
        <div className="inq-reply-modal__upload-header">
          <Text size="sm" weight="bold">
            添付画像
          </Text>

          <Text size="xs" tone="muted" weight="bold">
            {images.length} / {MAX_REPLY_IMAGES}
          </Text>
        </div>

        <label className="inq-reply-modal__upload-box">
          <input
            type="file"
            accept="image/*"
            multiple
            className="inq-reply-modal__upload-input"
            disabled={
              submitting ||
              images.length >= MAX_REPLY_IMAGES
            }
            onChange={onChangeImages}
          />

          <Text size="sm" weight="bold">
            画像を選択
          </Text>

          <Text size="xs" tone="muted">
            JPG / PNG / WebP / GIF、1枚 {MAX_REPLY_IMAGE_SIZE_MB}MBまで
          </Text>
        </label>

        {images.length > 0 ? (
          <div className="inq-reply-modal__preview-grid">
            {images.map((image) => (
              <div
                key={image.id}
                className="inq-reply-modal__preview-item"
              >
                <img
                  src={image.previewUrl}
                  alt={image.file.name}
                  className="inq-reply-modal__preview-image"
                />

                <DeleteButton
                  size="sm"
                  disabled={submitting}
                  ariaLabel={`${image.file.name}を削除`}
                  onClick={() => onRemoveImage(image.id)}
                />
              </div>
            ))}
          </div>
        ) : null}
      </div>
    </Modal>
  );
}