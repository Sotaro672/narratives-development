// frontend/amol/src/features/catalog/application/catalogImageFactory.ts

import type { CatalogListImage } from "../types";

export function createCatalogImages(
  images: CatalogListImage[] | undefined,
): CatalogListImage[] {
  const uniqueImages = new Map<string, CatalogListImage>();

  for (const image of images ?? []) {
    if (!image.url) {
      continue;
    }

    uniqueImages.set(image.id, image);
  }

  return Array.from(uniqueImages.values()).sort((a, b) => {
    if (a.displayOrder !== b.displayOrder) {
      return a.displayOrder - b.displayOrder;
    }

    return a.id.localeCompare(b.id);
  });
}

export function resolveActiveCatalogImage(args: {
  images: CatalogListImage[];
  activeImageIndex: number;
}): CatalogListImage | undefined {
  return args.images[args.activeImageIndex];
}

export function hasMultipleCatalogImages(images: CatalogListImage[]): boolean {
  return images.length > 1;
}