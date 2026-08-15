// frontend/console/shell/src/features/productBlueprint/application/modelVariationRequestBuilder.ts

import {
  isApparelCategoryCode,
  isApparelModelVariationCategoryCode,
  mapApparelSizeInputToMeasurements,
  normalizeApparelMeasurementsForRequest,
  type ApparelCategoryCode,
  type ApparelModelNumberRow,
  type ApparelSizeInput,
} from "../../../shared/types/apparel";
import { hexToRgbInt } from "../../../shared/util/color";
import {
  volumeRowToVolume,
  type AlcoholModelNumber,
  type Volume,
  type VolumeRow,
} from "../../model/application/modelCreateService";
import type {
  CreateModelVariationRequest,
} from "../../model/infrastructure/modelRepositoryHTTP";
import { isAlcoholCategoryCode } from "../domain/alcohol";
import type {
  ProductBlueprintCategorySnapshot,
} from "../domain/productBlueprintCategory";

/* =========================================================
 * Public types
 * =======================================================*/

export type ProductBlueprintModelNumberInput =
  Omit<ApparelModelNumberRow, "code"> & {
    code?: string;
  };

export type MissingRgbBehavior = "use-zero" | "throw";

export type BuildModelVariationRequestsArgs = {
  productBlueprintCategory: ProductBlueprintCategorySnapshot;
  colors?: string[];
  sizes?: ApparelSizeInput[];
  modelNumbers?: ProductBlueprintModelNumberInput[];
  colorRgbMap?: Record<string, string>;
  volumes?: VolumeRow[];
  alcoholModelNumbers?: AlcoholModelNumber[];

  /**
   * colorRgbMapからRGBを解決できなかった場合の処理。
   *
   * use-zero:
   * - 既存の新規作成処理との互換性を維持し、0を使用する。
   *
   * throw:
   * - RGBが解決できない場合にエラーを返す。
   */
  missingRgbBehavior?: MissingRgbBehavior;
};

/* =========================================================
 * Common helpers
 * =======================================================*/

/**
 * 色、サイズ、型番など、ユーザーが入力できる値だけを正規化する。
 * seedから取得するcategory codeには使用しない。
 */
function normalizeUserText(
  value: string | null | undefined,
): string {
  return value?.trim() ?? "";
}

function makeApparelVariationKey(
  size: string,
  color: string,
): string {
  return JSON.stringify([size, color]);
}

/* =========================================================
 * Apparel measurement helpers
 * =======================================================*/

function buildMeasurementsFromSize(
  categoryCode: ApparelCategoryCode,
  size: ApparelSizeInput,
): Record<string, number> | undefined {
  return normalizeApparelMeasurementsForRequest(
    mapApparelSizeInputToMeasurements(
      size,
      categoryCode,
    ),
  );
}

/* =========================================================
 * Apparel input normalization
 * =======================================================*/

function normalizeColors(
  colors: string[],
): string[] {
  const normalizedColors: string[] = [];
  const seen = new Set<string>();

  for (const rawColor of colors) {
    const color = normalizeUserText(rawColor);

    if (
      color === "" ||
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
    new Map<string, ApparelSizeInput>();

  for (const size of sizes) {
    const sizeLabel =
      normalizeUserText(size.sizeLabel);

    if (sizeLabel === "") {
      continue;
    }

    /*
     * 同じsizeLabelが複数存在する場合は後勝ち。
     * Mapの並び順は最初に登録された位置を維持する。
     */
    sizeMap.set(sizeLabel, {
      ...size,
      sizeLabel,
    });
  }

  return Array.from(sizeMap.values());
}

function buildApparelModelNumberMap(
  modelNumbers: ProductBlueprintModelNumberInput[],
): Map<string, string> {
  const modelNumberMap =
    new Map<string, string>();

  for (const modelNumber of modelNumbers) {
    const size =
      normalizeUserText(modelNumber.size);
    const color =
      normalizeUserText(modelNumber.color);
    const code =
      normalizeUserText(modelNumber.code);

    if (
      size === "" ||
      color === "" ||
      code === ""
    ) {
      continue;
    }

    /*
     * 同じsize・colorの組み合わせが複数存在する場合は後勝ち。
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
    normalizeUserText(
      colorRgbMap[colorName],
    );

  const rgb =
    rgbHex === ""
      ? undefined
      : hexToRgbInt(rgbHex);

  if (
    typeof rgb === "number" &&
    Number.isFinite(rgb)
  ) {
    return rgb;
  }

  if (
    missingRgbBehavior === "use-zero"
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

function buildApparelModelVariationRequests(
  args: {
    categoryCode: ApparelCategoryCode;
    colors: string[];
    sizes: ApparelSizeInput[];
    modelNumbers: ProductBlueprintModelNumberInput[];
    colorRgbMap: Record<string, string>;
    missingRgbBehavior: MissingRgbBehavior;
  },
): CreateModelVariationRequest[] {
  const {
    categoryCode,
    colors,
    sizes,
    modelNumbers,
    colorRgbMap,
    missingRgbBehavior,
  } = args;

  if (
    !isApparelModelVariationCategoryCode(
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
        size.sizeLabel;

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
        size: sizeLabel,
        color,
        rgb: resolveRgbInt({
          colorName: color,
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
  if (
    typeof volume.value !== "number" ||
    !Number.isFinite(volume.value) ||
    volume.value <= 0
  ) {
    return null;
  }

  if (
    volume.unit !== "ml" &&
    volume.unit !== "L"
  ) {
    return null;
  }

  return {
    value: volume.value,
    unit: volume.unit,
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
    new Map<string, AlcoholModelNumber>();

  for (
    const modelNumber
    of modelNumbers
  ) {
    const volume =
      normalizeVolume(
        modelNumber.volume,
      );

    const code =
      normalizeUserText(
        modelNumber.code,
      );

    if (
      !volume ||
      code === ""
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

function buildAlcoholModelVariationRequests(
  args: {
    volumes: VolumeRow[];
    alcoholModelNumbers: AlcoholModelNumber[];
  },
): CreateModelVariationRequest[] {
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

  for (
    const row
    of volumes
  ) {
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
 * カテゴリコードはseed由来の値をそのまま使用し、
 * frontend側でtrimやkindからの推測は行わない。
 *
 * ApparelのうちModel Variation対象外のカテゴリでは
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

  const categoryCode =
    productBlueprintCategory.code;

  if (
    isApparelCategoryCode(
      categoryCode,
    )
  ) {
    return buildApparelModelVariationRequests({
      categoryCode,
      colors,
      sizes,
      modelNumbers,
      colorRgbMap,
      missingRgbBehavior,
    });
  }

  if (
    isAlcoholCategoryCode(
      categoryCode,
    )
  ) {
    return buildAlcoholModelVariationRequests({
      volumes,
      alcoholModelNumbers,
    });
  }

  return null;
}