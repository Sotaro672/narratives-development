// frontend/console/productBlueprint/src/application/productBlueprintDetailService.ts

import type { ItemType } from "../domain/entity/catalog";
import type { SizeRow } from "../../../model/src/domain/entity/catalog";
import { updateProductBlueprintHTTP } from "../infrastructure/repository/productBlueprintRepositoryHTTP";

import {
  getProductBlueprintDetailApi,
  type ProductBlueprintDetailResponse,
  type UpdateProductBlueprintParams,
  type NewModelVariationMeasurements,
} from "../infrastructure/api/productBlueprintDetailApi";

import { authorizedFetch } from "../infrastructure/httpClient/authorizedFetch";
import { hexToRgbInt } from "../../../shell/src/shared/util/color";

import {
  updateModelVariation,
  type ModelVariationUpdateRequest,
  deleteRemovedModelVariations,
  type ModelVariationResponse as ModelUpdateServiceVariationResponse,
} from "../../../model/src/application/modelUpdateService";

import {
  createModelVariations,
  type CreateModelVariationRequest,
} from "../../../model/src/infrastructure/repository/modelRepositoryHTTP";

const makeKey = (sizeLabel: string, color: string) => `${sizeLabel}__${color}`;

function isBottomsLike(itemType: ItemType): boolean {
  const normalized = String(itemType ?? "").trim().toLowerCase();
  return normalized === "bottoms" || normalized === "ボトムス";
}

function buildMeasurements(
  itemType: ItemType,
  size: SizeRow,
): NewModelVariationMeasurements {
  const result: NewModelVariationMeasurements = {};

  if (isBottomsLike(itemType)) {
    result["ウエスト"] = size.waist ?? null;
    result["ヒップ"] = size.hip ?? null;
    result["股上"] = size.rise ?? null;
    result["股下"] = size.inseam ?? null;
    result["わたり幅"] = size.thigh ?? null;
    result["裾幅"] = size.hemWidth ?? null;
    return result;
  }

  result["着丈"] = size.length ?? null;
  result["身幅"] = size.width ?? null;
  result["胸囲"] = size.chest ?? null;
  result["肩幅"] = size.shoulder ?? null;
  result["袖丈"] = size.sleeveLength ?? null;

  return result;
}

function buildMeasurementsFromSizeRowForUpdate(
  itemType: ItemType,
  size: SizeRow,
): Record<string, number> | undefined {
  const base = buildMeasurements(itemType, size);
  const result: Record<string, number> = {};

  Object.entries(base).forEach(([k, v]) => {
    if (typeof v === "number" && !Number.isNaN(v)) {
      result[k] = v;
    }
  });

  return Object.keys(result).length > 0 ? result : undefined;
}

function buildMeasurementsForCreate(
  itemType: ItemType,
  size: SizeRow,
): Record<string, number> | undefined {
  return buildMeasurementsFromSizeRowForUpdate(itemType, size);
}

/**
 * UI 側の colorRgbMap(hex) → int を優先し、
 * 無ければ既存 variation の rgb を使う。
 */
function resolveRgbInt(args: {
  colorName: string;
  colorRgbMap?: Record<string, string>;
  fallbackRgb?: number | null;
}): number {
  const { colorName, colorRgbMap = {}, fallbackRgb } = args;

  const rgbHex = String(colorRgbMap[colorName] ?? "").trim();
  const fromHex = rgbHex ? hexToRgbInt(rgbHex) : undefined;
  if (typeof fromHex === "number" && Number.isFinite(fromHex)) {
    return fromHex;
  }

  if (typeof fallbackRgb === "number" && Number.isFinite(fallbackRgb)) {
    return fallbackRgb;
  }

  throw new Error(
    `updateProductBlueprint: rgb が解決できません（color="${colorName}", hex="${rgbHex}"）`,
  );
}

// -----------------------------------------
// GET: 商品設計 詳細
// -----------------------------------------
export async function getProductBlueprintDetail(
  id: string,
): Promise<ProductBlueprintDetailResponse> {
  const trimmed = String(id ?? "").trim();
  if (!trimmed) throw new Error("getProductBlueprintDetail: id が空です");

  return await getProductBlueprintDetailApi(trimmed);
}

