// frontend/console/inventory/src/application/inventoryDetail/inventoryDetail.mapper.ts

import type { InventoryRow } from "../inventoryTypes";
import type {
  InventoryDetailDTO,
  ProductBlueprintModelRefDTO,
  ProductBlueprintPatchDTO,
  TokenBlueprintPatchDTO,
} from "../../infrastructure/http/inventoryRepositoryHTTP.types";
import type { InventoryDetailViewModel } from "./inventoryDetail.types";

export function mapTokenBlueprintPatch(
  raw: any,
): TokenBlueprintPatchDTO | undefined {
  if (!raw) return undefined;

  return {
    tokenName: raw.tokenName || null,
    symbol: raw.symbol || null,
    brandId: raw.brandId || null,
    brandName: raw.brandName || null,
    description: raw.description || null,
    iconUrl: raw.iconUrl || null,
  };
}

export function buildModelDisplayOrderMap(
  patch: ProductBlueprintPatchDTO | undefined,
): Record<string, number> {
  const refs = (patch as any)?.modelRefs as
    | ProductBlueprintModelRefDTO[]
    | null
    | undefined;

  if (!Array.isArray(refs)) return {};

  const out: Record<string, number> = {};

  for (const r of refs) {
    const modelId = (r as any)?.modelId;
    const displayOrderRaw = Number((r as any)?.displayOrder);

    if (!modelId || !Number.isFinite(displayOrderRaw)) continue;

    const displayOrder = Math.trunc(displayOrderRaw);
    if (displayOrder <= 0) continue;

    out[modelId] = displayOrder;
  }

  return out;
}

export function mapInventoryDetailRows(
  dto: InventoryDetailDTO,
  modelOrderById: Record<string, number>,
): InventoryRow[] {
  const rowsRaw: any[] = Array.isArray((dto as any)?.rows)
    ? ((dto as any).rows as any[])
    : [];

  return rowsRaw.map((r: any) => {
    const modelId = r.modelId;
    const displayOrder = modelId ? modelOrderById[modelId] : undefined;

    const stockRaw = Number(r.stock ?? 0);
    const stock = Number.isFinite(stockRaw) ? stockRaw : 0;

    return {
      token: r.token || "",
      modelNumber: r.modelNumber,
      size: r.size,
      color: r.color,
      rgb: (r.rgb ?? null) as any,
      stock,
      displayOrder,
    } as InventoryRow;
  });
}

export function buildInventoryDetailViewModel(args: {
  inventoryId: string;
  detail: InventoryDetailDTO;
  tokenBlueprintPatch?: TokenBlueprintPatchDTO;
}): InventoryDetailViewModel {
  const { inventoryId, detail, tokenBlueprintPatch } = args;

  const productBlueprintId = (detail as any)?.productBlueprintId;
  const tokenBlueprintId = (detail as any)?.tokenBlueprintId;

  if (!productBlueprintId || !tokenBlueprintId) {
    throw new Error("inventory_detail_missing_product_or_token_blueprint_id");
  }

  const productBlueprintPatch = ((detail as any)?.productBlueprintPatch ??
    {}) as ProductBlueprintPatchDTO;

  const modelOrderById = buildModelDisplayOrderMap(productBlueprintPatch);

  const rows = mapInventoryDetailRows(detail, modelOrderById);

  const totalStockRaw =
    (detail as any)?.totalStock !== undefined &&
    (detail as any)?.totalStock !== null
      ? Number((detail as any).totalStock)
      : rows.reduce((sum, r) => {
          const stockRaw = Number((r as any).stock ?? 0);
          const stock = Number.isFinite(stockRaw) ? stockRaw : 0;
          return sum + stock;
        }, 0);

  const totalStock = Number.isFinite(totalStockRaw) ? totalStockRaw : 0;

  const productName =
    (productBlueprintPatch as any)?.productName ||
    (detail as any)?.productName ||
    "-";

  const tokenName =
    (tokenBlueprintPatch as any)?.tokenName ||
    (detail as any)?.tokenName ||
    tokenBlueprintId ||
    "-";

  return {
    inventoryId,

    productBlueprintId,
    tokenBlueprintId,

    productName,
    tokenName,
    headerTitle: `${productName} / ${tokenName}`,

    productBlueprintPatch,
    tokenBlueprintPatch,

    updatedAt: (detail as any)?.updatedAt
      ? String((detail as any).updatedAt)
      : undefined,
    totalStock,

    rows,
  };
}