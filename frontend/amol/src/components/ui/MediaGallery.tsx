// frontend/amol/src/components/ui/MediaGallery.tsx
import {
  useRef,
  type TouchEvent,
} from "react";

import "./media-gallery.css";

export type MediaGalleryItem = {
  id: string;
  url: string;
  fileName?: string;
  type?: string;
};

type MediaGalleryProps = {
  items: MediaGalleryItem[];
  activeIndex: number;
  altFallback: string;
  placeholderText?: string;
  className?: string;
  onPrev: () => void;
  onNext: () => void;
  onSelect: (index: number) => void;
  onTouchStart?: (event: TouchEvent<HTMLDivElement>) => void;
  onTouchEnd?: (event: TouchEvent<HTMLDivElement>) => void;
};

const SWIPE_THRESHOLD = 48;

export default function MediaGallery({
  items,
  activeIndex,
  altFallback,
  placeholderText = "No Image",
  className,
  onPrev,
  onNext,
  onSelect,
  onTouchStart,
  onTouchEnd,
}: MediaGalleryProps) {
  const touchStartXRef = useRef<number | null>(null);

  const activeItem = items[activeIndex];
  const hasMultipleItems = items.length > 1;

  const handleTouchStart = (event: TouchEvent<HTMLDivElement>) => {
    if (onTouchStart) {
      onTouchStart(event);
      return;
    }

    touchStartXRef.current = event.changedTouches[0]?.clientX ?? null;
  };

  const handleTouchEnd = (event: TouchEvent<HTMLDivElement>) => {
    if (onTouchEnd) {
      onTouchEnd(event);
      return;
    }

    const startX = touchStartXRef.current;
    const endX = event.changedTouches[0]?.clientX ?? null;

    touchStartXRef.current = null;

    if (startX === null || endX === null) {
      return;
    }

    const diff = endX - startX;

    if (Math.abs(diff) < SWIPE_THRESHOLD) {
      return;
    }

    if (diff > 0) {
      onPrev();
      return;
    }

    onNext();
  };

  return (
    <div
      className={["media-gallery", className || ""]
        .filter(Boolean)
        .join(" ")}
    >
      {activeItem?.url ? (
        <div
          className="media-gallery__viewer"
          onTouchStart={handleTouchStart}
          onTouchEnd={handleTouchEnd}
        >
          <MediaGalleryPreview item={activeItem} altFallback={altFallback} />

          {hasMultipleItems ? (
            <>
              <button
                type="button"
                className="media-gallery__nav media-gallery__nav--prev"
                onClick={onPrev}
                aria-label="前のメディアを表示"
              >
                ‹
              </button>

              <button
                type="button"
                className="media-gallery__nav media-gallery__nav--next"
                onClick={onNext}
                aria-label="次のメディアを表示"
              >
                ›
              </button>

              <div className="media-gallery__counter">
                {activeIndex + 1} / {items.length}
              </div>
            </>
          ) : null}
        </div>
      ) : (
        <div className="media-gallery__placeholder">{placeholderText}</div>
      )}

      {hasMultipleItems ? (
        <div className="media-gallery__thumbnail-list">
          {items.map((item, index) => (
            <button
              key={item.id}
              type="button"
              className={[
                "media-gallery__thumbnail-button",
                index === activeIndex
                  ? "media-gallery__thumbnail-button--active"
                  : "",
              ]
                .filter(Boolean)
                .join(" ")}
              onClick={() => onSelect(index)}
              aria-label={`${index + 1}番目のメディアを表示`}
            >
              {item.type?.startsWith("video/") ? (
                <span className="media-gallery__thumbnail-video-label">
                  video
                </span>
              ) : (
                <img
                  src={item.url}
                  alt={item.fileName || altFallback}
                  className="media-gallery__thumbnail"
                  draggable={false}
                />
              )}
            </button>
          ))}
        </div>
      ) : null}
    </div>
  );
}

type MediaGalleryPreviewProps = {
  item: MediaGalleryItem;
  altFallback: string;
};

function MediaGalleryPreview({
  item,
  altFallback,
}: MediaGalleryPreviewProps) {
  if (item.type?.startsWith("video/")) {
    return (
      <video
        src={item.url}
        className="media-gallery__media"
        controls
        playsInline
        preload="metadata"
      />
    );
  }

  return (
    <img
      src={item.url}
      alt={item.fileName || altFallback}
      className="media-gallery__media"
      draggable={false}
    />
  );
}