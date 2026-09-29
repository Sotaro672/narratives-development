// frontend/mall/src/components/layout/MobileComposerFooter.tsx

import {
  useEffect,
  useMemo,
  useRef,
  type ChangeEvent,
} from "react";
import {
  ImagePlus,
  X,
} from "lucide-react";

import IconButton from "../ui/IconButton";
import "./mobile-composer-footer.css";
import "./footer.css";

export type MobileComposerFooterProps = {
  content: string;
  placeholder?: string;
  files?: File[];
  error?: string | null;
  submitting?: boolean;
  canSubmit: boolean;
  disabled?: boolean;
  submitLabel?: string;
  submittingLabel?: string;
  maxLength?: number | null;
  maxFiles?: number;
  accept?: string;
  onContentChange: (value: string) => void;
  onFilesAdd?: (files: File[]) => void;
  onRemoveFile?: (index: number) => void;
  onSubmit: () => void | Promise<void>;
};

type MobileComposerImagePreviewProps = {
  file: File;
  index: number;
  disabled: boolean;
  onRemove: (index: number) => void;
};

function MobileComposerImagePreview({
  file,
  index,
  disabled,
  onRemove,
}: MobileComposerImagePreviewProps) {
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
    <div className="mobile-composer-footer__preview-item">
      <img
        src={previewUrl}
        alt=""
        className="mobile-composer-footer__preview-image"
      />

      <IconButton
        type="button"
        variant="secondary"
        size="sm"
        className="mobile-composer-footer__preview-remove"
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

export default function MobileComposerFooter({
  content,
  placeholder = "メッセージを入力",
  files = [],
  error,
  submitting = false,
  canSubmit,
  disabled = false,
  submitLabel = "送信",
  submittingLabel = "送信中",
  maxLength = 500,
  maxFiles = 10,
  accept = "image/*",
  onContentChange,
  onFilesAdd,
  onRemoveFile,
  onSubmit,
}: MobileComposerFooterProps) {
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const inputDisabled = disabled || submitting;
  const supportsFiles =
    typeof onFilesAdd === "function" &&
    typeof onRemoveFile === "function";
  const remainingFileCount = Math.max(
    maxFiles - files.length,
    0,
  );
  const canAddFiles =
    supportsFiles &&
    !inputDisabled &&
    remainingFileCount > 0;
  const submitDisabled =
    inputDisabled ||
    !canSubmit;

  useEffect(() => {
    const textarea = textareaRef.current;

    if (!textarea) {
      return;
    }

    textarea.style.height = "auto";

    const nextHeight = Math.min(
      Math.max(textarea.scrollHeight, 40),
      96,
    );

    textarea.style.height = `${nextHeight}px`;
    textarea.style.overflowY =
      textarea.scrollHeight > 96
        ? "auto"
        : "hidden";
  }, [content]);

  const handleFilesChange = (
    event: ChangeEvent<HTMLInputElement>,
  ): void => {
    const selectedFiles = Array.from(
      event.currentTarget.files ?? [],
    ).filter((file) =>
      file.type.startsWith("image/"),
    );

    event.currentTarget.value = "";

    if (
      !supportsFiles ||
      inputDisabled ||
      selectedFiles.length === 0 ||
      remainingFileCount <= 0
    ) {
      return;
    }

    onFilesAdd(
      selectedFiles.slice(
        0,
        remainingFileCount,
      ),
    );
  };

  const handleSubmit = (): void => {
    if (submitDisabled) {
      return;
    }

    void onSubmit();
  };

  return (
    <footer
      className="mobile-composer-footer"
      aria-busy={submitting || undefined}
    >
      {supportsFiles && files.length > 0 ? (
        <div
          className="mobile-composer-footer__previews"
          aria-label="添付画像"
        >
          {files.map((file, index) => (
            <MobileComposerImagePreview
              key={`${file.name}-${file.size}-${file.lastModified}-${index}`}
              file={file}
              index={index}
              disabled={inputDisabled}
              onRemove={onRemoveFile}
            />
          ))}
        </div>
      ) : null}

      {error ? (
        <p
          className="mobile-composer-footer__error"
          role="alert"
        >
          {error}
        </p>
      ) : null}

      <div className="mobile-composer-footer__row">
        {supportsFiles ? (
          <>
            <input
              ref={fileInputRef}
              type="file"
              accept={accept}
              multiple
              hidden
              disabled={!canAddFiles}
              onChange={handleFilesChange}
            />

            <IconButton
              type="button"
              variant="ghost"
              size="md"
              className="mobile-composer-footer__attach-button"
              aria-label={
                remainingFileCount > 0
                  ? "画像を追加"
                  : `画像は最大${maxFiles}枚までです`
              }
              disabled={!canAddFiles}
              onClick={() => {
                fileInputRef.current?.click();
              }}
            >
              <ImagePlus
                size={22}
                aria-hidden="true"
              />
            </IconButton>
          </>
        ) : null}

        <textarea
          ref={textareaRef}
          className="mobile-composer-footer__input"
          value={content}
          rows={1}
          maxLength={maxLength ?? undefined}
          placeholder={placeholder}
          aria-label={placeholder}
          disabled={inputDisabled}
          enterKeyHint="enter"
          onChange={(event) => {
            onContentChange(
              event.currentTarget.value,
            );
          }}
        />

        <button
          type="button"
          className="mobile-composer-footer__submit"
          disabled={submitDisabled}
          aria-label={submitLabel}
          onClick={handleSubmit}
        >
          {submitting
            ? submittingLabel
            : submitLabel}
        </button>
      </div>
    </footer>
  );
}