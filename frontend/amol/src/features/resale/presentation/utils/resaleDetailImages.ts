// frontend/amol/src/features/resale/presentation/utils/resaleDetailImages.ts

import type {
  MediaGalleryItem,
} from "../../../../components/ui/MediaGallery";

import type {
  ResaleConditionImage,
} from "../../../shared/types/resaleTypes";

/**
 * 商品状態画像を表示順で並べ替える。
 * displayOrderが同じ場合は画像IDで順序を固定する。
 * 元の配列は変更しない。
 */
export function sortResaleConditionImages(
  images: readonly ResaleConditionImage[],
): ResaleConditionImage[] {
  return [...images].sort((first, second) => {
    if (first.displayOrder !== second.displayOrder) {
      return first.displayOrder - second.displayOrder;
    }

    return first.id.localeCompare(second.id, "ja");
  });
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
    type: "image",
  };
}

/**
 * 商品状態画像一覧を並べ替え、
 * MediaGallery用の項目へ変換する。
 */
export function createResaleGalleryItems(
  images: readonly ResaleConditionImage[],
): MediaGalleryItem[] {
  return sortResaleConditionImages(images).map(
    createResaleGalleryItem,
  );
}