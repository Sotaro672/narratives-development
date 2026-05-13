// frontend/console/productBlueprint/src/application/productBlueprintDetailService.ts

import {
  isApparelCategoryCode,
  type ApparelCategoryCode,
  type ApparelMeasurements,
  type ApparelSizeInput,
} from "../domain/entity/apparel";

import { updateProductBlueprintHTTP } from "../infrastructure/repository/productBlueprintRepositoryHTTP";

import {
  getProductBlueprintDetailApi,
  type ProductBlueprintDetailResponse,
  type UpdateProductBlueprintParams,
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

function resolveApparelCategoryCode(
  params: Pick<UpdateProductBlueprintParams, "productBlueprintCategory">,
): ApparelCategoryCode | null {
  const code = String(params.productBlueprintCategory?.code ?? "").trim();

  if (!isApparelCategoryCode(code)) {
    return null;
  }

  return code;
}

function buildApparelMeasurements(
  categoryCode: ApparelCategoryCode,
  size: ApparelSizeInput,
): ApparelMeasurements {
  const result: ApparelMeasurements = {};

  switch (categoryCode) {
    case "apparel.bottoms": {
      result.waist = size.waist ?? null;
      result.hip = size.hip ?? null;
      result.rise = size.rise ?? null;
      result.inseam = size.inseam ?? null;
      result.thighWidth = size.thighWidth ?? null;
      result.hemWidth = size.hemWidth ?? null;
      result.totalLength = size.totalLength ?? null;
      return result;
    }

    case "apparel.dress": {
      result.shoulderWidth = size.shoulderWidth ?? null;
      result.bodyWidth = size.bodyWidth ?? null;
      result.bodyLength = size.bodyLength ?? null;
      result.sleeveLength = size.sleeveLength ?? null;
      result.waist = size.waist ?? null;
      result.hip = size.hip ?? null;
      result.totalLength = size.totalLength ?? null;
      return result;
    }

    case "apparel.tops": {
      result.shoulderWidth = size.shoulderWidth ?? null;
      result.bodyWidth = size.bodyWidth ?? null;
      result.bodyLength = size.bodyLength ?? null;
      result.sleeveLength = size.sleeveLength ?? null;
      result.neckWidth = size.neckWidth ?? null;
      return result;
    }

    case "apparel.outerwear":
    case "apparel.shoes":
    case "apparel.bag":
    case "apparel.accessory":
    default: {
      return result;
    }
  }
}

function normalizeMeasurementsForRequest(
  measurements: ApparelMeasurements,
): Record<string, number> | undefined {
  const result: Record<string, number> = {};

  Object.entries(measurements).forEach(([key, value]) => {
    if (typeof value === "number" && Number.isFinite(value)) {
      result[key] = value;
    }
  });

  return Object.keys(result).length > 0 ? result : undefined;
}

function buildMeasurementsFromSizeRowForUpdate(
  categoryCode: ApparelCategoryCode,
  size: ApparelSizeInput,
): Record<string, number> | undefined {
  return normalizeMeasurementsForRequest(
    buildApparelMeasurements(categoryCode, size),
  );
}

function buildMeasurementsForCreate(
  categoryCode: ApparelCategoryCode,
  size: ApparelSizeInput,
): Record<string, number> | undefined {
  return buildMeasurementsFromSizeRowForUpdate(categoryCode, size);
}

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

  if (!trimmed) {
    throw new Error("getProductBlueprintDetail: id が空です");
  }

  return await getProductBlueprintDetailApi(trimmed);
}

// -----------------------------------------
// UPDATE（Blueprint メタ情報 + apparel ModelVariation）
// -----------------------------------------

