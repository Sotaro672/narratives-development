// frontend/mall/src/features/shared/presentation/components/ChatInlineComposer.tsx

import {
  ImagePlus,
  X,
} from "lucide-react";
import {
  type KeyboardEvent,
  useEffect,
  useMemo,
  useRef,
} from "react";

import IconButton from "../../../../components/ui/IconButton";
import Textbox from "../../../../components/ui/Textbox";

type ChatInlineComposerProps = {
  content: string;
  placeholder: string;
  error?: string | null;
  submitting: boolean;
  canSubmit: boolean;
  disabled?: boolean;
  maxLength?: number | null;
  files?: File[];
  onContentChange: (value: string) => void;
  onFilesAdd?: (files: File[]) => void;
  onRemoveFile?: (index: number) => void;
  onSubmit: () => void | Promise<void>;
};

type ChatInlineImagePreviewProps = {
  file: File;
  index: number;
  disabled: boolean;
  onRemove: (index: number) => void;
};

function ChatInlineImagePreview({
  file,
  index,
  disabled,
  onRemove,
}: ChatInlineImagePreviewProps) {
  const previewUrl = useMemo(
    () => URL.createObjectURL(file),
    [file],
  );

  useEffect(() => {
    return () => {
      URL.revokeObjectURL(previewUrl);
    };
  }, [previewUrl]);

  return (
    <div className="chat-detail-page__desktop-preview-item">
      <img
        src={previewUrl}
        alt={file.name}
        className="chat-detail-page__desktop-preview-image"
      />

      <IconButton
        type="button"
        variant="secondary"
        size="sm"
        className="chat-detail-page__desktop-preview-remove"
        aria-label={`${file.name}を削除`}
        disabled={disabled}
        onClick={() => {
          onRemove(index);
        }}
      >
        <X size={16} aria-hidden="true" />
      </IconButton>
    </div>
  );
}

export default function ChatInlineComposer({
  content,
  placeholder,
  error,
  submitting,
  canSubmit,
  disabled = false,
  maxLength = 500,
  files = [],
  onContentChange,
  onFilesAdd,
  onRemoveFile,
  onSubmit,
}: ChatInlineComposerProps) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const inputDisabled = disabled || submitting;

  const supportsFiles =
    typeof onFilesAdd === "function" &&
    typeof onRemoveFile === "function";

  const handleKeyDown = (
    event: KeyboardEvent<HTMLTextAreaElement>,
  ): void => {
    if (
      event.key !== "Enter" ||
      event.shiftKey ||
      event.nativeEvent.isComposing
    ) {
      return;
    }

    event.preventDefault();

    if (inputDisabled || !canSubmit) {
      return;
    }

    void onSubmit();
  };

  return (
    <div className="chat-detail-page__desktop-composer">
      {supportsFiles && files.length > 0 ? (
        <div
          className="chat-detail-page__desktop-previews"
          aria-label="添付画像"
        >
          {files.map((file, index) => (
            <ChatInlineImagePreview
              key={`${file.name}-${file.size}-${file.lastModified}-${index}`}
              file={file}
              index={index}
              disabled={inputDisabled}
              onRemove={onRemoveFile}
            />
          ))}
        </div>
      ) : null}

      <div className="chat-detail-page__desktop-composer-row">
        {supportsFiles ? (
          <>
            <input
              ref={fileInputRef}
              type="file"
              accept="image/*"
              multiple
              disabled={inputDisabled}
              hidden
              onChange={(event) => {
                const selectedFiles = Array.from(
                  event.currentTarget.files ?? [],
                ).filter((file) =>
                  file.type.startsWith("image/"),
                );

                event.currentTarget.value = "";

                if (selectedFiles.length === 0) {
                  return;
                }

                onFilesAdd(selectedFiles);
              }}
            />

            <IconButton
              type="button"
              variant="ghost"
              size="md"
              className="chat-detail-page__desktop-attach-button"
              aria-label="画像を追加"
              disabled={inputDisabled}
              onClick={() => {
                fileInputRef.current?.click();
              }}
            >
              <ImagePlus size={22} aria-hidden="true" />
            </IconButton>
          </>
        ) : null}

        <Textbox
          value={content}
          rows={1}
          maxLength={maxLength ?? undefined}
          placeholder={placeholder}
          aria-label={placeholder}
          disabled={inputDisabled}
          className="chat-detail-page__desktop-composer-input"
          onChange={(event) => {
            onContentChange(event.currentTarget.value);
          }}
          onKeyDown={handleKeyDown}
        />
      </div>

      {error ? (
        <p
          className="chat-detail-page__desktop-composer-error"
          role="alert"
        >
          {error}
        </p>
      ) : null}
    </div>
  );
}