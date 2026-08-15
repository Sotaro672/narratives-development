// frontend/amol/src/features/resale/presentation/hooks/useResaleDetailConditionMedia.ts

import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type ChangeEvent,
} from "react";

import type {
  ResaleConditionImage,
} from "../../../shared/types/resale";

import type {
  ResaleDetailConditionMediaItem,
} from "../types/resaleDetailPageTypes";

import {
  createResaleConditionMediaItems,
} from "../utils/resaleConditionMedia";

import {
  sortResaleConditionImages,
} from "../utils/resaleDetailImages";

/**
 * APIから取得した既存画像を、
 * 詳細編集画面のMediaUploader用項目へ変換する。
 */
function createExistingConditionMediaItem(
  image: ResaleConditionImage,
): ResaleDetailConditionMediaItem {
  return {
    id: image.id,
    type: "image",
    previewUrl: image.url,
    title: "商品状態の写真",
    source: "existing",
    image,
  };
}

/**
 * 新規追加画像のObject URLだけを解放する。
 */
function revokeNewConditionMediaPreviews(
  items: readonly ResaleDetailConditionMediaItem[],
): void {
  items.forEach((item) => {
    if (
      item.source !== "new" ||
      !item.previewUrl
    ) {
      return;
    }

    URL.revokeObjectURL(item.previewUrl);
  });
}

/**
 * 再販詳細画面の商品状態画像を管理する。
 */
