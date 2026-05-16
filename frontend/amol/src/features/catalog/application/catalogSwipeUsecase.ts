// frontend/amol/src/features/catalog/application/catalogSwipeUsecase.ts

export type CatalogSwipeDirection = "prev" | "next" | "none";

export function resolveCatalogSwipeDirection(args: {
  startX: number;
  startY: number;
  endX: number;
  endY: number;
  thresholdPx: number;
}): CatalogSwipeDirection {
  const diffX = args.endX - args.startX;
  const diffY = args.endY - args.startY;

  if (Math.abs(diffX) < args.thresholdPx) {
    return "none";
  }

  if (Math.abs(diffY) > Math.abs(diffX)) {
    return "none";
  }

  return diffX < 0 ? "next" : "prev";
}