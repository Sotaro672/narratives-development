// frontend/console/productBlueprint/src/application/productBlueprintDetailService.ts

import {
  isApparelCategoryCode,
  type ApparelCategoryCode,
  type ApparelMeasurements,
  type ApparelSizeInput,
} from "../domain/entity/apparel";

import { isAlcoholCategoryCode } from "../domain/entity/alcohol";

import type {
  AlcoholModelNumber,
  Volume,
  VolumeRow,
} from "../../../model/src/application/modelCreateService";

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

const makeApparelKey = (sizeLabel: string, color: string) =>
  `${sizeLabel}__${color}`;

const makeVolumeKey = (volume: Volume): string => {
  const value =
    typeof volume.value === "number" && Number.isFinite(volume.value)
      ? volume.value
      : 0;

  const unit = String(volume.unit ?? "").trim() || "ml";

  if (value <= 0) {
    return "";
  }

  return `${value}${unit}`;
};

function volumeRowToVolume(row: VolumeRow): Volume {
  return {
    value: row.volumeValue,
    unit: row.volumeUnit,
  };
}

function resolveApparelCategoryCode(
  params: Pick<UpdateProductBlueprintParams, "productBlueprintCategory">,
): ApparelCategoryCode | null {
  const code = String(params.productBlueprintCategory?.code ?? "").trim();

  if (!isApparelCategoryCode(code)) {
    return null;
  }

  return code;
}

function isAlcoholCategory(
  params: Pick<UpdateProductBlueprintParams, "productBlueprintCategory">,
): boolean {
  const code = String(params.productBlueprintCategory?.code ?? "").trim();
  return isAlcoholCategoryCode(code);
}

function isApparelVariation(
  variation: ModelVariationResponse,
): variation is Extract<ModelVariationResponse, { kind: "apparel" }> {
  return variation.kind === "apparel";
}

function isAlcoholVariation(
  variation: ModelVariationResponse,
): variation is Extract<ModelVariationResponse, { kind: "alcohol" }> {
  return variation.kind === "alcohol";
}

