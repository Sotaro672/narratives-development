// frontend/admin/shell/src/shared/ui/MediaGallery/MediaGallery.tsx
import { useEffect, useRef, useState, type TouchEvent } from "react";

import "./MediaGallery.css";

export type MediaGalleryItem = {
  id: string;
  url: string;
  fileName: string;
};

type MediaGalleryProps = {
  items: MediaGalleryItem[];
  altFallback?: string;
  placeholderText?: string;
  onDownload?: (item: MediaGalleryItem) => void | Promise<void>;
};

const SWIPE_THRESHOLD = 48;

export default function MediaGallery({
  items,
  altFallback = "添付画像",
  placeholderText = "添付ファイルはありません。",
  onDownload,
}: MediaGalleryProps) {
  const [activeIndex, setActiveIndex] = useState(0);
  const [downloading, setDownloading] = useState(false);
  const touchStartXRef = useRef<number | null>(null);

  const activeItem = items[activeIndex];
  const hasMultipleItems = items.length > 1;

  useEffect(() => {
    setActiveIndex((current) => {
      if (items.length === 0) {
        return 0;
      }

      return Math.min(current, items.length - 1);
    });
  }, [items.length]);

  const handlePrev = () => {
    if (items.length <= 1) {
      return;
    }

    setActiveIndex((current) =>
      current === 0 ? items.length - 1 : current - 1,
    );
  };

  const handleNext = () => {
    if (items.length <= 1) {
      return;
    }

    setActiveIndex((current) =>
      current === items.length - 1 ? 0 : current + 1,
    );
  };

  const handleSelect = (index: number) => {
    if (index < 0 || index >= items.length) {
      return;
    }

    setActiveIndex(index);
  };

  const handleTouchStart = (event: TouchEvent<HTMLDivElement>) => {
    touchStartXRef.current = event.changedTouches[0]?.clientX ?? null;
  };

  const handleTouchEnd = (event: TouchEvent<HTMLDivElement>) => {
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
      handlePrev();
      return;
    }

    handleNext();
  };

  const handleDownload = async () => {
    if (!activeItem || !onDownload || downloading) {
      return;
    }

    setDownloading(true);

    try {
      await onDownload(activeItem);
    } finally {
      setDownloading(false);
    }
  };

  if (!activeItem) {
    return (
      <div className="media-gallery">
        <div className="media-gallery__placeholder">{placeholderText}</div>
      </div>
    );
  }

  return (
    <div className="media-gallery">
      <div
        className="media-gallery__viewer"
        onTouchStart={handleTouchStart}
        onTouchEnd={handleTouchEnd}
      >
        <img
          src={activeItem.url}
          alt={activeItem.fileName || altFallback}
          className="media-gallery__media"
          draggable={false}
        />

        {hasMultipleItems ? (
          <>
            <button
              type="button"
              className="media-gallery__nav media-gallery__nav--prev"
              onClick={handlePrev}
              aria-label="前の添付ファイルを表示"
            >
              ‹
            </button>

            <button
              type="button"
              className="media-gallery__nav media-gallery__nav--next"
              onClick={handleNext}
              aria-label="次の添付ファイルを表示"
            >
              ›
            </button>

            <div className="media-gallery__counter">
              {activeIndex + 1} / {items.length}
            </div>
          </>
        ) : null}
      </div>

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
              onClick={() => handleSelect(index)}
              aria-label={`${index + 1}番目の添付ファイルを表示`}
              aria-current={index === activeIndex ? "true" : undefined}
            >
              <img
                src={item.url}
                alt={item.fileName || `${altFallback} ${index + 1}`}
                className="media-gallery__thumbnail"
                draggable={false}
              />
            </button>
          ))}
        </div>
      ) : null}

      <div className="media-gallery__footer">
        <span className="media-gallery__file-name">{activeItem.fileName}</span>

        {onDownload ? (
          <button
            type="button"
            className="media-gallery__download"
            onClick={() => void handleDownload()}
            disabled={downloading}
          >
            {downloading ? "ダウンロード中..." : "ダウンロード"}
          </button>
        ) : null}
      </div>
    </div>
  );
}