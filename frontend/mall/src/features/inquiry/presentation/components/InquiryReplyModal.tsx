// frontend/amol/src/features/inquiry/presentation/components/InquiryReplyModal.tsx

import {
  type ChangeEvent,
  useRef,
} from "react";

import Button from "../../../../components/ui/Button";
import ChatComposerModal from "../../../shared/presentation/components/ChatComposerModal";

type InquiryReplyModalProps = {
  open: boolean;
  content: string;
  files: File[];
  error?: string | null;
  submitting: boolean;
  canSubmit: boolean;
  onContentChange: (value: string) => void;
  onFilesChange: (event: ChangeEvent<HTMLInputElement>) => void;
  onRemoveFile: (index: number) => void;
  onCancel: () => void;
  onSubmit: () => void;
};

export default function InquiryReplyModal({
  open,
  content,
  files,
  error,
  submitting,
  canSubmit,
  onContentChange,
  onFilesChange,
  onRemoveFile,
  onCancel,
  onSubmit,
}: InquiryReplyModalProps) {
  const fileInputRef = useRef<HTMLInputElement>(null);

  return (
    <ChatComposerModal
      open={open}
      title="返信する"
      content={content}
      placeholder="返信内容を入力"
      error={error}
      submitting={submitting}
      canSubmit={canSubmit}
      submitLabel="送信"
      submittingLabel="送信中..."
      onContentChange={onContentChange}
      onCancel={onCancel}
      onSubmit={onSubmit}
      rows={6}
      maxLength={null}
      afterInput={
        <>
          <input
            ref={fileInputRef}
            type="file"
            accept="image/*"
            multiple
            onChange={onFilesChange}
            disabled={submitting}
            hidden
          />

          <Button
            type="button"
            variant="secondary"
            size="sm"
            disabled={submitting}
            onClick={() => fileInputRef.current?.click()}
          >
            画像を追加
          </Button>

          {files.length > 0 ? (
            <div className="chat-detail-page__selected-files">
              {files.map((file, index) => (
                <div
                  key={`${file.name}-${file.lastModified}-${index}`}
                  className="chat-detail-page__selected-file"
                >
                  <span>{file.name}</span>

                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    disabled={submitting}
                    onClick={() => onRemoveFile(index)}
                  >
                    削除
                  </Button>
                </div>
              ))}
            </div>
          ) : null}
        </>
      }
    />
  );
}