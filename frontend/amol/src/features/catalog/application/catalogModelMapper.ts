// frontend/amol/src/features/catalog/application/catalogModelMapper.ts

import type { CatalogModelVariation } from "../types";

export type CatalogModelKind = "apparel" | "alcohol" | "unknown";

export function mapCatalogModelKind(
  model: CatalogModelVariation | null | undefined,
): CatalogModelKind {
  const kind = String(model?.kind ?? "").trim().toLowerCase();

  if (kind === "alcohol") return "alcohol";
  if (kind === "apparel") return "apparel";

  return "unknown";
}

export function mapCatalogKind(
  models: CatalogModelVariation[] | undefined,
): CatalogModelKind {
  const items = Array.isArray(models) ? models : [];

  if (items.some((model) => mapCatalogModelKind(model) === "alcohol")) {
    return "alcohol";
  }

  if (items.some((model) => mapCatalogModelKind(model) === "apparel")) {
    return "apparel";
  }

  return "unknown";
}

export function formatAlcoholVolumeLabel(
  model: CatalogModelVariation,
): string {
  const value = model.volumeValue;
  const unit = String(model.volumeUnit ?? "").trim();

  if (typeof value === "number" && Number.isFinite(value) && unit) {
    return `${value}${unit}`;
  }

  if (typeof value === "number" && Number.isFinite(value)) {
    return String(value);
  }

  return "";
}

export function formatAlcoholModelLabel(
  model: CatalogModelVariation,
): string {
  const modelNumber = String(model.modelNumber ?? "").trim();
  const volumeLabel = formatAlcoholVolumeLabel(model);

  if (modelNumber && volumeLabel) {
    return `${modelNumber} / ${volumeLabel}`;
  }

  if (volumeLabel) {
    return volumeLabel;
  }

  if (modelNumber) {
    return modelNumber;
  }

  return "-";
}

export function createAlcoholSelectionKey(
  model: CatalogModelVariation,
): string {
  return String(model.id ?? "").trim();
}

export function formatAlcoholSizeLabel(
  model: CatalogModelVariation,
): string {
  return (
    String(model.modelNumber ?? "").trim() ||
    formatAlcoholVolumeLabel(model) ||
    "-"
  );
}

export function formatApparelSizeLabel(
  model: CatalogModelVariation,
): string {
  return String(model.size ?? "").trim() || "-";
}