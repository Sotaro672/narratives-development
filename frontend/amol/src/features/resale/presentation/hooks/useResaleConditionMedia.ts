// frontend/amol/src/features/resale/presentation/hooks/useResaleConditionMedia.ts

import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type ChangeEvent,
} from "react";

import type {
  ResaleConditionMediaItem,
} from "../types/resaleCreatePageTypes";

function createConditionMediaItem(
  file: File,
): ResaleConditionMediaItem {
  return {
    id: [
      file.name,
      file.size,
      file.lastModified,
      crypto.randomUUID(),
    ].join("-"),
    type: "image",
    previewUrl:
      URL.createObjectURL(file),
    title: file.name,
    fileName: file.name,
    file,
  };
}

function isImageFile(
  file: File,
): boolean {
  return file.type.startsWith(
    "image/",
  );
}

export function useResaleConditionMedia() {
  const conditionMediaInputRef =
    useRef<HTMLInputElement>(null);

  const conditionMediaCarouselRef =
    useRef<HTMLDivElement>(null);

  const conditionMediaItemsRef =
    useRef<
      ResaleConditionMediaItem[]
    >([]);

  const previewUrlsRef =
    useRef<Set<string>>(
      new Set(),
    );

  const [
    conditionMediaItems,
    setConditionMediaItems,
  ] = useState<
    ResaleConditionMediaItem[]
  >([]);

  const [
    conditionMediaCurrentIndex,
    setConditionMediaCurrentIndex,
  ] = useState(0);

  const replaceConditionMediaItems =
    useCallback(
      (
        items:
          ResaleConditionMediaItem[],
      ) => {
        conditionMediaItemsRef.current =
          items;

        setConditionMediaItems(
          items,
        );
      },
      [],
    );

  const revokePreviewUrl =
    useCallback(
      (
        previewUrl: string,
      ) => {
        URL.revokeObjectURL(
          previewUrl,
        );

        previewUrlsRef.current.delete(
          previewUrl,
        );
      },
      [],
    );

  const handleConditionMediaSelected =
    useCallback(
      (
        event:
          ChangeEvent<HTMLInputElement>,
      ) => {
        const input =
          event.currentTarget;

        const files = Array.from(
          input.files ?? [],
        ).filter(isImageFile);

        input.value = "";

        if (files.length === 0) {
          return;
        }

        const addedItems =
          files.map((file) => {
            const item =
              createConditionMediaItem(
                file,
              );

            previewUrlsRef.current.add(
              item.previewUrl,
            );

            return item;
          });

        replaceConditionMediaItems([
          ...conditionMediaItemsRef
            .current,
          ...addedItems,
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
        const currentItems =
          conditionMediaItemsRef
            .current;

        const removingItem =
          currentItems.find(
            (item) =>
              item.id === id,
          );

        if (!removingItem) {
          return;
        }

        revokePreviewUrl(
          removingItem.previewUrl,
        );

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
        revokePreviewUrl,
      ],
    );

  const handleConditionMediaCarouselScroll =
    useCallback(() => {
      const carousel =
        conditionMediaCarouselRef
          .current;

      if (!carousel) {
        return;
      }

      const width =
        carousel.clientWidth;

      if (width <= 0) {
        return;
      }

      const itemCount =
        conditionMediaItemsRef
          .current.length;

      if (itemCount === 0) {
        setConditionMediaCurrentIndex(
          0,
        );
        return;
      }

      const nextIndex =
        Math.round(
          carousel.scrollLeft /
            width,
        );

      setConditionMediaCurrentIndex(
        Math.min(
          Math.max(
            nextIndex,
            0,
          ),
          itemCount - 1,
        ),
      );
    }, []);

  const handleMoveToConditionMediaSlide =
    useCallback(
      (
        index: number,
      ) => {
        const itemCount =
          conditionMediaItemsRef
            .current.length;

        if (itemCount === 0) {
          setConditionMediaCurrentIndex(
            0,
          );
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
          conditionMediaCarouselRef
            .current;

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

  const clearConditionMedia =
    useCallback(() => {
      previewUrlsRef.current.forEach(
        (previewUrl) => {
          URL.revokeObjectURL(
            previewUrl,
          );
        },
      );

      previewUrlsRef.current.clear();

      replaceConditionMediaItems(
        [],
      );

      setConditionMediaCurrentIndex(
        0,
      );

      if (
        conditionMediaInputRef.current
      ) {
        conditionMediaInputRef.current.value =
          "";
      }

      conditionMediaCarouselRef
        .current
        ?.scrollTo({
          left: 0,
          behavior: "auto",
        });
    }, [
      replaceConditionMediaItems,
    ]);

  useEffect(() => {
    return () => {
      previewUrlsRef.current.forEach(
        (previewUrl) => {
          URL.revokeObjectURL(
            previewUrl,
          );
        },
      );

      previewUrlsRef.current.clear();
      conditionMediaItemsRef.current =
        [];
    };
  }, []);

  return {
    conditionMediaItems,
    conditionMediaCurrentIndex,
    conditionMediaInputRef,
    conditionMediaCarouselRef,

    hasConditionMedia:
      conditionMediaItems.length > 0,

    handleConditionMediaSelected,
    handleRemoveConditionMedia,
    handleConditionMediaCarouselScroll,
    handleMoveToConditionMediaSlide,
    clearConditionMedia,
  };
}