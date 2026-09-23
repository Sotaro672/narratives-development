// frontend/console/shell/src/features/inquiry/presentation/components/inquiryCreateForm.tsx

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";
import { ErrorMessage } from "../../../../shared/ui/error";
import { Label } from "../../../../shared/ui/label";
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
              onChange={(event) => onChangeMessage(event.target.value)}
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

          <MediaUploader
            items={mediaItems}
            accept="image/*"
            multiple
            maxFiles={maxImages}
            variant="grid"
            pickerVariant="dropzone"
            mediaVariant="square"
            mediaFit="cover"
            title="添付ファイル"
            pickerLabel="画像を選択"
            pickerDescription={`JPG / PNG / WebP / GIF、1枚 ${maxImageSizeMB}MBまで`}
            disabled={submitting}
            onFilesSelected={onChangeFiles}
            onRemove={onRemoveAttachment}
          />

          {submitting && attachments.length > 0 ? (
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