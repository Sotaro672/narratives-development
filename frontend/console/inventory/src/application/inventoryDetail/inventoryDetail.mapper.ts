// frontend/console/inventory/src/application/inventoryDetail/inventoryDetail.mapper.ts

import type { InventoryRow } from "../inventoryTypes";
import type {
  InventoryDetailDTO,
  ProductBlueprintModelRefDTO,
  ProductBlueprintPatchDTO,
} from "../../infrastructure/http/inventoryRepositoryHTTP.types";
import type { InventoryDetailViewModel } from "./inventoryDetail.types";

export function buildModelDisplayOrderMap(
  patch: ProductBlueprintPatchDTO | undefined,
): Record<string, number> {
  const refs = patch?.modelRefs as
    | ProductBlueprintModelRefDTO[]
    | null
    | undefined;

  if (!Array.isArray(refs)) return {};

  const out: Record<string, number> = {};

  for (const ref of refs) {
    const modelId = ref.modelId;
    const displayOrder = Number(ref.displayOrder);

    if (!modelId || !Number.isFinite(displayOrder)) continue;

    const normalizedDisplayOrder = Math.trunc(displayOrder);
    if (normalizedDisplayOrder <= 0) continue;

    out[modelId] = normalizedDisplayOrder;
  }

  return out;
}

export function mapInventoryDetailRows(
  dto: InventoryDetailDTO,
  modelOrderById: Record<string, number>,
): InventoryRow[] {
  const rowsRaw = Array.isArray(dto.rows) ? dto.rows : [];

  return rowsRaw.map((row) => {
    const modelId = row.modelId;
    const displayOrder = modelId ? modelOrderById[modelId] : undefined;

    const stockRaw = Number(row.stock ?? 0);
    const stock = Number.isFinite(stockRaw) ? stockRaw : 0;

    return {
      kind: row.kind ?? null,

      modelNumber: row.modelNumber || "",

      size: row.size ?? null,
      color: row.color ?? null,
      rgb: row.rgb ?? null,

      volumeValue: row.volumeValue ?? null,
      volumeUnit: row.volumeUnit ?? null,

      stock,
      displayOrder,
    };
  });
}

export function buildInventoryDetailViewModel(args: {
  inventoryId: string;
  detail: InventoryDetailDTO;
}): InventoryDetailViewModel {
  const { inventoryId, detail } = args;

  const productBlueprintId = detail.productBlueprintId;
  const tokenBlueprintId = detail.tokenBlueprintId;

  if (!productBlueprintId || !tokenBlueprintId) {
    throw new Error("inventory_detail_missing_product_or_token_blueprint_id");
  }

  const productBlueprintPatch =
    detail.productBlueprintPatch ?? ({} as ProductBlueprintPatchDTO);

  const tokenBlueprintPatch = detail.tokenBlueprintPatch;

  const modelOrderById = buildModelDisplayOrderMap(productBlueprintPatch);
  const rows = mapInventoryDetailRows(detail, modelOrderById);

  const totalStockRaw =
    detail.totalStock !== undefined && detail.totalStock !== null
      ? Number(detail.totalStock)
      : rows.reduce((sum, row) => {
          const stockRaw = Number(row.stock ?? 0);
          const stock = Number.isFinite(stockRaw) ? stockRaw : 0;
          return sum + stock;
        }, 0);

  const totalStock = Number.isFinite(totalStockRaw) ? totalStockRaw : 0;

  const productName = productBlueprintPatch.productName || "-";

  const tokenName =
    tokenBlueprintPatch?.tokenName ||
    tokenBlueprintId ||
    "-";

  const category = productBlueprintPatch.productBlueprintCategory ?? null;

  const productBlueprintCategoryName =
    category?.nameJa || category?.nameEn || category?.code || "-";

  const productBlueprintCategoryCode = category?.code;
  const productBlueprintCategoryKind = category?.kind;
  const categoryFields = productBlueprintPatch.categoryFields ?? null;

  return {
    inventoryId,

    productBlueprintId,
    tokenBlueprintId,

    productName,
    tokenName,
    headerTitle: `${productName} / ${tokenName}`,

    productBlueprintCategoryName,
    productBlueprintCategoryCode,
    productBlueprintCategoryKind,
    categoryFields,

    productBlueprintPatch,
    tokenBlueprintPatch,

    updatedAt: detail.updatedAt ? String(detail.updatedAt) : undefined,
    totalStock,

    rows,
  };
}