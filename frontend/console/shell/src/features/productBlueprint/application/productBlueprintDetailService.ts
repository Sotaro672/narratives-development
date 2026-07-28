// frontend/console/shell/src/features/productBlueprint/application/productBlueprintDetailService.ts

import {
  isApparelCategoryCode,
  normalizeApparelMeasurementsForRequest,
  type ApparelCategoryCode,
  type ApparelMeasurements,
  type ApparelSizeInput,
} from "../domain/apparel";

import {
  isAlcoholCategoryCode,
} from "../domain/alcohol";

import {
  volumeRowToVolume,
  type AlcoholModelNumber,
  type Volume,
  type VolumeRow,
} from "../../model/application/modelCreateService";

import {
  updateProductBlueprintHTTP,
} from "../infrastructure/repository/productBlueprintRepositoryHTTP";

import {
  getProductBlueprintDetailApi,
  type ProductBlueprintDetailResponse,
  type UpdateProductBlueprintParams,
} from "../infrastructure/api/productBlueprintDetailApi";

import {
  hexToRgbInt,
} from "../../../shared/util/color";

import {
  createModelVariations,
  listModelVariationsByProductBlueprintId,
  type CreateModelVariationRequest,
} from "../../model/infrastructure/repository/modelRepositoryHTTP";

export {
  listModelVariationsByProductBlueprintId,
};

type ProductBlueprintModelNumber = {
  size: string;
  color: string;
  code?: string;
};

/* =========================================================
 * Category helpers
 * =======================================================*/

function resolveApparelCategoryCode(
  params: Pick<
    UpdateProductBlueprintParams,
    "productBlueprintCategory"
  >,
): ApparelCategoryCode | null {
  const code = String(
    params.productBlueprintCategory?.code ?? "",
  ).trim();

  if (!isApparelCategoryCode(code)) {
    return null;
  }

  return code;
}

function isAlcoholCategory(
  params: Pick<
    UpdateProductBlueprintParams,
    "productBlueprintCategory"
  >,
): boolean {
  const kind = String(
    params.productBlueprintCategory?.kind ?? "",
  ).trim();

  const code = String(
    params.productBlueprintCategory?.code ?? "",
  ).trim();

  return (
    kind === "alcohol" ||
    isAlcoholCategoryCode(code)
  );
}

function shouldCreateApparelModelVariations(
  categoryCode: ApparelCategoryCode,
): boolean {
  return (
    categoryCode === "apparel.tops" ||
    categoryCode === "apparel.bottoms" ||
    categoryCode === "apparel.dress" ||
    categoryCode === "apparel.outerwear" ||
    categoryCode === "apparel.shoes"
  );
}

/* =========================================================
 * Apparel helpers
 * =======================================================*/

