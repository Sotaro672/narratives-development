// frontend/console/shell/src/features/production/presentation/create/mappers.ts

import type { ProductBlueprintManagementRow } from "../../infrastructure/api/productionCreateApi";

export function filterProductBlueprintsByBrand(
  rows: ProductBlueprintManagementRow[],
  brandName: string | null,
): ProductBlueprintManagementRow[] {
  if (!brandName) {
    return [];
  }

  return rows.filter(
    (productBlueprint) => productBlueprint.brandName === brandName,
  );
}

export function buildProductRows(
  filtered: ProductBlueprintManagementRow[],
): Array<{ id: string; name: string }> {
  return filtered.map((productBlueprint) => ({
    id: productBlueprint.id,
    name: productBlueprint.productName,
  }));
}