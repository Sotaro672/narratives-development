// frontend/amol/src/features/catalog/application/catalogImageFactory.ts

import type { CatalogListImage } from "../../shared/types/catalog";

export function createCatalogImages(images: CatalogListImage[] | undefined): CatalogListImage[] {
  return images ?? [];
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