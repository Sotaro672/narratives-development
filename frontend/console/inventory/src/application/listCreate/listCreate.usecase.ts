// frontend/console/inventory/src/application/listCreate/listCreate.usecase.ts

import { getListCreateRaw } from "../../infrastructure/api/listCreateApi";
import type {
  ListCreateDTO,
  ListCreatePriceRowDTO,
} from "../../infrastructure/http/listCreateRepositoryHTTP.types";
import { mapListCreateDTO } from "../../infrastructure/http/listCreateRepositoryHTTP.mappers";

// Firebase Auth（uid 取得）
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

// model repository
import {
  listModelVariationsByProductBlueprintId,
  type ModelVariationResponse,
} from "../../../../model/src/infrastructure/repository/modelRepositoryHTTP";

// list create (POST /lists)
import {
  createListHTTP,
  type CreateListInput,
  type ListDTO,
} from "../../../../list/src/infrastructure/http/list";

import type { ResolvedListCreateParams } from "./listCreate.types";
import { buildListCreateFetchInput } from "./listCreate.routing";
import {
  buildCreateListInput,
  validateCreateListInput,
} from "./listCreate.input";
import {
  uploadListImagesPolicyB,
  _internal_getListIdFromListDTO,
} from "./listCreate.images";

function buildModelVariationById(
  variations: ModelVariationResponse[],
): Record<string, ModelVariationResponse> {
  const out: Record<string, ModelVariationResponse> = {};

  for (const variation of variations) {
    if (!variation.id) continue;
    out[variation.id] = variation;
  }

  return out;
}

function getVariationKind(
  variation: ModelVariationResponse | undefined,
): "apparel" | "alcohol" | null {
  if (!variation) return null;

  if (variation.kind === "apparel") return "apparel";
  if (variation.kind === "alcohol") return "alcohol";

  return null;
}

function getProductBlueprintCategoryKindFromVariations(
  variations: ModelVariationResponse[],
): "apparel" | "alcohol" | null {
  const hasAlcohol = variations.some((variation) => variation.kind === "alcohol");
  if (hasAlcohol) return "alcohol";

  const hasApparel = variations.some((variation) => variation.kind === "apparel");
  if (hasApparel) return "apparel";

  return null;
}

function mergePriceRowWithModelVariation(args: {
  row: ListCreatePriceRowDTO;
  variation?: ModelVariationResponse;
}): ListCreatePriceRowDTO {
  const { row, variation } = args;

  if (!variation) {
    return row;
  }

  if (variation.kind === "alcohol") {
    return {
      ...row,
      kind: "alcohol",

      // alcohol では PriceCard 側で size/color ではなく volumeValue/volumeUnit を表示する
      volumeValue: variation.volume.value,
      volumeUnit: variation.volume.unit,
    };
  }

  if (variation.kind === "apparel") {
    return {
      ...row,
      kind: "apparel",

      // apparel では PriceCard 側で size/color/rgb を表示する
      size: variation.size,
      color: variation.color.name,
      rgb: variation.color.rgb,
    };
  }

  return row;
}

function mergeListCreateDTOWithModelVariations(args: {
  dto: ListCreateDTO;
  modelVariations: ModelVariationResponse[];
}): ListCreateDTO {
  const { dto, modelVariations } = args;

  if (!Array.isArray(dto.priceRows) || dto.priceRows.length === 0) {
    return dto;
  }

  const variationById = buildModelVariationById(modelVariations);

  const priceRows = dto.priceRows.map((row) =>
    mergePriceRowWithModelVariation({
      row,
      variation: variationById[row.modelId],
    }),
  );

  const inferredCategoryKind =
    dto.productBlueprintCategoryKind ??
    getProductBlueprintCategoryKindFromVariations(modelVariations);

  const inferredCategoryCode =
    dto.productBlueprintCategory ??
    (inferredCategoryKind ? inferredCategoryKind : null);

  return {
    ...dto,
    productBlueprintCategory: inferredCategoryCode,
    productBlueprintCategoryKind: inferredCategoryKind,
    priceRows,
  };
}

/**
 * ListCreateDTO を取得する（Hook からはこれだけ呼ぶ）
 *
 * - getListCreateRaw の raw response は mapListCreateDTO で ListCreateDTO へ変換する
 * - priceRows の modelId を正として、model repository から variation を取得する
 * - model variation の kind に応じて PriceCard 表示用の row に合成する
 *
 * category ごとの表示:
 * - apparel: size / color / rgb
 * - alcohol: volumeValue / volumeUnit
 */
export async function loadListCreateDTOFromParams(
  p: ResolvedListCreateParams,
): Promise<ListCreateDTO> {
  const input = buildListCreateFetchInput(p);

  const raw = await getListCreateRaw(input);
  const dto = mapListCreateDTO(raw);

  const productBlueprintId = dto.productBlueprintId;

  if (!productBlueprintId) {
    return dto;
  }

  let modelVariations: ModelVariationResponse[] = [];

  try {
    modelVariations =
      await listModelVariationsByProductBlueprintId(productBlueprintId);
  } catch {
    modelVariations = [];
  }

  return mergeListCreateDTOWithModelVariations({
    dto,
    modelVariations,
  });
}

/**
 * list 作成（POST /lists） + 画像（Policy B）
 */
export async function createListWithImages(args: {
  params: ResolvedListCreateParams;
  listingTitle: string;
  description: string;
  priceRows: any[];
  decision: "list" | "hold";
  assigneeId?: string;

  images?: File[];
  mainImageIndex?: number;
}): Promise<ListDTO> {
  const images = Array.isArray(args.images) ? args.images : [];
  const mainImageIndex = Number.isFinite(Number(args.mainImageIndex))
    ? Number(args.mainImageIndex)
    : 0;

  // 1) build + validate
  const input: CreateListInput = buildCreateListInput({
    params: args.params, // inventoryId(pb__tb) を保持
    listingTitle: args.listingTitle,
    description: args.description,
    priceRows: args.priceRows,
    decision: args.decision,
    assigneeId: args.assigneeId,
  });

  validateCreateListInput(input);

  // 2) create list
  const created = await createListHTTP(input);

  const listId = _internal_getListIdFromListDTO(
    created,
    (input as any).id || (input as any).inventoryId,
  );

  if (!listId) {
    throw new Error("created_list_missing_id");
  }

  // 3) images (Policy B)
  if (images.length > 0) {
    await uploadListImagesPolicyB({
      listId,
      files: images,
      mainImageIndex,
      createdBy: auth.currentUser?.uid,
    });
  }

  return created;
}