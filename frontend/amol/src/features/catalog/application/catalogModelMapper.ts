// frontend/amol/src/features/catalog/application/catalogModelMapper.ts

import type { CatalogModelVariation } from "../../shared/types/catalog";

export function formatAlcoholVolumeLabel(model: CatalogModelVariation): string {
  const value = model.volumeValue;
  const unit = model.volumeUnit?.trim() ?? "";

  if (typeof value === "number" && Number.isFinite(value) && unit) return `${value}${unit}`;
  if (typeof value === "number" && Number.isFinite(value)) return String(value);

  return "";
}

export function formatAlcoholModelLabel(model: CatalogModelVariation): string {
  const modelNumber = model.modelNumber.trim();
  const volumeLabel = formatAlcoholVolumeLabel(model);

  if (modelNumber && volumeLabel) return `${modelNumber} / ${volumeLabel}`;
  if (volumeLabel) return volumeLabel;
  if (modelNumber) return modelNumber;

  return "-";
}

export function formatApparelSizeLabel(model: CatalogModelVariation): string {
  return model.size?.trim() || "-";
}