// -----------------------------------------
// UPDATE（Blueprint メタ情報 + ModelVariation）
// -----------------------------------------
export async function updateProductBlueprint(
  params: UpdateProductBlueprintParams & {
    sizes?: SizeRow[];
    modelNumbers?: { size: string; color: string }[];
    colorRgbMap?: Record<string, string>;
  },
): Promise<ProductBlueprintDetailResponse> {
  const {
    id,
    productName,
    itemType,
    fit,
    material,
    weight,
    qualityAssurance,
    productIdTagType,
    brandId,
    assigneeId,
    companyId,
    updatedBy,
    colors,
    colorRgbMap = {},
    sizes = [],
    modelNumbers = [],
  } = params;

  if (!id) {
    throw new Error("updateProductBlueprint: id が空です");
  }

  const updated = await updateProductBlueprintHTTP(
    id,
    {
      id,
      productName,
      brandId,
      itemType,
      fit,
      material,
      weight,
      qualityAssurance,
      productIdTagType,
      companyId,
      assigneeId,
      colors: colors ?? [],
      colorRgbMap: colorRgbMap ?? {},
      updatedBy: updatedBy ?? null,
    } satisfies UpdateProductBlueprintParams,
  );

  if (!itemType) {
    return updated;
  }

  const itemTypeValue = itemType as ItemType;

  const variations = await listModelVariationsByProductBlueprintId(id);

  const existingMap = new Map<string, ModelVariationResponse>();
  variations.forEach((v) => {
    const sizeLabel = (v.size ?? "").trim();
    const colorName = (v.color?.name ?? "").trim();
    if (!sizeLabel || !colorName) return;

    existingMap.set(makeKey(sizeLabel, colorName), v);
  });

  const selectedKeys = new Set<string>();
  modelNumbers.forEach((m) => {
    if (!m.size || !m.color) return;
    selectedKeys.add(makeKey(m.size, m.color));
  });

  const measurementsMap = new Map<string, Record<string, number>>();
  sizes.forEach((s) => {
    const ms = buildMeasurementsFromSizeRowForUpdate(itemTypeValue, s);
    if (ms) measurementsMap.set(s.sizeLabel, ms);
  });

  const updateTasks: Promise<void>[] = [];

  existingMap.forEach((v, key) => {
    if (!selectedKeys.has(key)) return;

    const variationId = v.id;
    if (!variationId) return;

    const sizeLabel = (v.size ?? "").trim();
    const colorName = (v.color?.name ?? "").trim();
    if (!sizeLabel || !colorName) return;

    const rgb = resolveRgbInt({
      colorName,
      colorRgbMap,
      fallbackRgb: v.color?.rgb ?? null,
    });

    const measurements = measurementsMap.get(sizeLabel);

    const payload: ModelVariationUpdateRequest = {
      modelNumber: v.modelNumber ?? "",
      size: sizeLabel,
      color: colorName,
      rgb,
      ...(measurements ? { measurements } : {}),
    };

    updateTasks.push(updateModelVariation(variationId, payload).then(() => undefined));
  });

  await Promise.all(updateTasks);

  const createPayloads: CreateModelVariationRequest[] = [];

  selectedKeys.forEach((key) => {
    if (existingMap.has(key)) return;

    const [sizeLabel, colorName] = key.split("__");
    if (!sizeLabel || !colorName) return;

    const sizeRow = sizes.find((s) => s.sizeLabel === sizeLabel);
    if (!sizeRow) return;

    const rgb = resolveRgbInt({
      colorName,
      colorRgbMap,
      fallbackRgb: null,
    });

    const measurements = buildMeasurementsForCreate(itemTypeValue, sizeRow) ?? {};

    const createReq: CreateModelVariationRequest = {
      productBlueprintId: id,
      modelNumber: "",
      size: sizeLabel,
      color: colorName,
      rgb,
      measurements,
    };

    createPayloads.push(createReq);
  });

  if (createPayloads.length > 0) {
    await createModelVariations(id, createPayloads);
  }

  const remainingIds = (variations as ModelUpdateServiceVariationResponse[])
    .filter((v) => {
      const key = makeKey(v.size, v.color?.name ?? "");
      return selectedKeys.has(key);
    })
    .map((v) => v.id);

  await deleteRemovedModelVariations(
    variations as ModelUpdateServiceVariationResponse[],
    remainingIds,
  );

  return updated;
}

// -----------------------------------------
// ModelVariation list
// -----------------------------------------
export type ModelVariationResponse = {
  id: string;
  productBlueprintId: string;
  modelNumber: string;
  size: string;
  color: { name: string; rgb: number };
  measurements?: Record<string, number>;
  createdAt?: string | null;
  createdBy?: string | null;
  updatedAt?: string | null;
  updatedBy?: string | null;
};

export async function listModelVariationsByProductBlueprintId(
  productBlueprintId: string,
): Promise<ModelVariationResponse[]> {
  const id = productBlueprintId.trim();
  if (!id) throw new Error("productBlueprintId が空です");

  const res = await authorizedFetch(
    `/models/by-blueprint/${encodeURIComponent(id)}/variations`,
    {
      method: "GET",
      throwOnError: false,
      acceptJson: true,
    },
  );

  if (!res.ok) {
    throw new Error(
      `モデル一覧の取得に失敗しました（${res.status} ${res.statusText ?? ""}）`,
    );
  }

  const raw = (await res.json()) as ModelVariationResponse[] | null;
  return raw ?? [];
}

// -----------------------------------------
// 商品設計の履歴一覧取得
// -----------------------------------------
export type ProductBlueprintHistoryItem = {
  id: string;
  productName: string;
  brandId: string;
  assigneeId: string;
  updatedAt: string;
  updatedBy?: string;
  deletedAt?: string;
  expireAt?: string;
};

export async function getProductBlueprintHistory(
  productBlueprintId: string,
): Promise<ProductBlueprintHistoryItem[]> {
  const id = productBlueprintId.trim();
  if (!id) {
    throw new Error("getProductBlueprintHistory: productBlueprintId が空です");
  }

  const res = await authorizedFetch(
    `/product-blueprints/${encodeURIComponent(id)}/history`,
    {
      method: "GET",
      throwOnError: false,
      acceptJson: true,
    },
  );

  if (!res.ok) {
    throw new Error(
      `商品設計履歴の取得に失敗しました（${res.status} ${res.statusText ?? ""}）`,
    );
  }

  const raw = (await res.json()) as any[] | null;
  if (!raw) return [];

  return raw.map((v: any): ProductBlueprintHistoryItem => ({
    id: v.id ?? "",
    productName: v.productName ?? "",
    brandId: v.brandId ?? "",
    assigneeId: v.assigneeId ?? "",
    updatedAt: v.updatedAt ?? "",
    updatedBy: v.updatedBy ?? undefined,
    deletedAt: v.deletedAt ?? undefined,
    expireAt: v.expireAt ?? undefined,
  }));
}