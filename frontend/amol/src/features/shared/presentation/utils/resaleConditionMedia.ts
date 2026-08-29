// frontend/amol/src/features/shared/presentation/utils/resaleConditionMedia.ts

import type {
  MediaGalleryItem,
} from "../../../../components/ui/MediaGallery";

import type {
  ResaleConditionImage,
} from "../../types/resale";

export type ResaleConditionMediaFallback = {
  id: string;
  url?: string | null;
  fileName?: string;
};

export type CreateResaleConditionGalleryItemsOptions = {
  fallback?: ResaleConditionMediaFallback | null;
};

function getMediaTypeFromUrl(url: string): string {
  const normalizedUrl = url.toLowerCase();

  if (
    normalizedUrl.includes(".mp4") ||
    normalizedUrl.includes(".mov") ||
    normalizedUrl.includes(".webm")
  ) {
    return "video/mp4";
  }

  return "image/*";
}

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

export function createResaleConditionGalleryItem(
  image: ResaleConditionImage,
): MediaGalleryItem {
  return {
    id: image.id,
    url: image.url,
    fileName: "出品画像",
    type: getMediaTypeFromUrl(image.url),
  };
}

function createFallbackGalleryItem(
  fallback: ResaleConditionMediaFallback | null | undefined,
): MediaGalleryItem | null {
  const id = fallback?.id?.trim() ?? "";
  const url = fallback?.url?.trim() ?? "";

  if (!id || !url) {
    return null;
  }

  return {
    id,
    url,
    fileName: fallback?.fileName?.trim() || "出品画像",
    type: getMediaTypeFromUrl(url),
  };
}

export function createResaleConditionGalleryItems(
  images: readonly ResaleConditionImage[],
  options: CreateResaleConditionGalleryItemsOptions = {},
): MediaGalleryItem[] {
  const galleryItems = sortResaleConditionImages(images).map(
    createResaleConditionGalleryItem,
  );

  if (galleryItems.length > 0) {
    return galleryItems;
  }

  const fallbackItem = createFallbackGalleryItem(options.fallback);

  return fallbackItem ? [fallbackItem] : [];
}