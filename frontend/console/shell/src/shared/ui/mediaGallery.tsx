// frontend/console/shell/src/shared/ui/mediaGallery.tsx

import * as React from "react";
import {
  ChevronLeft,
  ChevronRight,
  FileText,
} from "lucide-react";

import { Button } from "./button";
import DeleteButton from "./delete";
import Media, {
  type MediaFit,
  type MediaProps,
  type MediaType,
  type MediaVariant,
} from "./media";

import "./mediaGallery.css";

function cn(...classes: Array<string | undefined | false | null>): string {
  return classes.filter(Boolean).join(" ");
}

function clampIndex(index: number, itemCount: number): number {
  if (itemCount <= 0) {
    return 0;
  }

  if (!Number.isFinite(index)) {
    return 0;
  }

  return Math.min(
    Math.max(Math.trunc(index), 0),
    itemCount - 1,
  );
}

export type MediaGalleryItem = {
  id: string;
  src?: string | null;
  name?: string;
  alt?: string;
  type?: MediaType;
  contentType?: string;
};

export type MediaGalleryProps = {
  items?: MediaGalleryItem[];

  activeIndex?: number;
  defaultActiveIndex?: number;
  onActiveIndexChange?: (index: number) => void;

  editable?: boolean;
  deleteDisabled?: boolean;

  mainVariant?: MediaVariant;
  mainFit?: MediaFit;
  thumbnailFit?: MediaFit;

  showViewer?: boolean;
  showNavigation?: boolean;
  showThumbnails?: boolean;

  emptyIcon?: React.ReactNode;
  emptyText?: React.ReactNode;
  emptyDescription?: React.ReactNode;

  className?: string;

  imageProps?: MediaProps["imageProps"];
  videoProps?: MediaProps["videoProps"];

  onDelete?: (
    item: MediaGalleryItem,
    index: number,
  ) => void | Promise<void>;
};

