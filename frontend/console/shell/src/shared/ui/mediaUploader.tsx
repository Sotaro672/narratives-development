// frontend/console/shell/src/shared/ui/mediaUploader.tsx

import * as React from "react";
import { Upload } from "lucide-react";

import { Button } from "./button";
import DeleteButton, {
  type DeleteButtonSize,
} from "./delete";
import Media, {
  type MediaFit,
  type MediaType,
  type MediaVariant,
} from "./media";
import Text from "./text";

import "./mediaUploader.css";

function cn(...classes: Array<string | undefined | false | null>): string {
  return classes.filter(Boolean).join(" ");
}

export type MediaUploaderVariant =
  | "single"
  | "grid";

export type MediaUploaderPickerVariant =
  | "dropzone"
  | "button";

export type MediaUploaderItem = {
  id: string;
  src?: string | null;
  name?: string;
  alt?: string;
  type?: MediaType;
  contentType?: string;
};

export type MediaUploaderRenderContext = {
  disabled: boolean;
  openPicker: () => void;
};

export type MediaUploaderProps = {
  items?: MediaUploaderItem[];
  accept?: string;
  multiple?: boolean;
  maxFiles?: number;

  variant?: MediaUploaderVariant;
  pickerVariant?: MediaUploaderPickerVariant;
  mediaVariant?: MediaVariant;
  mediaFit?: MediaFit;

  title?: React.ReactNode;
  pickerLabel?: React.ReactNode;
  replaceLabel?: React.ReactNode;
  pickerDescription?: React.ReactNode;
  emptyIcon?: React.ReactNode;

  disabled?: boolean;
  showCount?: boolean;
  showPicker?: boolean;
  showFileNames?: boolean;
  showRemoveButton?: boolean;
  previewFramed?: boolean;
  removeButtonSize?: DeleteButtonSize;

  className?: string;
  pickerClassName?: string;
  previewClassName?: string;
  mediaClassName?: string;

  renderPreview?: (
    item: MediaUploaderItem,
    context: MediaUploaderRenderContext,
  ) => React.ReactNode;
  renderEmpty?: (
    context: MediaUploaderRenderContext,
  ) => React.ReactNode;

  onFilesSelected: (files: File[]) => void;
  onRemove?: (id: string) => void;
};

