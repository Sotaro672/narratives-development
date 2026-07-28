// frontend/console/shell/src/features/productBlueprint/application/modelVariationRequestBuilder.ts

import {
  isApparelCategoryCode,
  normalizeApparelMeasurementsForRequest,
  type ApparelCategoryCode,
  type ApparelMeasurements,
  type ApparelSizeInput,
} from "../../../shared/types/apparel";

import {
  hexToRgbInt,
} from "../../../shared/util/color";

import {
  volumeRowToVolume,
  type AlcoholModelNumber,
  type Volume,
  type VolumeRow,
} from "../../model/application/modelCreateService";

import type {
  CreateModelVariationRequest,
} from "../../model/infrastructure/repository/modelRepositoryHTTP";

import {
  isAlcoholCategoryCode,
} from "../domain/alcohol";

import type {
  ProductBlueprintCategorySnapshot,
} from "../domain/productBlueprintCategory";

/* =========================================================
 * Public types
 * =======================================================*/

export type ProductBlueprintModelNumberInput = {
  size: string;
  color: string;
  code?: string;
};

export type MissingRgbBehavior =
  | "use-zero"
  | "throw";

export type BuildModelVariationRequestsArgs = {
  productBlueprintCategory:
    ProductBlueprintCategorySnapshot;

  colors?: string[];

  sizes?: ApparelSizeInput[];

  modelNumbers?:
    ProductBlueprintModelNumberInput[];

  colorRgbMap?:
    Record<string, string>;

  volumes?: VolumeRow[];

  alcoholModelNumbers?:
    AlcoholModelNumber[];

  /**
   * colorRgbMapからRGBを解決できなかった場合の処理。
   *
   * use-zero:
   * - 従来の新規作成処理と同じく0を使用する。
   *
   * throw:
   * - 従来の更新処理と同じくエラーにする。
   */
  missingRgbBehavior?:
    MissingRgbBehavior;
};

/* =========================================================
 * Common helpers
 * =======================================================*/

function normalizeString(
  value: unknown,
): string {
  return String(value ?? "").trim();
}

function makeApparelVariationKey(
  size: string,
  color: string,
): string {
  return `${size}__${color}`;
}

/* =========================================================
 * Category helpers
 * =======================================================*/

function resolveApparelCategoryCode(
  category: ProductBlueprintCategorySnapshot,
): ApparelCategoryCode | null {
  const code =
    normalizeString(category.code);

  if (!isApparelCategoryCode(code)) {
    return null;
  }

  return code;
}

function isAlcoholCategory(
  category: ProductBlueprintCategorySnapshot,
): boolean {
  const kind =
    normalizeString(category.kind);

  const code =
    normalizeString(category.code);

  return (
    kind === "alcohol" ||
    isAlcoholCategoryCode(code)
  );
}