function buildApparelMeasurements(
  categoryCode: ApparelCategoryCode,
  size: ApparelSizeInput,
): ApparelMeasurements {
  const result: ApparelMeasurements = {};

  switch (categoryCode) {
    case "apparel.bottoms": {
      result["ウエスト"] =
        size.waist ?? null;

      result["ヒップ"] =
        size.hip ?? null;

      result["股上"] =
        size.rise ?? null;

      result["股下"] =
        size.inseam ?? null;

      result["わたり幅"] =
        size.thigh ?? null;

      result["裾幅"] =
        size.hemWidth ?? null;

      return result;
    }

    case "apparel.dress": {
      result["着丈"] =
        size.length ?? null;

      result["身幅"] =
        size.width ?? null;

      result["胸囲"] =
        size.chest ?? null;

      result["肩幅"] =
        size.shoulder ?? null;

      result["袖丈"] =
        size.sleeveLength ?? null;

      result["ウエスト"] =
        size.waist ?? null;

      result["ヒップ"] =
        size.hip ?? null;

      return result;
    }

    case "apparel.tops": {
      result["着丈"] =
        size.length ?? null;

      result["身幅"] =
        size.width ?? null;

      result["胸囲"] =
        size.chest ?? null;

      result["肩幅"] =
        size.shoulder ?? null;

      result["袖丈"] =
        size.sleeveLength ?? null;

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

function buildMeasurementsFromSizeRow(
  categoryCode: ApparelCategoryCode,
  size: ApparelSizeInput,
): Record<string, number> | undefined {
  return normalizeApparelMeasurementsForRequest(
    buildApparelMeasurements(
      categoryCode,
      size,
    ),
  );
}

function resolveRgbInt(args: {
  colorName: string;
  colorRgbMap?: Record<string, string>;
}): number {
  const {
    colorName,
    colorRgbMap = {},
  } = args;

  const rgbHex = String(
    colorRgbMap[colorName] ?? "",
  ).trim();

  const rgb = rgbHex
    ? hexToRgbInt(rgbHex)
    : undefined;

  if (
    typeof rgb === "number" &&
    Number.isFinite(rgb)
  ) {
    return rgb;
  }

  throw new Error(
    `updateProductBlueprint: rgb が解決できません（color="${colorName}", hex="${rgbHex}"）`,
  );
}

function buildSizeMap(
  sizes: ApparelSizeInput[],
): Map<string, ApparelSizeInput> {
  const sizeMap =
    new Map<string, ApparelSizeInput>();

  for (const size of sizes) {
    const sizeLabel = String(
      size.sizeLabel ?? "",
    ).trim();

    if (!sizeLabel) {
      continue;
    }

    sizeMap.set(
      sizeLabel,
      size,
    );
  }

  return sizeMap;
}

function buildApparelModelNumberMap(
  modelNumbers: ProductBlueprintModelNumber[],
): Map<string, ProductBlueprintModelNumber> {
  const modelNumberMap =
    new Map<
      string,
      ProductBlueprintModelNumber
    >();

  for (const modelNumber of modelNumbers) {
    const size = String(
      modelNumber.size ?? "",
    ).trim();

    const color = String(
      modelNumber.color ?? "",
    ).trim();

    const code = String(
      modelNumber.code ?? "",
    ).trim();

    if (
      !size ||
      !color ||
      !code
    ) {
      continue;
    }

    modelNumberMap.set(
      `${size}__${color}`,
      {
        size,
        color,
        code,
      },
    );
  }

  return modelNumberMap;
}

function toCreateApparelModelVariationRequests(args: {
  apparelCategoryCode: ApparelCategoryCode;
  sizes: ApparelSizeInput[];
  modelNumbers: ProductBlueprintModelNumber[];
  colorRgbMap: Record<string, string>;
}): CreateModelVariationRequest[] {
  const {
    apparelCategoryCode,
    sizes,
    modelNumbers,
    colorRgbMap,
  } = args;

  if (
    !shouldCreateApparelModelVariations(
      apparelCategoryCode,
    )
  ) {
    return [];
  }

  const sizeMap =
    buildSizeMap(sizes);

  const modelNumberMap =
    buildApparelModelNumberMap(
      modelNumbers,
    );

  const requests:
    CreateModelVariationRequest[] = [];

  for (
    const modelNumber
    of modelNumberMap.values()
  ) {
    const sizeRow =
      sizeMap.get(
        modelNumber.size,
      );

    if (!sizeRow) {
      continue;
    }

    requests.push({
      kind: "apparel",

      modelNumber: String(
        modelNumber.code ?? "",
      ).trim(),

      size:
        modelNumber.size,

      color:
        modelNumber.color,

      rgb: resolveRgbInt({
        colorName:
          modelNumber.color,

        colorRgbMap,
      }),

      measurements:
        buildMeasurementsFromSizeRow(
          apparelCategoryCode,
          sizeRow,
        ),
    });
  }

  return requests;
}

/* =========================================================
 * Alcohol helpers
 * =======================================================*/

function normalizeVolume(
  volume: Volume,
): Volume | null {
  const value =
    typeof volume.value === "number" &&
    Number.isFinite(volume.value)
      ? volume.value
      : 0;

  const unit =
    String(
      volume.unit ?? "",
    ).trim() || "ml";

  if (
    value <= 0 ||
    (
      unit !== "ml" &&
      unit !== "L"
    )
  ) {
    return null;
  }

  return {
    value,
    unit,
  };
}

function makeVolumeKey(
  volume: Volume,
): string {
  return `${volume.value}:${volume.unit}`;
}

function buildAlcoholModelNumberMap(
  modelNumbers: AlcoholModelNumber[],
): Map<string, AlcoholModelNumber> {
  const modelNumberMap =
    new Map<
      string,
      AlcoholModelNumber
    >();

  for (const modelNumber of modelNumbers) {
    const volume =
      normalizeVolume(
        modelNumber.volume,
      );

    const code = String(
      modelNumber.code ?? "",
    ).trim();

    if (
      !volume ||
      !code
    ) {
      continue;
    }

    modelNumberMap.set(
      makeVolumeKey(volume),
      {
        ...modelNumber,
        volume,
        code,
      },
    );
  }

  return modelNumberMap;
}

function toCreateAlcoholModelVariationRequests(args: {
  volumes: VolumeRow[];
  alcoholModelNumbers: AlcoholModelNumber[];
}): CreateModelVariationRequest[] {
  const {
    volumes,
    alcoholModelNumbers,
  } = args;

  const modelNumberMap =
    buildAlcoholModelNumberMap(
      alcoholModelNumbers,
    );

  const requests:
    CreateModelVariationRequest[] = [];

  const seen =
    new Set<string>();

  for (const row of volumes) {
    const volume =
      normalizeVolume(
        volumeRowToVolume(row),
      );

    if (!volume) {
      continue;
    }

    const key =
      makeVolumeKey(volume);

    if (seen.has(key)) {
      continue;
    }

    seen.add(key);

    const modelNumber =
      modelNumberMap.get(key);

    if (!modelNumber) {
      continue;
    }

    requests.push({
      kind: "alcohol",

      modelNumber: String(
        modelNumber.code ?? "",
      ).trim(),

      volume,
    });
  }

  return requests;
}

/* =========================================================
 * Final Model Variation request builder
 * =======================================================*/

function buildFinalModelVariationRequests(args: {
  productBlueprintCategory:
    UpdateProductBlueprintParams["productBlueprintCategory"];

  sizes: ApparelSizeInput[];

  modelNumbers:
    ProductBlueprintModelNumber[];

  colorRgbMap:
    Record<string, string>;

  volumes:
    VolumeRow[];

  alcoholModelNumbers:
    AlcoholModelNumber[];
}): CreateModelVariationRequest[] | null {
  const {
    productBlueprintCategory,
    sizes,
    modelNumbers,
    colorRgbMap,
    volumes,
    alcoholModelNumbers,
  } = args;

  const apparelCategoryCode =
    resolveApparelCategoryCode({
      productBlueprintCategory,
    });

  if (apparelCategoryCode) {
    return toCreateApparelModelVariationRequests({
      apparelCategoryCode,
      sizes,
      modelNumbers,
      colorRgbMap,
    });
  }

  if (
    isAlcoholCategory({
      productBlueprintCategory,
    })
  ) {
    return toCreateAlcoholModelVariationRequests({
      volumes,
      alcoholModelNumbers,
    });
  }

  return null;
}

/* =========================================================
 * GET: Product Blueprint detail
 * =======================================================*/

export async function getProductBlueprintDetail(
  id: string,
): Promise<ProductBlueprintDetailResponse> {
  const normalizedId =
    String(id ?? "").trim();

  if (!normalizedId) {
    throw new Error(
      "getProductBlueprintDetail: id が空です",
    );
  }

  return getProductBlueprintDetailApi(
    normalizedId,
  );
}

/* =========================================================
 * UPDATE: Product Blueprint + Model Variations
 * =======================================================*/

export async function updateProductBlueprint(
  params: UpdateProductBlueprintParams & {
    sizes?: ApparelSizeInput[];

    modelNumbers?:
      ProductBlueprintModelNumber[];

    colorRgbMap?:
      Record<string, string>;

    volumes?:
      VolumeRow[];

    alcoholModelNumbers?:
      AlcoholModelNumber[];
  },
): Promise<ProductBlueprintDetailResponse> {
  const {
    id,
    productName,
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

  const productBlueprintId =
    String(id ?? "").trim();

  if (!productBlueprintId) {
    throw new Error(
      "updateProductBlueprint: id が空です",
    );
  }

  if (
    !productBlueprintCategoryId?.trim()
  ) {
    throw new Error(
      "updateProductBlueprint: productBlueprintCategoryId が空です",
    );
  }

  if (
    !productBlueprintCategory?.id?.trim()
  ) {
    throw new Error(
      "updateProductBlueprint: productBlueprintCategory が空です",
    );
  }

  const finalModelVariationRequests =
    buildFinalModelVariationRequests({
      productBlueprintCategory,
      sizes,
      modelNumbers,
      colorRgbMap,
      volumes,
      alcoholModelNumbers,
    });

  const updated =
    await updateProductBlueprintHTTP(
      productBlueprintId,
      {
        id:
          productBlueprintId,

        productName,

        brandId,

        productBlueprintCategoryId,

        productBlueprintCategory,

        categoryFields:
          categoryFields ?? null,

        productIdTagType,

        companyId,

        assigneeId,

        colors:
          colors ?? [],

        colorRgbMap:
          colorRgbMap ?? {},

        sizes,

        modelNumbers,

        updatedBy:
          updatedBy ?? null,
      } satisfies UpdateProductBlueprintParams,
    );

  /*
   * ApparelまたはAlcoholの場合だけModel Variationを一括置換する。
   *
   * 空配列も送信することで、Model Variationを0件にする更新にも対応する。
   * 個別更新・差分削除は行わない。
   */
  if (finalModelVariationRequests !== null) {
    await createModelVariations(
      productBlueprintId,
      finalModelVariationRequests,
    );
  }

  return updated;
}