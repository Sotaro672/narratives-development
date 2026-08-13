// frontend/console/shell/src/features/productBlueprint/application/productBlueprintCreateService.ts

import type { ApparelSizeInput } from "../../../shared/types/apparel";
import type {
  AlcoholModelNumber,
  VolumeRow,
} from "../../model/application/modelCreateService";
import type {
  CategoryFieldValues,
  ProductBlueprintCategorySnapshot,
} from "../domain/productBlueprintCategory";
import type { ProductBlueprintDetailResponse } from "../infrastructure/api/productBlueprintDetailApi";
import { createProductBlueprintHTTP } from "../infrastructure/repository/productBlueprintRepositoryHTTP";
import {
  createModelVariations,
  type CreateModelVariationRequest,
} from "../../model/infrastructure/repository/modelRepositoryHTTP";
import {
  buildModelVariationRequests,
  type ProductBlueprintModelNumberInput,
} from "./modelVariationRequestBuilder";

// ------------------------------------------------------
// Product ID Tag
// ------------------------------------------------------

export type ProductIDTagType = "qr" | "nfc";

export type ProductIDTag = {
  type: ProductIDTagType;
};

// ------------------------------------------------------
// 作成用型
// ------------------------------------------------------

export type CreateProductBlueprintParams = {
  productName: string;
  brandId: string;
  productBlueprintCategoryId: string;
  productBlueprintCategory: ProductBlueprintCategorySnapshot;
  fit?: string | null;
  material?: string | null;
  weight?: number | null;
  qualityAssurance?: string[] | null;
  productIdTag: ProductIDTag;
  companyId: string;
  assigneeId?: string;
  createdBy?: string;
  colors?: string[];
  sizes?: ApparelSizeInput[];
  modelNumbers?: ProductBlueprintModelNumberInput[];
  colorRgbMap?: Record<string, string>;

  /**
   * alcohol model variation用。
   * volumeはProductBlueprint.categoryFieldsではなく
   * model domain側で扱う。
   */
  volumes?: VolumeRow[];
  alcoholModelNumbers?: AlcoholModelNumber[];
  categoryFields?: CategoryFieldValues | null;
};

// ------------------------------------------------------
// Validation helpers
// ------------------------------------------------------

function assertProductBlueprintCategory(
  params: CreateProductBlueprintParams,
): void {
  if (!params.productBlueprintCategoryId?.trim()) {
    throw new Error(
      "createProductBlueprint: productBlueprintCategoryId が空です",
    );
  }

  if (!params.productBlueprintCategory?.id?.trim()) {
    throw new Error(
      "createProductBlueprint: productBlueprintCategory.id が空です",
    );
  }

  if (
    params.productBlueprintCategoryId !== params.productBlueprintCategory.id
  ) {
    throw new Error(
      "createProductBlueprint: productBlueprintCategoryId と productBlueprintCategory.id が一致しません",
    );
  }
}

function extractProductBlueprintId(
  json: ProductBlueprintDetailResponse,
): string {
  return typeof json.id === "string" ? json.id : "";
}

// ------------------------------------------------------
// ProductBlueprint creation
// ------------------------------------------------------

async function createProductBlueprintWithModelRequests(
  params: CreateProductBlueprintParams,
  requests: CreateModelVariationRequest[],
): Promise<ProductBlueprintDetailResponse> {
  assertProductBlueprintCategory(params);

  const created = await createProductBlueprintHTTP(params);
  const productBlueprintId = extractProductBlueprintId(created);

  if (!productBlueprintId) {
    throw new Error("createProductBlueprint: 作成後の id が空です");
  }

  if (requests.length === 0) {
    return created;
  }

  await createModelVariations(productBlueprintId, requests);

  return created;
}

// ------------------------------------------------------
// Service本体
// ------------------------------------------------------

export async function createProductBlueprint(
  params: CreateProductBlueprintParams,
): Promise<ProductBlueprintDetailResponse> {
  const requests =
    buildModelVariationRequests({
      productBlueprintCategory: params.productBlueprintCategory,
      colors: params.colors ?? [],
      sizes: params.sizes ?? [],
      modelNumbers: params.modelNumbers ?? [],
      colorRgbMap: params.colorRgbMap ?? {},
      volumes: params.volumes ?? [],
      alcoholModelNumbers: params.alcoholModelNumbers ?? [],

      /*
       * 新規作成では、従来どおりRGBを解決できない場合に
       * 0を使用する。
       */
      missingRgbBehavior: "use-zero",
    }) ?? [];

  return createProductBlueprintWithModelRequests(params, requests);
}