// frontend/amol/src/features/resale/presentation/utils/resaleDetailImages.ts

import type {
  MediaGalleryItem,
} from "../../../../components/ui/MediaGallery";

import type {
  ResaleConditionImage,
} from "../../api/resaleApi";

/**
 * 商品状態画像を表示順で並べ替える。
 *
 * displayOrderが同じ場合は、画像IDで順序を固定する。
 * 元の配列は変更しない。
 */
export function sortResaleConditionImages(
  images: readonly ResaleConditionImage[],
): ResaleConditionImage[] {
  return [...images].sort(
    (
      first,
      second,
    ) => {
      const firstDisplayOrder =
        Number(
          first.displayOrder ?? 0,
        );

      const secondDisplayOrder =
        Number(
          second.displayOrder ?? 0,
        );

      const normalizedFirstDisplayOrder =
        Number.isFinite(
          firstDisplayOrder,
        )
          ? firstDisplayOrder
          : 0;

      const normalizedSecondDisplayOrder =
        Number.isFinite(
          secondDisplayOrder,
        )
          ? secondDisplayOrder
          : 0;

      if (
        normalizedFirstDisplayOrder !==
        normalizedSecondDisplayOrder
      ) {
        return (
          normalizedFirstDisplayOrder -
          normalizedSecondDisplayOrder
        );
      }

      return String(
        first.id ?? "",
      ).localeCompare(
        String(
          second.id ?? "",
        ),
        "ja",
      );
    },
  );
}

/**
 * 商品状態画像をMediaGallery用の項目へ変換する。
 */
export function createResaleGalleryItem(
  image: ResaleConditionImage,
): MediaGalleryItem {
  return {
    id: image.id,
    url: image.url,
    fileName:
      image.fileName,
    type:
      image.mimeType,
  };
}

/**
 * 商品状態画像一覧を並べ替え、
 * MediaGallery用の項目へ変換する。
 */
export function createResaleGalleryItems(
  images: readonly ResaleConditionImage[],
): MediaGalleryItem[] {
  return sortResaleConditionImages(
    images,
  ).map(
    createResaleGalleryItem,
  );
}