// frontend/console/shell/src/features/productBlueprint/application/productBlueprintDetailService.ts

import type {
  ApparelSizeInput,
} from "../../../shared/types/apparel";

import type {
  AlcoholModelNumber,
  VolumeRow,
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
  createModelVariations,
  listModelVariationsByProductBlueprintId,
} from "../../model/infrastructure/repository/modelRepositoryHTTP";

import {
  buildModelVariationRequests,
  type ProductBlueprintModelNumberInput,
} from "./modelVariationRequestBuilder";

export {
  listModelVariationsByProductBlueprintId,
};

type ProductBlueprintModelNumber =
  ProductBlueprintModelNumberInput;

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
    !productBlueprintCategoryId
      ?.trim()
  ) {
    throw new Error(
      "updateProductBlueprint: productBlueprintCategoryId が空です",
    );
  }

  if (
    !productBlueprintCategory
      ?.id
      ?.trim()
  ) {
    throw new Error(
      "updateProductBlueprint: productBlueprintCategory が空です",
    );
  }

  const finalModelVariationRequests =
    buildModelVariationRequests({
      productBlueprintCategory,

      colors:
        colors ?? [],

      sizes,

      modelNumbers,

      colorRgbMap,

      volumes,

      alcoholModelNumbers,

      /*
       * 更新処理では、従来どおりRGBを解決できない場合に
       * エラーとする。
       */
      missingRgbBehavior:
        "throw",
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

        colorRgbMap,

        sizes,

        modelNumbers,

        updatedBy:
          updatedBy ?? null,
      } satisfies UpdateProductBlueprintParams,
    );

  /*
   * ApparelまたはAlcoholの場合だけ
   * Model Variationを一括置換する。
   *
   * 空配列も送信することで、
   * Model Variationを0件にする更新にも対応する。
   *
   * nullの場合はModel Variationを扱わないカテゴリなので
   * createModelVariationsを実行しない。
   */
  if (
    finalModelVariationRequests !==
    null
  ) {
    await createModelVariations(
      productBlueprintId,
      finalModelVariationRequests,
    );
  }

  return updated;
}