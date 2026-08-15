// frontend/console/shell/src/features/production/presentation/create/mappers.ts

import type { ProductBlueprintListRow } from "../../../productBlueprint/infrastructure/repository/productBlueprintRepositoryHTTP";

export function filterProductBlueprintsByBrand(
  rows: ProductBlueprintListRow[],
  brandName: string | null,
): ProductBlueprintListRow[] {
  if (!brandName) {
    return [];
  }

  return rows.filter((productBlueprint) => productBlueprint.brandName === brandName);
}

export function buildProductRows(
  filtered: ProductBlueprintListRow[],
): Array<{ id: string; name: string }> {
  return filtered.map((productBlueprint) => ({
    id: productBlueprint.id,
    name: productBlueprint.productName,
  }));
}