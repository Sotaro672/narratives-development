// frontend/console/shell/src/features/production/presentation/create/mappers.ts

import type { Brand } from "../../../../shared/types/brand";
import type { ProductBlueprintManagementRow } from "../../../productBlueprint/infrastructure/query/productBlueprintQuery";

export function buildBrandOptions(brands: Brand[]): string[] {
  return brands.map((brand) => brand.name).filter(Boolean);
}

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