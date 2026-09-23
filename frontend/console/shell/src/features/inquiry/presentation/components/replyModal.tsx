// frontend/console/shell/src/features/inquiry/presentation/components/replyModal.tsx

import { ErrorMessage } from "../../../../shared/ui/error";
import { Label } from "../../../../shared/ui/label";
import MediaUploader from "../../../../shared/ui/mediaUploader";
import {
  Modal,
  ModalButton,
} from "../../../../shared/ui/modal";
import Stack from "../../../../shared/ui/stack";
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
  onChangeImages: (files: File[]) => void;
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
  const uploadItems = images.map((image) => ({
    id: image.id,
    src: image.previewUrl,
    name: image.file.name,
    alt: image.file.name,
    type: "image" as const,
    contentType: image.file.type,
  }));

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
        <ModalButton
          variant="primary"
          disabled={
            submitting ||
            (!content.trim() && images.length === 0)
          }
          onClick={onSubmit}
        >
          {submitting ? "送信中" : "送信"}
        </ModalButton>
      }
    >
      {errorMessage ? (
        <ErrorMessage>
          {errorMessage}
        </ErrorMessage>
      ) : null}

      <Stack gap="xs">
        <Label htmlFor="inquiry-reply-content">
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
      </Stack>

      <MediaUploader
        items={uploadItems}
        accept="image/*"
        multiple
        maxFiles={MAX_REPLY_IMAGES}
        variant="grid"
        mediaVariant="square"
        mediaFit="cover"
        title="添付画像"
        pickerLabel="画像を選択"
        pickerDescription={`JPG / PNG / WebP / GIF、1枚 ${MAX_REPLY_IMAGE_SIZE_MB}MBまで`}
        disabled={submitting}
        showCount
        showFileNames={false}
        showRemoveButton
        onFilesSelected={onChangeImages}
        onRemove={onRemoveImage}
      />
    </Modal>
  );
}