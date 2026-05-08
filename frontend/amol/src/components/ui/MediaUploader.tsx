// frontend/src/components/ui/MediaUploader.tsx
import { ChangeEvent, RefObject } from "react";
import Button from "./Button";
import "./media-uploader.css";

export type MediaUploaderItem = {
  id: string;
  type: "image" | "pdf" | "youtube";
  previewUrl?: string;
  youtubeUrl?: string;
  title?: string;
  fileName?: string;
};

type MediaUploaderProps = {
  label: string;
  hint?: string;
  emptyText?: string;
  selectButtonLabel?: string;
  selectingButtonLabel?: string;
  accept?: string;
  multiple?: boolean;
  items: MediaUploaderItem[];
  currentIndex: number;
  disabled?: boolean;
  selecting?: boolean;
  inputRef: RefObject<HTMLInputElement>;
  carouselRef?: RefObject<HTMLDivElement>;
  onFilesSelected: (event: ChangeEvent<HTMLInputElement>) => void;
  onRemoveItem?: (id: string) => void;
  onCarouselScroll?: () => void;
  onMoveToSlide?: (index: number) => void;
  renderEmbedUrl?: (item: MediaUploaderItem) => string;
};

export default function MediaUploader({
  label,
  hint,
  emptyText = "ファイルが登録されていません。",
  selectButtonLabel = "ファイルを選択",
  selectingButtonLabel = "アップロード中...",
  accept = "image/*",
  multiple = true,
  items,
  currentIndex,
  disabled = false,
  selecting = false,
  inputRef,
  carouselRef,
  onFilesSelected,
  onRemoveItem,
  onCarouselScroll,
  onMoveToSlide,
  renderEmbedUrl,
}: MediaUploaderProps) {
  return (
    <div className="media-uploader">
      <input
        ref={inputRef}
        type="file"
        accept={accept}
        multiple={multiple}
        onChange={onFilesSelected}
        disabled={disabled}
        className="media-uploader__input"
      />

      <div className="media-uploader__panel">
        <div className="media-uploader__header">
          <div className="media-uploader__header-text">
            <strong>{label}</strong>
            {hint ? <p className="media-uploader__hint">{hint}</p> : null}
          </div>

          <Button
            type="button"
            size="sm"
            variant="secondary"
            disabled={disabled}
            onClick={() => inputRef.current?.click()}
          >
            {selecting ? selectingButtonLabel : selectButtonLabel}
          </Button>
        </div>

        {items.length === 0 ? (
          <div className="media-uploader__empty">{emptyText}</div>
        ) : (
          <>
            <div
              ref={carouselRef}
              onScroll={onCarouselScroll}
              className="media-uploader__carousel"
            >
              {items.map((item) => (
                <div key={item.id} className="media-uploader__carousel-card">
                  {onRemoveItem ? (
                    <button
                      type="button"
                      aria-label="メディアを削除"
                      onClick={() => onRemoveItem(item.id)}
                      disabled={disabled}
                      className="media-uploader__remove"
                    >
                      ×
                    </button>
                  ) : null}

                  {item.type === "image" && item.previewUrl ? (
                    <div className="media-uploader__image-frame">
                      <img
                        src={item.previewUrl}
                        alt={item.title || item.fileName || "image preview"}
                        className="media-uploader__image"
                      />
                    </div>
                  ) : null}

                  {item.type === "pdf" && item.previewUrl ? (
                    <iframe
                      src={item.previewUrl}
                      title={item.title || item.fileName || "pdf preview"}
                      className="media-uploader__frame"
                    />
                  ) : null}

                  {item.type === "youtube" &&
                  item.youtubeUrl &&
                  renderEmbedUrl ? (
                    <iframe
                      src={renderEmbedUrl(item)}
                      title={item.title || "youtube preview"}
                      allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
                      referrerPolicy="strict-origin-when-cross-origin"
                      allowFullScreen
                      className="media-uploader__frame"
                    />
                  ) : null}
                </div>
              ))}
            </div>

            {onMoveToSlide ? (
              <div className="media-uploader__dots">
                {items.map((item, index) => (
                  <button
                    key={item.id}
                    type="button"
                    aria-label={`${index + 1}番目のメディアを表示`}
                    onClick={() => onMoveToSlide(index)}
                    className={
                      index === currentIndex
                        ? "media-uploader__dot media-uploader__dot--active"
                        : "media-uploader__dot"
                    }
                  />
                ))}
              </div>
            ) : null}
          </>
        )}
      </div>
    </div>
  );
}