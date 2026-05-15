// frontend/console/inventory/src/application/inventoryDetail/inventoryDetail.mapper.ts

import type { InventoryRow } from "../inventoryTypes";
import type {
  InventoryDetailDTO,
  ProductBlueprintModelRefDTO,
  ProductBlueprintPatchDTO,
  TokenBlueprintPatchDTO,
} from "../../infrastructure/http/inventoryRepositoryHTTP.types";
import type { InventoryDetailViewModel } from "./inventoryDetail.types";

type InventoryModelVariationColorDTO =
  | string
  | {
      name?: string | null;
      rgb?: number | null;
    };

type InventoryModelVariationDTO = {
  id: string;
  productBlueprintId?: string;
  kind?: string | null;
  modelNumber?: string | null;
  size?: string | null;
  color?: InventoryModelVariationColorDTO | null;
  rgb?: number | null;
  volume?: {
    value?: number | null;
    unit?: string | null;
  } | null;
};

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

export function buildModelVariationMap(
  modelVariations?: InventoryModelVariationDTO[] | null,
): Record<string, InventoryModelVariationDTO> {
  if (!Array.isArray(modelVariations)) return {};

  const out: Record<string, InventoryModelVariationDTO> = {};

  for (const variation of modelVariations) {
    if (!variation.id) continue;
    out[variation.id] = variation;
  }

  return out;
}

function getVariationColorName(
  variation: InventoryModelVariationDTO | undefined,
): string | null {
  const color = variation?.color;

  if (typeof color === "string") {
    const trimmed = color.trim();
    return trimmed || null;
  }

  if (color && typeof color === "object") {
    const name = String(color.name ?? "").trim();
    return name || null;
  }

  return null;
}

function getVariationRgb(
  variation: InventoryModelVariationDTO | undefined,
): number | null {
  const color = variation?.color;

  if (
    color &&
    typeof color === "object" &&
    typeof color.rgb === "number" &&
    Number.isFinite(color.rgb)
  ) {
    return color.rgb;
  }

  if (typeof variation?.rgb === "number" && Number.isFinite(variation.rgb)) {
    return variation.rgb;
  }

  return null;
}

function getVariationVolumeValue(
  variation: InventoryModelVariationDTO | undefined,
): number | null {
  const value = variation?.volume?.value;

  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }

  return null;
}

function getVariationVolumeUnit(
  variation: InventoryModelVariationDTO | undefined,
): string | null {
  const unit = String(variation?.volume?.unit ?? "").trim();
  return unit || null;
}

export function mapInventoryDetailRows(
  dto: InventoryDetailDTO,
  modelOrderById: Record<string, number>,
  modelVariationById: Record<string, InventoryModelVariationDTO> = {},
): InventoryRow[] {
  const rowsRaw = Array.isArray(dto.rows) ? dto.rows : [];

  return rowsRaw.map((row) => {
    const modelId = row.modelId;
    const displayOrder = modelId ? modelOrderById[modelId] : undefined;
    const variation = modelId ? modelVariationById[modelId] : undefined;

    const stockRaw = Number(row.stock ?? 0);
    const stock = Number.isFinite(stockRaw) ? stockRaw : 0;

    return {
      token: row.token || "",

      kind: variation?.kind ?? row.kind ?? null,

      modelNumber: variation?.modelNumber || row.modelNumber || "",

      size: variation?.size ?? row.size ?? null,
      color: getVariationColorName(variation) ?? row.color ?? null,
      rgb: getVariationRgb(variation) ?? row.rgb ?? null,

      volumeValue: getVariationVolumeValue(variation),
      volumeUnit: getVariationVolumeUnit(variation),

      stock,
      displayOrder,
    };
  });
}

export function buildInventoryDetailViewModel(args: {
  inventoryId: string;
  detail: InventoryDetailDTO;
  tokenBlueprintPatch?: TokenBlueprintPatchDTO;
  modelVariations?: InventoryModelVariationDTO[] | null;
}): InventoryDetailViewModel {
  const { inventoryId, detail, tokenBlueprintPatch, modelVariations } = args;

  const productBlueprintId = detail.productBlueprintId;
  const tokenBlueprintId = detail.tokenBlueprintId;

  if (!productBlueprintId || !tokenBlueprintId) {
    throw new Error("inventory_detail_missing_product_or_token_blueprint_id");
  }

  const productBlueprintPatch =
    detail.productBlueprintPatch ?? ({} as ProductBlueprintPatchDTO);

  const modelOrderById = buildModelDisplayOrderMap(productBlueprintPatch);
  const modelVariationById = buildModelVariationMap(modelVariations);

  const rows = mapInventoryDetailRows(
    detail,
    modelOrderById,
    modelVariationById,
  );

  const totalStockRaw =
    detail.totalStock !== undefined && detail.totalStock !== null
      ? Number(detail.totalStock)
      : rows.reduce((sum, row) => {
          const stockRaw = Number(row.stock ?? 0);
          const stock = Number.isFinite(stockRaw) ? stockRaw : 0;
          return sum + stock;
        }, 0);

  const totalStock = Number.isFinite(totalStockRaw) ? totalStockRaw : 0;

  const productName =
    productBlueprintPatch.productName || (detail as any).productName || "-";

  const tokenName =
    tokenBlueprintPatch?.tokenName ||
    (detail as any).tokenName ||
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