export async function updateProductBlueprint(
  params: UpdateProductBlueprintParams & {
    sizes?: ApparelSizeInput[];
    modelNumbers?: { size: string; color: string; code?: string }[];
    colorRgbMap?: Record<string, string>;
  },
): Promise<ProductBlueprintDetailResponse> {
  const {
    id,
    productName,
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
    productBlueprintCategoryId,
    productBlueprintCategory,
    categoryFields,
  } = params;

  if (!id) {
    throw new Error("updateProductBlueprint: id が空です");
  }

  if (!productBlueprintCategoryId?.trim()) {
    throw new Error("updateProductBlueprint: productBlueprintCategoryId が空です");
  }

  if (!productBlueprintCategory?.id?.trim()) {
    throw new Error("updateProductBlueprint: productBlueprintCategory が空です");
  }

  const updated = await updateProductBlueprintHTTP(
    id,
    {
      id,
      productName,
      brandId,
      productBlueprintCategoryId,
      productBlueprintCategory,
      categoryFields: categoryFields ?? null,
      fit,
      material,
      weight,
      qualityAssurance,
      productIdTagType,
      companyId,
      assigneeId,
      colors: colors ?? [],
      colorRgbMap: colorRgbMap ?? {},
      sizes,
      modelNumbers,
      updatedBy: updatedBy ?? null,
    } satisfies UpdateProductBlueprintParams,
  );

  const apparelCategoryCode = resolveApparelCategoryCode({
    productBlueprintCategory,
  });

  if (!apparelCategoryCode) {
    return updated;
  }

  const variations = await listModelVariationsByProductBlueprintId(id);

  const existingMap = new Map<string, ModelVariationResponse>();

  variations.forEach((v) => {
    const sizeLabel = (v.size ?? "").trim();
    const colorName = (v.color?.name ?? "").trim();

    if (!sizeLabel || !colorName) {
      return;
    }

    existingMap.set(makeKey(sizeLabel, colorName), v);
  });

  const selectedKeys = new Set<string>();

  modelNumbers.forEach((m) => {
    const sizeLabel = String(m.size ?? "").trim();
    const colorName = String(m.color ?? "").trim();

    if (!sizeLabel || !colorName) {
      return;
    }

    selectedKeys.add(makeKey(sizeLabel, colorName));
  });

  const measurementsMap = new Map<string, Record<string, number>>();

  sizes.forEach((s) => {
    const ms = buildMeasurementsFromSizeRowForUpdate(apparelCategoryCode, s);

    if (ms) {
      measurementsMap.set(s.sizeLabel, ms);
    }
  });

  const updateTasks: Promise<void>[] = [];

  existingMap.forEach((v, key) => {
    if (!selectedKeys.has(key)) {
      return;
    }

    const variationId = v.id;

    if (!variationId) {
      return;
    }

    const sizeLabel = (v.size ?? "").trim();
    const colorName = (v.color?.name ?? "").trim();

    if (!sizeLabel || !colorName) {
      return;
    }

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

    updateTasks.push(
      updateModelVariation(variationId, payload).then(() => undefined),
    );
  });

  await Promise.all(updateTasks);

  const createPayloads: CreateModelVariationRequest[] = [];

  selectedKeys.forEach((key) => {
    if (existingMap.has(key)) {
      return;
    }

    const [sizeLabel, colorName] = key.split("__");

    if (!sizeLabel || !colorName) {
      return;
    }

    const sizeRow = sizes.find((s) => s.sizeLabel === sizeLabel);

    if (!sizeRow) {
      return;
    }

    const rgb = resolveRgbInt({
      colorName,
      colorRgbMap,
      fallbackRgb: null,
    });

    const measurements = buildMeasurementsForCreate(
      apparelCategoryCode,
      sizeRow,
    );

    const createReq: CreateModelVariationRequest = {
      productBlueprintId: id,
      modelNumber: "",
      size: sizeLabel,
      color: colorName,
      rgb,
      measurements: measurements ?? {},
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

  if (!id) {
    throw new Error("productBlueprintId が空です");
  }

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

  if (!raw) {
    return [];
  }

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