function shouldBuildApparelModelVariations(
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
 * Apparel measurement helpers
 * =======================================================*/

function buildApparelMeasurements(
  categoryCode: ApparelCategoryCode,
  size: ApparelSizeInput,
): ApparelMeasurements {
  const result:
    ApparelMeasurements = {};

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

function buildMeasurementsFromSize(
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

/* =========================================================
 * Apparel input normalization
 * =======================================================*/

function normalizeColors(
  colors: string[],
): string[] {
  const normalizedColors:
    string[] = [];

  const seen =
    new Set<string>();

  for (const rawColor of colors) {
    const color =
      normalizeString(rawColor);

    if (
      !color ||
      seen.has(color)
    ) {
      continue;
    }

    seen.add(color);
    normalizedColors.push(color);
  }

  return normalizedColors;
}

function normalizeSizes(
  sizes: ApparelSizeInput[],
): ApparelSizeInput[] {
  const sizeMap =
    new Map<
      string,
      ApparelSizeInput
    >();

  for (const size of sizes) {
    const sizeLabel =
      normalizeString(
        size.sizeLabel,
      );

    if (!sizeLabel) {
      continue;
    }

    /*
     * 同じsizeLabelが複数存在する場合は後勝ち。
     * Mapの挿入順は最初に登録された順序を維持する。
     */
    sizeMap.set(
      sizeLabel,
      size,
    );
  }

  return Array.from(
    sizeMap.values(),
  );
}

function buildApparelModelNumberMap(
  modelNumbers:
    ProductBlueprintModelNumberInput[],
): Map<string, string> {
  const modelNumberMap =
    new Map<string, string>();

  for (const modelNumber of modelNumbers) {
    const size =
      normalizeString(
        modelNumber.size,
      );

    const color =
      normalizeString(
        modelNumber.color,
      );

    const code =
      normalizeString(
        modelNumber.code,
      );

    if (
      !size ||
      !color ||
      !code
    ) {
      continue;
    }

    /*
     * 同じsize・colorの組み合わせが複数ある場合は後勝ち。
     */
    modelNumberMap.set(
      makeApparelVariationKey(
        size,
        color,
      ),
      code,
    );
  }

  return modelNumberMap;
}

function resolveRgbInt(args: {
  colorName: string;
  colorRgbMap: Record<string, string>;
  missingRgbBehavior: MissingRgbBehavior;
}): number {
  const {
    colorName,
    colorRgbMap,
    missingRgbBehavior,
  } = args;

  const rgbHex =
    normalizeString(
      colorRgbMap[colorName],
    );

  const rgb =
    rgbHex
      ? hexToRgbInt(rgbHex)
      : undefined;

  if (
    typeof rgb === "number" &&
    Number.isFinite(rgb)
  ) {
    return rgb;
  }

  if (
    missingRgbBehavior ===
    "use-zero"
  ) {
    return 0;
  }

  throw new Error(
    `modelVariationRequestBuilder: rgb が解決できません（color="${colorName}", hex="${rgbHex}"）`,
  );
}

/* =========================================================
 * Apparel request builder
 * =======================================================*/

function buildApparelModelVariationRequests(args: {
  categoryCode: ApparelCategoryCode;
  colors: string[];
  sizes: ApparelSizeInput[];
  modelNumbers:
    ProductBlueprintModelNumberInput[];
  colorRgbMap:
    Record<string, string>;
  missingRgbBehavior:
    MissingRgbBehavior;
}): CreateModelVariationRequest[] {
  const {
    categoryCode,
    colors,
    sizes,
    modelNumbers,
    colorRgbMap,
    missingRgbBehavior,
  } = args;

  if (
    !shouldBuildApparelModelVariations(
      categoryCode,
    )
  ) {
    return [];
  }

  const normalizedColors =
    normalizeColors(colors);

  const normalizedSizes =
    normalizeSizes(sizes);

  const modelNumberMap =
    buildApparelModelNumberMap(
      modelNumbers,
    );

  const requests:
    CreateModelVariationRequest[] = [];

  /*
   * Model Variationの並び順を
   * 「色登録順 → サイズ登録順」に固定する。
   */
  for (
    const color
    of normalizedColors
  ) {
    for (
      const size
      of normalizedSizes
    ) {
      const sizeLabel =
        normalizeString(
          size.sizeLabel,
        );

      if (!sizeLabel) {
        continue;
      }

      const modelNumber =
        modelNumberMap.get(
          makeApparelVariationKey(
            sizeLabel,
            color,
          ),
        );

      if (!modelNumber) {
        continue;
      }

      requests.push({
        kind: "apparel",

        modelNumber,

        size:
          sizeLabel,

        color,

        rgb:
          resolveRgbInt({
            colorName:
              color,

            colorRgbMap,

            missingRgbBehavior,
          }),

        measurements:
          buildMeasurementsFromSize(
            categoryCode,
            size,
          ),
      });
    }
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
    normalizeString(
      volume.unit,
    ) || "ml";

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
  modelNumbers:
    AlcoholModelNumber[],
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

    const code =
      normalizeString(
        modelNumber.code,
      );

    if (
      !volume ||
      !code
    ) {
      continue;
    }

    /*
     * 同じ容量が複数存在する場合は後勝ち。
     */
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

/* =========================================================
 * Alcohol request builder
 * =======================================================*/

function buildAlcoholModelVariationRequests(args: {
  volumes: VolumeRow[];
  alcoholModelNumbers:
    AlcoholModelNumber[];
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

      modelNumber:
        modelNumber.code,

      volume,
    });
  }

  return requests;
}

/* =========================================================
 * Public builder
 * =======================================================*/

/**
 * ProductBlueprintのカテゴリと入力値から、
 * Model Variation一括置換APIへ渡すrequestを生成する。
 *
 * 戻り値:
 * - Apparel: CreateModelVariationRequest[]
 * - Alcohol: CreateModelVariationRequest[]
 * - Model Variationを扱わないカテゴリ: null
 *
 * Apparelのうち、Model Variation対象外のカテゴリでは
 * 空配列を返す。
 */
export function buildModelVariationRequests(
  args: BuildModelVariationRequestsArgs,
): CreateModelVariationRequest[] | null {
  const {
    productBlueprintCategory,
    colors = [],
    sizes = [],
    modelNumbers = [],
    colorRgbMap = {},
    volumes = [],
    alcoholModelNumbers = [],
    missingRgbBehavior = "throw",
  } = args;

  const apparelCategoryCode =
    resolveApparelCategoryCode(
      productBlueprintCategory,
    );

  if (apparelCategoryCode) {
    return buildApparelModelVariationRequests({
      categoryCode:
        apparelCategoryCode,

      colors,

      sizes,

      modelNumbers,

      colorRgbMap,

      missingRgbBehavior,
    });
  }

  if (
    isAlcoholCategory(
      productBlueprintCategory,
    )
  ) {
    return buildAlcoholModelVariationRequests({
      volumes,
      alcoholModelNumbers,
    });
  }

  return null;
}