export default function MediaUploader({
  items = [],
  accept,
  multiple = false,
  maxFiles,
  variant = "grid",
  pickerVariant = "dropzone",
  mediaVariant = "square",
  mediaFit = "cover",
  title = "添付ファイル",
  pickerLabel = "ファイルを選択",
  replaceLabel = "ファイルを変更",
  pickerDescription,
  emptyIcon,
  disabled = false,
  showCount = true,
  showPicker = true,
  showFileNames,
  showRemoveButton = true,
  previewFramed = true,
  removeButtonSize,
  className,
  pickerClassName,
  previewClassName,
  mediaClassName,
  renderPreview,
  renderEmpty,
  onFilesSelected,
  onRemove,
}: MediaUploaderProps) {
  const inputRef = React.useRef<HTMLInputElement | null>(null);
  const [dragging, setDragging] = React.useState(false);

  const displayedItems =
    variant === "single"
      ? items.slice(0, 1)
      : items;

  const effectiveMaxFiles =
    variant === "single"
      ? 1
      : maxFiles;

  const hasItems = displayedItems.length > 0;
  const shouldShowFileNames =
    showFileNames ?? variant === "grid";

  const canSelect =
    !disabled &&
    (variant === "single" ||
      effectiveMaxFiles == null ||
      displayedItems.length < effectiveMaxFiles);

  const openPicker = React.useCallback(() => {
    if (!canSelect) {
      return;
    }

    inputRef.current?.click();
  }, [canSelect]);

  const normalizeFiles = React.useCallback(
    (files: File[]): File[] => {
      if (variant === "single") {
        return files.slice(0, 1);
      }

      return files;
    },
    [variant],
  );

  const submitFiles = React.useCallback(
    (files: File[]) => {
      if (!canSelect || files.length === 0) {
        return;
      }

      const normalizedFiles = normalizeFiles(files);

      if (normalizedFiles.length === 0) {
        return;
      }

      onFilesSelected(normalizedFiles);
    },
    [
      canSelect,
      normalizeFiles,
      onFilesSelected,
    ],
  );

  const handleInputChange = (
    event: React.ChangeEvent<HTMLInputElement>,
  ) => {
    const files = Array.from(event.currentTarget.files ?? []);
    event.currentTarget.value = "";
    submitFiles(files);
  };

  const handleDragEnter = (
    event: React.DragEvent<HTMLDivElement>,
  ) => {
    event.preventDefault();
    event.stopPropagation();

    if (!canSelect) {
      return;
    }

    setDragging(true);
  };

  const handleDragOver = (
    event: React.DragEvent<HTMLDivElement>,
  ) => {
    event.preventDefault();
    event.stopPropagation();

    if (!canSelect) {
      return;
    }

    event.dataTransfer.dropEffect = "copy";
    setDragging(true);
  };

  const handleDragLeave = (
    event: React.DragEvent<HTMLDivElement>,
  ) => {
    event.preventDefault();
    event.stopPropagation();

    if (event.currentTarget.contains(event.relatedTarget as Node | null)) {
      return;
    }

    setDragging(false);
  };

  const handleDrop = (
    event: React.DragEvent<HTMLDivElement>,
  ) => {
    event.preventDefault();
    event.stopPropagation();
    setDragging(false);

    if (!canSelect) {
      return;
    }

    submitFiles(Array.from(event.dataTransfer.files ?? []));
  };

  const renderContext: MediaUploaderRenderContext = {
    disabled,
    openPicker,
  };

  const renderItem = (item: MediaUploaderItem) => (
    <div
      key={item.id}
      className={cn(
        "ui-media-uploader__item",
        previewFramed && "ui-media-uploader__item--framed",
      )}
    >
      <div className="ui-media-uploader__preview">
        {renderPreview ? (
          renderPreview(item, renderContext)
        ) : (
          <Media
            src={item.src}
            type={item.type ?? "image"}
            alt={item.alt ?? item.name}
            name={item.name}
            contentType={item.contentType}
            variant={mediaVariant}
            fit={mediaFit}
            bordered={false}
            className={cn(
              "ui-media-uploader__media",
              mediaClassName,
            )}
          />
        )}

        {showRemoveButton && onRemove ? (
          <DeleteButton
            size={
              removeButtonSize ??
              (variant === "single" ? "md" : "sm")
            }
            className="ui-media-uploader__delete"
            disabled={disabled}
            ariaLabel={
              item.name
                ? `${item.name}を削除`
                : "メディアを削除"
            }
            onClick={(event) => {
              event.stopPropagation();
              onRemove(item.id);
            }}
          />
        ) : null}
      </div>

      {shouldShowFileNames && item.name ? (
        <Text
          as="div"
          size="xs"
          wrap="nowrap"
          className="ui-media-uploader__name"
          title={item.name}
        >
          {item.name}
        </Text>
      ) : null}
    </div>
  );

  const pickerText =
    variant === "single" && hasItems
      ? replaceLabel
      : pickerLabel;

  return (
    <div
      className={cn(
        "ui-media-uploader",
        `ui-media-uploader--${variant}`,
        disabled && "ui-media-uploader--disabled",
        className,
      )}
    >
      {(title || showCount) ? (
        <div className="ui-media-uploader__header">
          {title ? (
            <Text weight="semibold">
              {title}
            </Text>
          ) : (
            <span />
          )}

          {showCount ? (
            <Text
              size="xs"
              tone="muted"
              wrap="nowrap"
            >
              {effectiveMaxFiles != null
                ? `${displayedItems.length} / ${effectiveMaxFiles}`
                : `${displayedItems.length}`}
            </Text>
          ) : null}
        </div>
      ) : null}

      <input
        ref={inputRef}
        type="file"
        accept={accept}
        multiple={variant === "grid" && multiple}
        className="ui-media-uploader__input"
        disabled={!canSelect}
        onChange={handleInputChange}
      />

      {hasItems ? (
        <div
          className={cn(
            variant === "single"
              ? "ui-media-uploader__single"
              : "ui-media-uploader__grid",
            previewClassName,
          )}
        >
          {displayedItems.map(renderItem)}
        </div>
      ) : renderEmpty ? (
        <div className="ui-media-uploader__custom-empty">
          {renderEmpty(renderContext)}
        </div>
      ) : null}

      {showPicker && canSelect ? (
        pickerVariant === "button" ? (
          <Button
            type="button"
            variant="outline"
            className={cn(
              "ui-media-uploader__picker-button",
              pickerClassName,
            )}
            onClick={openPicker}
            disabled={disabled}
          >
            <Upload size={16} aria-hidden />
            {pickerText}
          </Button>
        ) : (
          <div
            className={cn(
              "ui-media-uploader__dropzone",
              dragging && "ui-media-uploader__dropzone--dragging",
              pickerClassName,
            )}
            role="button"
            tabIndex={disabled ? undefined : 0}
            aria-disabled={disabled}
            onClick={openPicker}
            onKeyDown={(event) => {
              if (
                event.key !== "Enter" &&
                event.key !== " "
              ) {
                return;
              }

              event.preventDefault();
              openPicker();
            }}
            onDragEnter={handleDragEnter}
            onDragOver={handleDragOver}
            onDragLeave={handleDragLeave}
            onDrop={handleDrop}
          >
            <div className="ui-media-uploader__picker-icon">
              {emptyIcon ?? <Upload size={20} aria-hidden />}
            </div>

            <Text weight="semibold">
              {pickerText}
            </Text>

            {pickerDescription ? (
              <Text
                size="xs"
                tone="muted"
                className="ui-media-uploader__picker-description"
              >
                {pickerDescription}
              </Text>
            ) : null}
          </div>
        )
      ) : null}
    </div>
  );
}