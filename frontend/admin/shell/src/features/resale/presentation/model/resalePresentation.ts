// frontend/admin/shell/src/features/resale/presentation/model/resalePresentation.ts

import type { ResaleImage } from "../../../../shared/type/resale";
import type { MediaGalleryItem } from "../../../../shared/ui/MediaGallery/MediaGallery";
import type { TabTone } from "../../../../shared/ui/Tab/Tab";

const STATUS_LABELS: Record<string, string> = {
  listing: "出品中",
  suspended: "停止中",
  sold: "売却済み",
};

export function getResaleStatusLabel(status: string): string {
  return STATUS_LABELS[status] ?? status;
}

export function getResaleStatusTone(status: string): TabTone {
  switch (status) {
    case "listing":
      return "success";
    case "suspended":
      return "danger";
    case "sold":
      return "neutral";
    default:
      return "neutral";
  }
}

export function createResaleGalleryItems(
  images: ResaleImage[],
): MediaGalleryItem[] {
  return [...images]
    .sort((a, b) => a.displayOrder - b.displayOrder)
    .filter((image) => Boolean(image.id) && Boolean(image.url))
    .map((image) => ({
      id: image.id,
      url: image.url,
    }));
}