export function useResaleDetailConditionMedia() {
  const conditionMediaInputRef =
    useRef<HTMLInputElement>(null);

  const conditionMediaCarouselRef =
    useRef<HTMLDivElement>(null);

  const conditionMediaItemsRef =
    useRef<ResaleDetailConditionMediaItem[]>([]);

  const deletedImageIdsRef =
    useRef<string[]>([]);

  const [
    conditionMediaItems,
    setConditionMediaItems,
  ] = useState<ResaleDetailConditionMediaItem[]>([]);

  const [
    conditionMediaCurrentIndex,
    setConditionMediaCurrentIndex,
  ] = useState(0);

  const [
    deletedImageIds,
    setDeletedImageIds,
  ] = useState<string[]>([]);

  const replaceConditionMediaItems =
    useCallback(
      (
        items: ResaleDetailConditionMediaItem[],
      ) => {
        conditionMediaItemsRef.current = items;
        setConditionMediaItems(items);
      },
      [],
    );

  const replaceDeletedImageIds =
    useCallback(
      (
        imageIds: readonly string[],
      ) => {
        const nextImageIds =
          Array.from(new Set(imageIds));

        deletedImageIdsRef.current =
          nextImageIds;

        setDeletedImageIds(
          nextImageIds,
        );
      },
      [],
    );

  const resetConditionMedia =
    useCallback(
      (
        images: readonly ResaleConditionImage[],
      ) => {
        revokeNewConditionMediaPreviews(
          conditionMediaItemsRef.current,
        );

        const nextItems =
          sortResaleConditionImages(
            images,
          ).map(
            createExistingConditionMediaItem,
          );

        replaceConditionMediaItems(
          nextItems,
        );

        replaceDeletedImageIds([]);
        setConditionMediaCurrentIndex(0);

        if (
          conditionMediaInputRef.current
        ) {
          conditionMediaInputRef.current.value =
            "";
        }

        conditionMediaCarouselRef.current?.scrollTo({
          left: 0,
          behavior: "auto",
        });
      },
      [
        replaceConditionMediaItems,
        replaceDeletedImageIds,
      ],
    );

  const handleConditionMediaSelected =
    useCallback(
      (
        event: ChangeEvent<HTMLInputElement>,
      ) => {
        const input =
          event.currentTarget;

        const newItems =
          createResaleConditionMediaItems(
            input.files,
          ).map<ResaleDetailConditionMediaItem>(
            (item) => ({
              ...item,
              source: "new",
            }),
          );

        input.value = "";

        if (newItems.length === 0) {
          return;
        }

        replaceConditionMediaItems([
          ...conditionMediaItemsRef.current,
          ...newItems,
        ]);
      },
      [
        replaceConditionMediaItems,
      ],
    );

  const handleRemoveConditionMedia =
    useCallback(
      (
        id: string,
      ) => {
        if (!id) {
          return;
        }

        const currentItems =
          conditionMediaItemsRef.current;

        const removingItem =
          currentItems.find(
            (item) =>
              item.id === id,
          );

        if (!removingItem) {
          return;
        }

        if (
          removingItem.source ===
          "existing"
        ) {
          replaceDeletedImageIds([
            ...deletedImageIdsRef.current,
            removingItem.id,
          ]);
        }

        if (
          removingItem.source === "new" &&
          removingItem.previewUrl
        ) {
          URL.revokeObjectURL(
            removingItem.previewUrl,
          );
        }

        const nextItems =
          currentItems.filter(
            (item) =>
              item.id !== id,
          );

        replaceConditionMediaItems(
          nextItems,
        );

        setConditionMediaCurrentIndex(
          (currentIndex) => {
            if (
              nextItems.length === 0
            ) {
              return 0;
            }

            return Math.min(
              currentIndex,
              nextItems.length - 1,
            );
          },
        );
      },
      [
        replaceConditionMediaItems,
        replaceDeletedImageIds,
      ],
    );

  const handleConditionMediaCarouselScroll =
    useCallback(() => {
      const carousel =
        conditionMediaCarouselRef.current;

      if (!carousel) {
        return;
      }

      const itemCount =
        conditionMediaItemsRef.current.length;

      if (itemCount === 0) {
        setConditionMediaCurrentIndex(0);
        return;
      }

      const carouselWidth =
        carousel.clientWidth;

      if (carouselWidth <= 0) {
        return;
      }

      const calculatedIndex =
        Math.round(
          carousel.scrollLeft /
          carouselWidth,
        );

      const nextIndex =
        Math.min(
          Math.max(
            calculatedIndex,
            0,
          ),
          itemCount - 1,
        );

      setConditionMediaCurrentIndex(
        nextIndex,
      );
    }, []);

  const handleMoveToConditionMediaSlide =
    useCallback(
      (
        index: number,
      ) => {
        const itemCount =
          conditionMediaItemsRef.current.length;

        if (itemCount === 0) {
          setConditionMediaCurrentIndex(0);
          return;
        }

        const nextIndex =
          Math.min(
            Math.max(
              index,
              0,
            ),
            itemCount - 1,
          );

        const carousel =
          conditionMediaCarouselRef.current;

        if (carousel) {
          carousel.scrollTo({
            left:
              carousel.clientWidth *
              nextIndex,
            behavior: "smooth",
          });
        }

        setConditionMediaCurrentIndex(
          nextIndex,
        );
      },
      [],
    );

  const getNewConditionFiles =
    useCallback(
      (): File[] => {
        return conditionMediaItemsRef.current
          .filter(
            (
              item,
            ): item is
              ResaleDetailConditionMediaItem & {
                source: "new";
                file: File;
              } =>
              item.source === "new" &&
              item.file instanceof File,
          )
          .map(
            (item) =>
              item.file,
          );
      },
      [],
    );

  useEffect(() => {
    return () => {
      revokeNewConditionMediaPreviews(
        conditionMediaItemsRef.current,
      );

      conditionMediaItemsRef.current =
        [];

      deletedImageIdsRef.current =
        [];
    };
  }, []);

  return {
    conditionMediaItems,
    conditionMediaCurrentIndex,
    conditionMediaInputRef,
    conditionMediaCarouselRef,
    deletedImageIds,

    hasConditionMedia:
      conditionMediaItems.length > 0,

    resetConditionMedia,
    getNewConditionFiles,

    handleConditionMediaSelected,
    handleRemoveConditionMedia,
    handleConditionMediaCarouselScroll,
    handleMoveToConditionMediaSlide,
  };
}