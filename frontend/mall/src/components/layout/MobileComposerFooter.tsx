// frontend/mall/src/components/layout/MobileComposerFooter.tsx

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
  type CompositionEvent,
  type ReactNode,
} from "react";
import { createPortal } from "react-dom";
import { ImagePlus, X } from "lucide-react";

import type { ChatComposerConfig } from "../../features/shared/types/chatComposer";
import IconButton from "../ui/IconButton";

import "./mobileComposerFooter.css";

export type MobileComposerFooterProps = ChatComposerConfig & {
  beforeInput?: ReactNode;
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
  beforeInput,
  onContentChange,
  onFilesAdd,
  onRemoveFile,
  onSubmit,
}: MobileComposerFooterProps) {
  const footerRef = useRef<HTMLElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const isComposingRef = useRef(false);
  const lastEmittedContentRef = useRef(content);
  const [draft, setDraft] = useState(content);
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

  const emitContentChange = useCallback(
    (value: string): void => {
      if (lastEmittedContentRef.current === value) {
        return;
      }

      lastEmittedContentRef.current = value;
      onContentChange(value);
    },
    [onContentChange],
  );

  useEffect(() => {
    if (isComposingRef.current) {
      return;
    }

    lastEmittedContentRef.current = content;

    setDraft((currentDraft) =>
      currentDraft === content
        ? currentDraft
        : content,
    );
  }, [content]);

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
  }, [draft]);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    const footer = footerRef.current;

    if (!footer) {
      return;
    }

    const root = document.documentElement;
    const visualViewport = window.visualViewport;
    let animationFrameId: number | null = null;

    const updateKeyboardOffset = (): void => {
      if (!visualViewport) {
        footer.style.setProperty(
          "--mobile-composer-keyboard-offset",
          "0px",
        );
        return;
      }

      const layoutViewportHeight = window.innerHeight;

      const visualViewportBottom =
        visualViewport.offsetTop +
        visualViewport.height;

      const keyboardOffset = Math.max(
        0,
        layoutViewportHeight -
          visualViewportBottom,
      );

      footer.style.setProperty(
        "--mobile-composer-keyboard-offset",
        `${Math.round(keyboardOffset)}px`,
      );
    };

    const updateComposerHeight = (): void => {
      const height =
        footer.getBoundingClientRect().height;

      root.style.setProperty(
        "--mobile-composer-height",
        `${Math.ceil(height)}px`,
      );
    };

    const updateLayout = (): void => {
      if (animationFrameId !== null) {
        window.cancelAnimationFrame(
          animationFrameId,
        );
      }

      animationFrameId =
        window.requestAnimationFrame(() => {
          animationFrameId = null;
          updateKeyboardOffset();
          updateComposerHeight();
        });
    };

    const resizeObserver =
      typeof ResizeObserver !== "undefined"
        ? new ResizeObserver(updateLayout)
        : null;

    resizeObserver?.observe(footer);

    window.addEventListener(
      "resize",
      updateLayout,
      { passive: true },
    );

    window.addEventListener(
      "orientationchange",
      updateLayout,
      { passive: true },
    );

    visualViewport?.addEventListener(
      "resize",
      updateLayout,
      { passive: true },
    );

    visualViewport?.addEventListener(
      "scroll",
      updateLayout,
      { passive: true },
    );

    document.addEventListener(
      "focusin",
      updateLayout,
    );

    document.addEventListener(
      "focusout",
      updateLayout,
    );

    updateLayout();

    return () => {
      if (animationFrameId !== null) {
        window.cancelAnimationFrame(
          animationFrameId,
        );
      }

      resizeObserver?.disconnect();

      window.removeEventListener(
        "resize",
        updateLayout,
      );

      window.removeEventListener(
        "orientationchange",
        updateLayout,
      );

      visualViewport?.removeEventListener(
        "resize",
        updateLayout,
      );

      visualViewport?.removeEventListener(
        "scroll",
        updateLayout,
      );

      document.removeEventListener(
        "focusin",
        updateLayout,
      );

      document.removeEventListener(
        "focusout",
        updateLayout,
      );

      footer.style.removeProperty(
        "--mobile-composer-keyboard-offset",
      );

      root.style.removeProperty(
        "--mobile-composer-height",
      );
    };
  }, []);

  const handleCompositionStart = (
    _event: CompositionEvent<HTMLTextAreaElement>,
  ): void => {
    isComposingRef.current = true;
  };

  const handleCompositionEnd = (
    event: CompositionEvent<HTMLTextAreaElement>,
  ): void => {
    const nextValue =
      event.currentTarget.value;

    isComposingRef.current = false;
    setDraft(nextValue);
    emitContentChange(nextValue);
  };

  const handleContentChange = (
    event: ChangeEvent<HTMLTextAreaElement>,
  ): void => {
    const nextValue =
      event.currentTarget.value;

    setDraft(nextValue);

    if (isComposingRef.current) {
      return;
    }

    emitContentChange(nextValue);
  };

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
      typeof onFilesAdd !== "function" ||
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
    if (
      submitDisabled ||
      isComposingRef.current
    ) {
      return;
    }

    void onSubmit();
  };

  if (typeof document === "undefined") {
    return null;
  }

  return createPortal(
    <footer
      ref={footerRef}
      className="mobile-composer-footer"
      aria-busy={submitting || undefined}
      data-mobile-swipe-dismiss-ignore="true"
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

      {beforeInput ? (
        <div className="mobile-composer-footer__before-input">
          {beforeInput}
        </div>
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
          value={draft}
          rows={1}
          maxLength={maxLength ?? undefined}
          placeholder={placeholder}
          aria-label={placeholder}
          disabled={inputDisabled}
          enterKeyHint="enter"
          onCompositionStart={handleCompositionStart}
          onCompositionEnd={handleCompositionEnd}
          onChange={handleContentChange}
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
    </footer>,
    document.body,
  );
}