// frontend/console/shell/src/features/inquiry/presentation/components/inquiryCreateForm.tsx

import type { ChangeEventHandler } from "react";

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";
import DeleteButton from "../../../../shared/ui/delete";
import { ErrorMessage } from "../../../../shared/ui/error";
import { Label } from "../../../../shared/ui/label";
import Media from "../../../../shared/ui/media";
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
  onChangeFiles: ChangeEventHandler<HTMLInputElement>;
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

          <div className="inquiry-create-form__attachments">
            <div className="inquiry-create-form__attachment-header">
              <Text
                weight="semibold"
                className="inquiry-create-form__attachment-title"
              >
                添付ファイル
              </Text>

              <Text
                size="xs"
                tone="muted"
                className="inquiry-create-form__attachment-count"
              >
                {attachments.length} / {maxImages}
              </Text>
            </div>

            <label className="inquiry-create-form__file-picker">
              <input
                type="file"
                accept="image/*"
                multiple
                className="inquiry-create-form__file-input"
                disabled={submitting || attachments.length >= maxImages}
                onChange={onChangeFiles}
              />

              <Text
                weight="semibold"
                className="inquiry-create-form__file-picker-title"
              >
                画像を選択
              </Text>

              <Text
                size="xs"
                tone="muted"
                className="inquiry-create-form__file-picker-help"
              >
                JPG / PNG / WebP / GIF、1枚 {maxImageSizeMB}MBまで
              </Text>
            </label>

            {attachments.length > 0 ? (
              <div className="inquiry-create-form__attachment-grid">
                {attachments.map((attachment) => (
                  <div
                    key={attachment.id}
                    className="inquiry-create-form__attachment"
                  >
                    <Media
                      src={attachment.previewUrl}
                      alt={attachment.file.name}
                      name={attachment.file.name}
                      variant="square"
                      fit="cover"
                      className="inquiry-create-form__attachment-media"
                    />

                    <DeleteButton
                      size="sm"
                      disabled={submitting}
                      ariaLabel={`${attachment.file.name}を削除`}
                      onClick={() => onRemoveAttachment(attachment.id)}
                    />

                    <Text
                      as="div"
                      size="xs"
                      wrap="nowrap"
                      className="inquiry-create-form__attachment-name"
                      title={attachment.file.name}
                    >
                      {attachment.file.name}
                    </Text>
                  </div>
                ))}
              </div>
            ) : null}
          </div>

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