function buildApparelMeasurements(
  categoryCode: ApparelCategoryCode,
  size: ApparelSizeInput,
): ApparelMeasurements {
  const result: ApparelMeasurements = {};

  switch (categoryCode) {
    case "apparel.bottoms": {
      result["ウエスト"] = size.waist ?? null;
      result["ヒップ"] = size.hip ?? null;
      result["股上"] = size.rise ?? null;
      result["股下"] = size.inseam ?? null;
      result["わたり幅"] = size.thigh ?? null;
      result["裾幅"] = size.hemWidth ?? null;
      return result;
    }

    case "apparel.dress": {
      result["着丈"] = size.length ?? null;
      result["身幅"] = size.width ?? null;
      result["胸囲"] = size.chest ?? null;
      result["肩幅"] = size.shoulder ?? null;
      result["袖丈"] = size.sleeveLength ?? null;
      result["ウエスト"] = size.waist ?? null;
      result["ヒップ"] = size.hip ?? null;
      return result;
    }

    case "apparel.tops": {
      result["着丈"] = size.length ?? null;
      result["身幅"] = size.width ?? null;
      result["胸囲"] = size.chest ?? null;
      result["肩幅"] = size.shoulder ?? null;
      result["袖丈"] = size.sleeveLength ?? null;
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
// UPDATE（Blueprint メタ情報 + ModelVariation）
// -----------------------------------------

export async function updateProductBlueprint(
  params: UpdateProductBlueprintParams & {
    sizes?: ApparelSizeInput[];
    modelNumbers?: { size: string; color: string; code?: string }[];
    colorRgbMap?: Record<string, string>;
    volumes?: VolumeRow[];
    alcoholModelNumbers?: AlcoholModelNumber[];
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
    volumes = [],
    alcoholModelNumbers = [],
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

  const shouldUpdateAlcoholVariations = isAlcoholCategory({
    productBlueprintCategory,
  });

  if (apparelCategoryCode) {
    await updateApparelModelVariations({
      productBlueprintId: id,
      apparelCategoryCode,
      sizes,
      modelNumbers,
      colorRgbMap,
    });
  }

  if (shouldUpdateAlcoholVariations) {
    await updateAlcoholModelVariations({
      productBlueprintId: id,
      volumes,
      alcoholModelNumbers,
    });
  }

  return updated;
}

async function updateApparelModelVariations(args: {
  productBlueprintId: string;
  apparelCategoryCode: ApparelCategoryCode;
  sizes: ApparelSizeInput[];
  modelNumbers: { size: string; color: string; code?: string }[];
  colorRgbMap: Record<string, string>;
}): Promise<void> {
  const {
    productBlueprintId,
    apparelCategoryCode,
    sizes,
    modelNumbers,
    colorRgbMap,
  } = args;

  const variations = await listModelVariationsByProductBlueprintId(
    productBlueprintId,
  );

  const apparelVariations = variations.filter(isApparelVariation);

  const existingMap = new Map<
    string,
    Extract<ModelVariationResponse, { kind: "apparel" }>
  >();

  apparelVariations.forEach((variation) => {
    const sizeLabel = String(variation.size ?? "").trim();
    const colorName = String(variation.color?.name ?? "").trim();

    if (!sizeLabel || !colorName) {
      return;
    }

    existingMap.set(makeApparelKey(sizeLabel, colorName), variation);
  });

  const selectedKeys = new Set<string>();

  modelNumbers.forEach((modelNumber) => {
    const sizeLabel = String(modelNumber.size ?? "").trim();
    const colorName = String(modelNumber.color ?? "").trim();

    if (!sizeLabel || !colorName) {
      return;
    }

    selectedKeys.add(makeApparelKey(sizeLabel, colorName));
  });

  const measurementsMap = new Map<string, Record<string, number>>();

  sizes.forEach((size) => {
    const sizeLabel = String(size.sizeLabel ?? "").trim();

    if (!sizeLabel) {
      return;
    }

    const measurements = buildMeasurementsFromSizeRowForUpdate(
      apparelCategoryCode,
      size,
    );

    if (measurements) {
      measurementsMap.set(sizeLabel, measurements);
    }
  });

  const updateTasks: Promise<void>[] = [];

  existingMap.forEach((variation, key) => {
    if (!selectedKeys.has(key)) {
      return;
    }

    const variationId = variation.id;

    if (!variationId) {
      return;
    }

    const sizeLabel = String(variation.size ?? "").trim();
    const colorName = String(variation.color?.name ?? "").trim();

    if (!sizeLabel || !colorName) {
      return;
    }

    const rgb = resolveRgbInt({
      colorName,
      colorRgbMap,
      fallbackRgb: variation.color?.rgb ?? null,
    });

    const measurements = measurementsMap.get(sizeLabel);

    const modelNumber =
      modelNumbers.find(
        (candidate) =>
          String(candidate.size ?? "").trim() === sizeLabel &&
          String(candidate.color ?? "").trim() === colorName,
      )?.code ?? "";

    const payload: ModelVariationUpdateRequest = {
      kind: "apparel",
      modelNumber,
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

    const sizeRow = sizes.find((size) => size.sizeLabel === sizeLabel);

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

    const modelNumber =
      modelNumbers.find(
        (candidate) =>
          String(candidate.size ?? "").trim() === sizeLabel &&
          String(candidate.color ?? "").trim() === colorName,
      )?.code ?? "";

    const createReq: CreateModelVariationRequest = {
      kind: "apparel",
      productBlueprintId,
      modelNumber,
      size: sizeLabel,
      color: colorName,
      rgb,
      measurements: measurements ?? {},
    };

    createPayloads.push(createReq);
  });

  if (createPayloads.length > 0) {
    await createModelVariations(productBlueprintId, createPayloads);
  }

  const remainingIds = apparelVariations
    .filter((variation) => {
      const key = makeApparelKey(variation.size, variation.color?.name ?? "");
      return selectedKeys.has(key);
    })
    .map((variation) => variation.id);

  await deleteRemovedModelVariations(
    apparelVariations as ModelUpdateServiceVariationResponse[],
    remainingIds,
  );
}

async function updateAlcoholModelVariations(args: {
  productBlueprintId: string;
  volumes: VolumeRow[];
  alcoholModelNumbers: AlcoholModelNumber[];
}): Promise<void> {
  const { productBlueprintId, volumes, alcoholModelNumbers } = args;

  const variations = await listModelVariationsByProductBlueprintId(
    productBlueprintId,
  );

  const alcoholVariations = variations.filter(isAlcoholVariation);

  const existingMap = new Map<
    string,
    Extract<ModelVariationResponse, { kind: "alcohol" }>
  >();

  alcoholVariations.forEach((variation) => {
    const key = makeVolumeKey(variation.volume);

    if (!key) {
      return;
    }

    existingMap.set(key, variation);
  });

  const selectedKeys = new Set<string>();

  volumes.forEach((row) => {
    const volume = volumeRowToVolume(row);
    const key = makeVolumeKey(volume);

    if (!key) {
      return;
    }

    selectedKeys.add(key);
  });

  const modelNumberMap = new Map<string, AlcoholModelNumber>();

  alcoholModelNumbers.forEach((modelNumber) => {
    const key = makeVolumeKey(modelNumber.volume);

    if (!key) {
      return;
    }

    modelNumberMap.set(key, modelNumber);
    selectedKeys.add(key);
  });

  const updateTasks: Promise<void>[] = [];

  existingMap.forEach((variation, key) => {
    if (!selectedKeys.has(key)) {
      return;
    }

    const variationId = variation.id;

    if (!variationId) {
      return;
    }

    const modelNumber = modelNumberMap.get(key);

    if (!modelNumber) {
      return;
    }

    const payload: ModelVariationUpdateRequest = {
      kind: "alcohol",
      modelNumber: modelNumber.code,
      volume: modelNumber.volume,
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

    const modelNumber = modelNumberMap.get(key);

    if (!modelNumber) {
      return;
    }

    const createReq: CreateModelVariationRequest = {
      kind: "alcohol",
      productBlueprintId,
      modelNumber: modelNumber.code,
      volume: modelNumber.volume,
    };

    createPayloads.push(createReq);
  });

  if (createPayloads.length > 0) {
    await createModelVariations(productBlueprintId, createPayloads);
  }

  const remainingIds = alcoholVariations
    .filter((variation) => {
      const key = makeVolumeKey(variation.volume);
      return selectedKeys.has(key);
    })
    .map((variation) => variation.id);

  await deleteRemovedModelVariations(
    alcoholVariations as ModelUpdateServiceVariationResponse[],
    remainingIds,
  );
}

// -----------------------------------------
// ModelVariation list
// -----------------------------------------

export type ApparelModelVariationResponse = {
  id: string;
  productBlueprintId: string;
  kind: "apparel";
  modelNumber: string;
  size: string;
  color: { name: string; rgb: number };
  measurements?: Record<string, number>;
  createdAt?: string | null;
  createdBy?: string | null;
  updatedAt?: string | null;
  updatedBy?: string | null;
};

export type AlcoholModelVariationResponse = {
  id: string;
  productBlueprintId: string;
  kind: "alcohol";
  modelNumber: string;
  volume: Volume;
  createdAt?: string | null;
  createdBy?: string | null;
  updatedAt?: string | null;
  updatedBy?: string | null;
};

export type ModelVariationResponse =
  | ApparelModelVariationResponse
  | AlcoholModelVariationResponse;

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