export default function MediaGallery({
  items = [],
  activeIndex,
  defaultActiveIndex = 0,
  onActiveIndexChange,
  editable = false,
  deleteDisabled = false,
  mainVariant = "viewer",
  mainFit = "contain",
  thumbnailFit = "cover",
  showViewer = true,
  showNavigation = true,
  showThumbnails = true,
  emptyIcon = <FileText />,
  emptyText = "メディアがまだ登録されていません",
  emptyDescription,
  className,
  imageProps,
  videoProps,
  onDelete,
}: MediaGalleryProps) {
  const [internalIndex, setInternalIndex] = React.useState(
    defaultActiveIndex,
  );

  const isControlled = activeIndex !== undefined;
  const requestedIndex = isControlled
    ? activeIndex
    : internalIndex;

  const hasItems = items.length > 0;
  const hasMultipleItems = items.length > 1;
  const safeIndex = clampIndex(
    requestedIndex,
    items.length,
  );

  const currentItem = hasItems
    ? items[safeIndex]
    : undefined;

  const shouldShowThumbnails =
    showThumbnails &&
    (hasMultipleItems || !showViewer);

  React.useEffect(() => {
    if (isControlled) {
      return;
    }

    setInternalIndex((currentIndex) =>
      clampIndex(currentIndex, items.length),
    );
  }, [
    isControlled,
    items.length,
  ]);

  const setActiveIndex = React.useCallback(
    (nextIndex: number) => {
      if (!hasItems) {
        return;
      }

      const nextSafeIndex = clampIndex(
        nextIndex,
        items.length,
      );

      if (!isControlled) {
        setInternalIndex(nextSafeIndex);
      }

      onActiveIndexChange?.(nextSafeIndex);
    },
    [
      hasItems,
      isControlled,
      items.length,
      onActiveIndexChange,
    ],
  );

  const prev = React.useCallback(() => {
    if (!hasMultipleItems) {
      return;
    }

    setActiveIndex(
      (safeIndex - 1 + items.length) %
        items.length,
    );
  }, [
    hasMultipleItems,
    items.length,
    safeIndex,
    setActiveIndex,
  ]);

  const next = React.useCallback(() => {
    if (!hasMultipleItems) {
      return;
    }

    setActiveIndex(
      (safeIndex + 1) % items.length,
    );
  }, [
    hasMultipleItems,
    items.length,
    safeIndex,
    setActiveIndex,
  ]);

  const handleDelete = React.useCallback(
    (
      item: MediaGalleryItem,
      itemIndex: number,
    ) => {
      if (
        !editable ||
        deleteDisabled ||
        !onDelete
      ) {
        return;
      }

      void onDelete(item, itemIndex);
    },
    [
      editable,
      deleteDisabled,
      onDelete,
    ],
  );

  const renderThumbnail = (
    item: MediaGalleryItem,
    itemIndex: number,
  ) => {
    if ((item.type ?? "image") === "image") {
      return (
        <Media
          src={item.src}
          type="image"
          name={item.name}
          alt={
            item.alt ??
            `メディア サムネイル ${itemIndex + 1}`
          }
          variant="square"
          fit={thumbnailFit}
          bordered={false}
          onActivate={() => {
            setActiveIndex(itemIndex);
          }}
        />
      );
    }

    return (
      <Media
        variant="square"
        bordered={false}
        emptyText={(item.type ?? "document").toUpperCase()}
        emptyDescription={item.name}
        onActivate={() => {
          setActiveIndex(itemIndex);
        }}
      />
    );
  };

  if (!currentItem) {
    return (
      <div
        className={cn(
          "ui-media-gallery",
          "ui-media-gallery--empty",
          className,
        )}
      >
        {showViewer ? (
          <Media
            variant={mainVariant}
            fit={mainFit}
            emptyIcon={emptyIcon}
            emptyText={emptyText}
            emptyDescription={emptyDescription}
          />
        ) : null}
      </div>
    );
  }

  return (
    <div
      className={cn(
        "ui-media-gallery",
        !showViewer && "ui-media-gallery--thumbnails-only",
        className,
      )}
    >
      {showViewer ? (
        <div
          className={cn(
            "ui-media-gallery__viewer",
            (!hasMultipleItems || !showNavigation) &&
              "ui-media-gallery__viewer--single",
          )}
        >
          {hasMultipleItems && showNavigation ? (
            <Button
              type="button"
              variant="outline"
              size="icon"
              className="ui-media-gallery__nav"
              onClick={prev}
              aria-label="前のメディア"
            >
              <ChevronLeft className="ui-media-gallery__nav-icon" />
            </Button>
          ) : null}

          <div className="ui-media-gallery__main">
            <Media
              src={currentItem.src}
              type={currentItem.type ?? "image"}
              name={currentItem.name}
              alt={
                currentItem.alt ??
                currentItem.name ??
                `メディア ${safeIndex + 1}`
              }
              contentType={currentItem.contentType}
              variant={mainVariant}
              fit={mainFit}
              bordered={false}
              imageProps={imageProps}
              videoProps={videoProps}
            />

            {editable && onDelete ? (
              <DeleteButton
                size="lg"
                className="ui-media-gallery__delete"
                disabled={deleteDisabled}
                ariaLabel="このメディアを削除"
                onClick={(event) => {
                  event.stopPropagation();
                  handleDelete(
                    currentItem,
                    safeIndex,
                  );
                }}
              />
            ) : null}
          </div>

          {hasMultipleItems && showNavigation ? (
            <Button
              type="button"
              variant="outline"
              size="icon"
              className="ui-media-gallery__nav"
              onClick={next}
              aria-label="次のメディア"
            >
              <ChevronRight className="ui-media-gallery__nav-icon" />
            </Button>
          ) : null}
        </div>
      ) : null}

      {shouldShowThumbnails ? (
        <div className="ui-media-gallery__thumbs">
          {items.map((item, itemIndex) => {
            const isActive =
              itemIndex === safeIndex;

            return (
              <div
                key={`${item.id}-${itemIndex}`}
                className={cn(
                  "ui-media-gallery__thumb",
                  isActive &&
                    "ui-media-gallery__thumb--active",
                )}
              >
                {renderThumbnail(
                  item,
                  itemIndex,
                )}

                {editable && onDelete ? (
                  <DeleteButton
                    size="sm"
                    className="ui-media-gallery__thumb-delete"
                    disabled={deleteDisabled}
                    ariaLabel={`メディア ${itemIndex + 1}を削除`}
                    onClick={(event) => {
                      event.stopPropagation();
                      handleDelete(
                        item,
                        itemIndex,
                      );
                    }}
                  />
                ) : null}
              </div>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}