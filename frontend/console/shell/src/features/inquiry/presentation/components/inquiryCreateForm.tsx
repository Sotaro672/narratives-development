// frontend/console/shell/src/features/inquiry/presentation/components/inquiryCreateForm.tsx

import { Image as ImageIcon } from "lucide-react";

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";
import { ErrorMessage } from "../../../../shared/ui/error";
import { Label } from "../../../../shared/ui/label";
import MediaGallery from "../../../../shared/ui/mediaGallery";
import MediaUploader from "../../../../shared/ui/mediaUploader";
import { Progress } from "../../../../shared/ui/progress";
import Stack from "../../../../shared/ui/stack";
import Text from "../../../../shared/ui/text";
import Textarea from "../../../../shared/ui/textarea";

import type { InquiryCreateAttachment } from "../hooks/useInquiryCreate";

import "./inquiryCreateForm.css";

type InquiryCreateFormProps = {
  message: string;
  attachments: InquiryCreateAttachment[];
  submitting: boolean;
  uploadProgress: number;
  errorMessage: string | null;
  maxMessageLength: number;
  maxImages: number;
  maxImageSizeMB: number;
  onChangeMessage: (value: string) => void;
  onChangeFiles: (files: File[]) => void;
  onRemoveAttachment: (id: string) => void;
};

type InquiryAttachmentCardProps = {
  attachments: InquiryCreateAttachment[];
  submitting: boolean;
  uploadProgress: number;
  maxImages: number;
  maxImageSizeMB: number;
  onChangeFiles: (files: File[]) => void;
  onRemoveAttachment: (id: string) => void;
};

function InquiryAttachmentCard({
  attachments,
  submitting,
  uploadProgress,
  maxImages,
  maxImageSizeMB,
  onChangeFiles,
  onRemoveAttachment,
}: InquiryAttachmentCardProps) {
  const hasAttachments = attachments.length > 0;
  const remainingCount = Math.max(
    maxImages - attachments.length,
    0,
  );

  const mediaItems = attachments.map((attachment) => ({
    id: attachment.id,
    src: attachment.previewUrl,
    name: attachment.file.name,
    alt: attachment.file.name,
    type: "image" as const,
    contentType: attachment.file.type,
  }));

  return (
    <Card>
      <CardHeader>
        <CardTitle>添付ファイル</CardTitle>

        {hasAttachments && remainingCount > 0 ? (
          <MediaUploader
            accept="image/*"
            multiple
            maxFiles={remainingCount}
            variant="grid"
            pickerVariant="button"
            title={null}
            showCount={false}
            showFileNames={false}
            showRemoveButton={false}
            previewFramed={false}
            pickerLabel="画像を追加"
            disabled={submitting}
            className="inquiry-create-form__header-uploader"
            onFilesSelected={onChangeFiles}
          />
        ) : null}
      </CardHeader>

      <CardContent>
        <Stack gap="md">
          {!hasAttachments ? (
            <MediaUploader
              accept="image/*"
              multiple
              maxFiles={maxImages}
              variant="grid"
              pickerVariant="dropzone"
              title={null}
              showCount={false}
              pickerLabel="画像を選択"
              pickerDescription={`JPG / PNG / WebP / GIF、1枚 ${maxImageSizeMB}MBまで`}
              emptyIcon={<ImageIcon />}
              disabled={submitting}
              className="inquiry-create-form__uploader"
              onFilesSelected={onChangeFiles}
            />
          ) : (
            <MediaGallery
              items={mediaItems}
              editable
              deleteDisabled={submitting}
              mainVariant="viewer"
              mainFit="contain"
              thumbnailFit="cover"
              emptyIcon={<ImageIcon />}
              emptyText="添付画像はありません"
              onDelete={(item) => {
                onRemoveAttachment(item.id);
              }}
            />
          )}

          {submitting && hasAttachments ? (
            <Progress
              value={uploadProgress}
              label="添付ファイルをアップロード中"
              ariaLabel="添付ファイルのアップロード進捗"
            />
          ) : null}
        </Stack>
      </CardContent>
    </Card>
  );
}

export default function InquiryCreateForm({
  message,
  attachments,
  submitting,
  uploadProgress,
  errorMessage,
  maxMessageLength,
  maxImages,
  maxImageSizeMB,
  onChangeMessage,
  onChangeFiles,
  onRemoveAttachment,
}: InquiryCreateFormProps) {
  return (
    <Stack gap="lg">
      <Card>
        <CardHeader>
          <CardTitle>問い合わせ内容</CardTitle>
        </CardHeader>

        <CardContent>
          <Stack gap="lg">
            {errorMessage ? (
              <ErrorMessage className="inquiry-create-form__error">
                {errorMessage}
              </ErrorMessage>
            ) : null}

            <Stack gap="sm">
              <Label
                htmlFor="amol-inquiry-message"
                className="inquiry-create-form__label"
              >
                本文
              </Label>

              <Textarea
                id="amol-inquiry-message"
                size="large"
                value={message}
                rows={10}
                maxLength={maxMessageLength}
                disabled={submitting}
                placeholder="AMOLへのお問い合わせ内容を入力してください"
                onChange={(event) =>
                  onChangeMessage(event.target.value)
                }
              />

              <Text
                as="div"
                size="xs"
                tone="muted"
                className="inquiry-create-form__counter"
              >
                {message.length.toLocaleString()} /{" "}
                {maxMessageLength.toLocaleString()}
              </Text>
            </Stack>
          </Stack>
        </CardContent>
      </Card>

      <InquiryAttachmentCard
        attachments={attachments}
        submitting={submitting}
        uploadProgress={uploadProgress}
        maxImages={maxImages}
        maxImageSizeMB={maxImageSizeMB}
        onChangeFiles={onChangeFiles}
        onRemoveAttachment={onRemoveAttachment}
      />
    </Stack>
  );
}