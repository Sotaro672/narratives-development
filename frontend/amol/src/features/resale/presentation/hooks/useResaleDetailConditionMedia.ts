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
} from "../../api/resaleApi";

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
    title:
      image.fileName ||
      "商品状態の写真",
    fileName:
      image.fileName,
    source: "existing",
    image,
  };
}

/**
 * 新規追加画像のObject URLだけを解放する。
 *
 * APIから取得した既存画像URLは、
 * Object URLではないため解放しない。
 */
function revokeNewConditionMediaPreviews(
  items:
    readonly ResaleDetailConditionMediaItem[],
): void {
  items.forEach((item) => {
    if (
      item.source !== "new" ||
      !item.previewUrl
    ) {
      return;
    }

    URL.revokeObjectURL(
      item.previewUrl,
    );
  });
}

/**
 * 再販詳細画面の商品状態画像を管理する。
 *
 * 管理対象:
 * - APIから取得した既存画像
 * - 利用者が新規追加した画像
 * - 削除予定の既存画像ID
 * - MediaUploaderのカルーセル位置
 * - 新規画像のObject URL解放
 */
export function useResaleDetailConditionMedia() {
  const conditionMediaInputRef =
    useRef<HTMLInputElement>(null);

  const conditionMediaCarouselRef =
    useRef<HTMLDivElement>(null);

  const conditionMediaItemsRef =
    useRef<
      ResaleDetailConditionMediaItem[]
    >([]);

  const deletedImageIdsRef =
    useRef<string[]>([]);

  const [
    conditionMediaItems,
    setConditionMediaItems,
  ] = useState<
    ResaleDetailConditionMediaItem[]
  >([]);

  const [
    conditionMediaCurrentIndex,
    setConditionMediaCurrentIndex,
  ] = useState(0);

  const [
    deletedImageIds,
    setDeletedImageIds,
  ] = useState<string[]>([]);

  /**
   * stateとrefの商品状態画像一覧を同時に更新する。
   */
  const replaceConditionMediaItems =
    useCallback(
      (
        items:
          ResaleDetailConditionMediaItem[],
      ) => {
        conditionMediaItemsRef.current =
          items;

        setConditionMediaItems(
          items,
        );
      },
      [],
    );

  /**
   * stateとrefの削除予定画像IDを同時に更新する。
   */
  const replaceDeletedImageIds =
    useCallback(
      (
        imageIds:
          readonly string[],
      ) => {
        const normalizedImageIds =
          Array.from(
            new Set(
              imageIds
                .map((imageId) =>
                  imageId.trim(),
                )
                .filter(Boolean),
            ),
          );

        deletedImageIdsRef.current =
          normalizedImageIds;

        setDeletedImageIds(
          normalizedImageIds,
        );
      },
      [],
    );

  /**
   * APIから取得した画像一覧を初期状態として設定する。
   *
   * 編集キャンセル、再取得、保存完了後の再初期化で使用する。
   */
  const resetConditionMedia =
    useCallback(
      (
        images:
          readonly ResaleConditionImage[],
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

        replaceDeletedImageIds(
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

        conditionMediaCarouselRef.current?.scrollTo(
          {
            left: 0,
            behavior: "auto",
          },
        );
      },
      [
        replaceConditionMediaItems,
        replaceDeletedImageIds,
      ],
    );

  /**
   * ファイル選択時に画像だけを抽出し、
   * 新規画像として一覧へ追加する。
   */
  const handleConditionMediaSelected =
    useCallback(
      (
        event:
          ChangeEvent<HTMLInputElement>,
      ) => {
        const input =
          event.currentTarget;

        const newItems =
          createResaleConditionMediaItems(
            input.files,
          ).map<
            ResaleDetailConditionMediaItem
          >((item) => ({
            ...item,
            source: "new",
          }));

        input.value = "";

        if (
          newItems.length === 0
        ) {
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

  /**
   * 指定された画像を編集対象一覧から除外する。
   *
   * 既存画像:
   * - API削除用の画像IDへ追加する
   *
   * 新規画像:
   * - Object URLを解放する
   * - API削除対象には追加しない
   */
  const handleRemoveConditionMedia =
    useCallback(
      (
        id: string,
      ) => {
        const normalizedId =
          id.trim();

        if (!normalizedId) {
          return;
        }

        const currentItems =
          conditionMediaItemsRef.current;

        const removingItem =
          currentItems.find(
            (item) =>
              item.id ===
              normalizedId,
          );

        if (!removingItem) {
          return;
        }

        if (
          removingItem.source ===
          "existing"
        ) {
          const imageId =
            removingItem.image?.id?.trim() ||
            removingItem.id.trim();

          if (imageId) {
            replaceDeletedImageIds([
              ...deletedImageIdsRef.current,
              imageId,
            ]);
          }
        }

        if (
          removingItem.source ===
            "new" &&
          removingItem.previewUrl
        ) {
          URL.revokeObjectURL(
            removingItem.previewUrl,
          );
        }

        const nextItems =
          currentItems.filter(
            (item) =>
              item.id !==
              normalizedId,
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

  /**
   * MediaUploaderのスクロール位置から
   * 現在表示中の画像番号を算出する。
   */
  const handleConditionMediaCarouselScroll =
    useCallback(() => {
      const carousel =
        conditionMediaCarouselRef.current;

      if (!carousel) {
        return;
      }

      const itemCount =
        conditionMediaItemsRef.current
          .length;

      if (
        itemCount === 0
      ) {
        setConditionMediaCurrentIndex(
          0,
        );
        return;
      }

      const carouselWidth =
        carousel.clientWidth;

      if (
        carouselWidth <= 0
      ) {
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

  /**
   * 指定された画像番号へカルーセルを移動する。
   */
  const handleMoveToConditionMediaSlide =
    useCallback(
      (
        index: number,
      ) => {
        const itemCount =
          conditionMediaItemsRef.current
            .length;

        if (
          itemCount === 0
        ) {
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

  /**
   * 保存対象となる新規画像ファイルだけを返す。
   */
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
              item.source ===
                "new" &&
              item.file instanceof
                File,
          )
          .map(
            (item) =>
              item.file,
          );
      },
      [],
    );

  /**
   * Hook破棄時に新規画像のObject URLを解放する。
   */
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