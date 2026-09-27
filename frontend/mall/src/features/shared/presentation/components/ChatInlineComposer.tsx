// frontend/mall/src/features/shared/presentation/components/ChatInlineComposer.tsx

import {
  type ChangeEvent,
  type KeyboardEvent,
  useRef,
} from "react";

import Button from "../../../../components/ui/Button";
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
  onFilesChange?: (event: ChangeEvent<HTMLInputElement>) => void;
  onRemoveFile?: (index: number) => void;
  onSubmit: () => void | Promise<void>;
};

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
  onFilesChange,
  onRemoveFile,
  onSubmit,
}: ChatInlineComposerProps) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const inputDisabled = disabled || submitting;
  const supportsFiles =
    typeof onFilesChange === "function" &&
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

    if (
      inputDisabled ||
      !canSubmit
    ) {
      return;
    }

    void onSubmit();
  };

  return (
    <div className="chat-detail-page__desktop-composer">
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
              onChange={onFilesChange}
            />

            <Button
              type="button"
              variant="ghost"
              size="sm"
              disabled={inputDisabled}
              onClick={() => {
                fileInputRef.current?.click();
              }}
            >
              画像
            </Button>
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

      {supportsFiles && files.length > 0 ? (
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
                disabled={inputDisabled}
                onClick={() => {
                  onRemoveFile(index);
                }}
              >
                削除
              </Button>
            </div>
          ))}
        </div>
      ) : null}

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