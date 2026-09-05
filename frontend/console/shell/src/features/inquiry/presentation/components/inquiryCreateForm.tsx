// frontend/console/shell/src/features/inquiry/presentation/components/inquiryCreateForm.tsx

import type { ChangeEventHandler } from "react";

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";

import type { InquiryCreateAttachment } from "../hooks/useInquiryCreate";

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
        <div className="flex flex-col gap-6">
          {errorMessage ? (
            <div className="inq__empty">
              {errorMessage}
            </div>
          ) : null}

          <div className="flex flex-col gap-2">
            <label
              htmlFor="amol-inquiry-message"
              className="text-sm font-semibold"
            >
              本文
            </label>

            <textarea
              id="amol-inquiry-message"
              value={message}
              rows={10}
              maxLength={maxMessageLength}
              disabled={submitting}
              placeholder="AMOLへのお問い合わせ内容を入力してください"
              className="min-h-[240px] w-full resize-y rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--background))] px-4 py-3 text-sm leading-7 text-[hsl(var(--foreground))] outline-none transition focus:border-[hsl(var(--ring))] focus:ring-2 focus:ring-[hsl(var(--ring)/0.2)] disabled:cursor-not-allowed disabled:opacity-60"
              onChange={(event) =>
                onChangeMessage(event.target.value)
              }
            />

            <div className="text-right text-xs text-[hsl(var(--muted-foreground))]">
              {message.length.toLocaleString()} /{" "}
              {maxMessageLength.toLocaleString()}
            </div>
          </div>

          <div className="flex flex-col gap-3">
            <div className="flex items-center justify-between gap-4">
              <span className="text-sm font-semibold">
                添付ファイル
              </span>

              <span className="text-xs text-[hsl(var(--muted-foreground))]">
                {attachments.length} / {maxImages}
              </span>
            </div>

            <label className="flex cursor-pointer flex-col items-center justify-center gap-1 rounded-lg border border-dashed border-[hsl(var(--border))] px-6 py-8 text-center transition hover:bg-[hsl(var(--muted)/0.5)]">
              <input
                type="file"
                accept="image/*"
                multiple
                className="hidden"
                disabled={
                  submitting ||
                  attachments.length >= maxImages
                }
                onChange={onChangeFiles}
              />

              <span className="text-sm font-semibold">
                画像を選択
              </span>

              <span className="text-xs text-[hsl(var(--muted-foreground))]">
                JPG / PNG / WebP / GIF、1枚
                {maxImageSizeMB}MBまで
              </span>
            </label>

            {attachments.length > 0 ? (
              <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4">
                {attachments.map((attachment) => (
                  <div
                    key={attachment.id}
                    className="relative overflow-hidden rounded-lg border border-[hsl(var(--border))]"
                  >
                    <img
                      src={attachment.previewUrl}
                      alt={attachment.file.name}
                      className="aspect-square w-full object-cover"
                    />

                    <button
                      type="button"
                      disabled={submitting}
                      className="absolute right-2 top-2 flex h-7 w-7 items-center justify-center rounded-full bg-black/65 text-sm text-white transition hover:bg-black/80 disabled:cursor-not-allowed disabled:opacity-50"
                      onClick={() =>
                        onRemoveAttachment(attachment.id)
                      }
                      aria-label={`${attachment.file.name}を削除`}
                    >
                      ×
                    </button>

                    <div className="truncate px-2 py-2 text-xs">
                      {attachment.file.name}
                    </div>
                  </div>
                ))}
              </div>
            ) : null}
          </div>

          {submitting && attachments.length > 0 ? (
            <div className="flex flex-col gap-2">
              <div className="flex items-center justify-between text-xs text-[hsl(var(--muted-foreground))]">
                <span>添付ファイルをアップロード中</span>
                <span>{uploadProgress}%</span>
              </div>

              <div className="h-2 overflow-hidden rounded-full bg-[hsl(var(--muted))]">
                <div
                  className="h-full bg-[hsl(var(--primary))] transition-[width]"
                  style={{
                    width: `${uploadProgress}%`,
                  }}
                />
              </div>
            </div>
          ) : null}
        </div>
      </CardContent>
    </Card>
  );
}