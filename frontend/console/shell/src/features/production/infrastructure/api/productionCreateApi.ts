// frontend/console/shell/src/features/production/infrastructure/api/productionCreateApi.ts

import type { Brand } from "../../../../shared/types/brand";
import { API_BASE } from "../../../../shared/http/apiBase";
import { fetchJSON } from "../../../../shared/http/fetchJSON";
import type { ProductionCreateContext } from "../../../../shared/types/production";
import { brandRepositoryHTTP } from "../../../brand/infrastructure/http/brandRepositoryHTTP";
import type { ProductBlueprintCategoryPath } from "../../../productBlueprint/domain/productBlueprintCategory";
import { fetchProductBlueprintManagementRows } from "../../../productBlueprint/application/productBlueprintManagementService";
import type { ProductBlueprintListRow } from "../../../productBlueprint/infrastructure/repository/productBlueprintRepositoryHTTP";

export type { Brand, ProductBlueprintCategoryPath };

// ======================================================================
// ブランドAPI
// ======================================================================

export async function loadBrands(): Promise<Brand[]> {
  try {
    const result = await brandRepositoryHTTP.list({ page: 1, perPage: 200 });
    return result.items.filter((brand) => brand.isActive);
  } catch {
    return [];
  }
}

// ======================================================================
// 商品設計一覧API
// ======================================================================

export async function loadProductBlueprints(): Promise<ProductBlueprintListRow[]> {
  try {
    return await fetchProductBlueprintManagementRows();
  } catch {
    return [];
  }
}

// ======================================================================
// Production Create BFF
// ======================================================================

export async function loadProductionCreateContext(
  productBlueprintId: string,
): Promise<ProductionCreateContext> {
  const normalizedProductBlueprintId = String(productBlueprintId ?? "").trim();

  if (!normalizedProductBlueprintId) {
    throw new Error("loadProductionCreateContext: productBlueprintId が空です");
  }

  const url = `${API_BASE}/productions/create-context?productBlueprintId=${encodeURIComponent(normalizedProductBlueprintId)}`;

  return fetchJSON<ProductionCreateContext>(url, {
    method: "GET",
    auth: "required",
  });
}