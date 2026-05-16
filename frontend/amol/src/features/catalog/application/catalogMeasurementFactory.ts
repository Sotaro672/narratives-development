// frontend/amol/src/features/catalog/application/catalogMeasurementFactory.ts

import type {
  CatalogModelVariation,
  MeasurementTableRow,
} from "../types";
import {
  formatApparelSizeLabel,
  mapCatalogModelKind,
} from "./catalogModelMapper";

export function createCatalogMeasurementRows(args: {
  models: CatalogModelVariation[] | undefined;
  isAlcoholCatalog: boolean;
}): MeasurementTableRow[] {
  if (args.isAlcoholCatalog) {
    return [];
  }

  const rows = new Map<string, MeasurementTableRow>();

  for (const model of args.models ?? []) {
    if (mapCatalogModelKind(model) === "alcohol") {
      continue;
    }

    const size = formatApparelSizeLabel(model);

    if (rows.has(size)) {
      continue;
    }

    rows.set(size, {
      id: model.id,
      size,
      measurements: model.measurements ?? {},
    });
  }

  return Array.from(rows.values());
}

export function createCatalogMeasurementKeys(
  rows: MeasurementTableRow[],
): string[] {
  const keys = new Set<string>();

  for (const row of rows) {
    for (const key of Object.keys(row.measurements ?? {})) {
      keys.add(key);
    }
  }

  return Array.from(keys);
}

export function shouldShowCatalogMeasurementTable(args: {
  isAlcoholCatalog: boolean;
  measurementRows: MeasurementTableRow[];
  measurementKeys: string[];
}): boolean {
  return (
    !args.isAlcoholCatalog &&
    args.measurementRows.length > 0 &&
    args.measurementKeys.length > 0
  );
}