//frontend\amol\src\features\catalog\utils\model.ts
import type {
  CatalogInventory,
  CatalogModelVariation,
} from "../types";

export function getModelColorKey(model: CatalogModelVariation): string {
  const colorName = model.colorName?.trim() || "-";
  const colorRGB = Number.isFinite(model.colorRGB) ? model.colorRGB : 0;

  return `${colorName}__${colorRGB}`;
}

export function getAvailableStock(
  inventory: CatalogInventory | undefined,
  modelId: string,
): number {
  const stock = inventory?.stock?.[modelId];

  if (!stock) {
    return 0;
  }

  return Math.max(0, stock.accumulation - stock.reservedCount);
}