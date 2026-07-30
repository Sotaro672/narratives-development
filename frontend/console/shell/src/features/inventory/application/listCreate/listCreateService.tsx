// frontend/console/shell/src/features/inventory/application/listCreate/listCreateService.tsx

import type * as React from "react";

import { getListCreateRaw } from "../../infrastructure/api/listCreateApi";
import type { ListCreateDTO } from "../../infrastructure/http/listCreateRepositoryHTTP.types";
import { mapListCreateDTO } from "../../infrastructure/http/listCreateRepositoryHTTP.mappers";

import type {
  List,
  ListPriceRow,
  ListStatus,
} from "../../../../shared/types/list";

import {
  createListHTTP,
  saveListImageFromFirebaseStorageHTTP,
  setListPrimaryImageHTTP,
} from "../../../list/infrastructure/repository";

import type {
  CreateListInput as ListPostCreateListInput,
} from "../../../list/infrastructure/dto";

import { uploadListImageToFirebaseStorage } from "../../../list/infrastructure/firebase/listImageStorage";

/**
 * List create route params
 *
 * UIルートはinventoryId（= inventoryKey: "pb__tb"）のみを正とする。
 * productBlueprintId / tokenBlueprintIdは互換用途では扱わない。
 */
export type ListCreateRouteParams = {
  inventoryId?: string;
};

export type ResolvedListCreateParams = {
  inventoryId: string;
  raw: ListCreateRouteParams;
};

export type PriceCardMode =
  | "view"
  | "edit";

export type PriceRowKind =
  | "apparel"
  | "alcohol"
  | string;

/**
 * PriceCard用row。
 *
 * backend responseを正とする。
 *
 * - modelIdを識別子として使う
 * - React keyはdisplayOrderではなくmodelIdを使う
 * - displayOrderは重複または未設定があり得る
 * - 並び順はdisplayOrder昇順のみ
 *
 * categoryごとの表示:
 * - apparel: size / color / rgb
 * - alcohol: volumeValue / volumeUnit
 */
export type PriceRow = {
  modelId: string;

  /**
   * モデル種別。
   */
  kind?: PriceRowKind | null;

  /**
   * 並び順。
   */
  displayOrder?: number | null;

  /**
   * apparel category用。
   */
  size?: string | null;

  /**
   * apparel category用。
   */
  color?: string | null;

  /**
   * RGB。
   *
   * backend responseではnumberが基本。
   * 既存UI互換として"#RRGGBB"形式のstringも許容する。
   */
  rgb?: number | string | null;

  /**
   * alcohol category用。
   */
  volumeValue?: number | null;

  /**
   * alcohol category用。
   */
  volumeUnit?: string | null;

  stock: number;

  /**
   * 価格。
   *
   * 入力中の空欄はundefinedで保持する。
   * nullは使用しない。
   */
  price?: number;
};

export type PriceCardProps = {
  title?: string;
  rows: PriceRow[];
  className?: string;

  mode?: PriceCardMode;

  /**
   * ProductBlueprintCategory.codeを渡す想定。
   *
   * 例:
   * - "apparel.tops"
   * - "alcohol.sake"
   */
  productBlueprintCategory?: string;

  /**
   * edit時に価格を更新するコールバック。
   *
   * 空欄の場合はundefinedを渡す。
   */
  onChangePrice?: (
    index: number,
    price: number | undefined,
    row: PriceRow,
  ) => void;

  currencySymbol?: string;
};

export type PriceRowVM = {
  /**
   * React key用の識別子。
   */
  modelId: string;

  /**
   * モデル種別。
   */
  kind?: PriceRowKind | null;

  /**
   * 並び順。
   */
  displayOrder: number | null;

  /**
   * apparel category用。
   */
  size?: string | null;

  /**
   * apparel category用。
   */
  color?: string | null;

  /**
   * alcohol category用。
   */
  volumeValue?: number | null;

  /**
   * alcohol category用。
   */
  volumeUnit?: string | null;

  stock: number;

  bgColor: string;
  rgbTitle: string;

  priceInputValue: string;
  priceDisplayText: string;

  onChangePriceInput: (
    event: React.ChangeEvent<HTMLInputElement>,
  ) => void;
};

export type UsePriceCardResult = {
  title: string;
  mode: PriceCardMode;
  isEdit: boolean;
  showModeBadge: boolean;

  currencySymbol: string;

  rowsVM: PriceRowVM[];
  isEmpty: boolean;
};

type CompletedPriceRow =
  PriceRow & {
    price: number;
  };

export type CreateListInput = {
  inventoryId: string;
  title: string;
  description: string;
  status: ListStatus;
  assigneeId?: string;
  priceRows: ListPriceRow[];
};

export const IMAGE_REQUIRED_MESSAGE =
  "画像が必須です。";

export const PRICE_REQUIRED_MESSAGE =
  "価格が未入力の商品があります。";

export const LIST_IMAGE_UPLOAD_FAILED_MESSAGE =
  "画像アップロードに失敗しました。後から追加できます。";

/**
 * - UIルートはinventoryId（= inventoryKey: "pb__tb"）のみを正とする
 * - backend fetchもinventoryIdのみを使う
 * - productBlueprintId / tokenBlueprintIdは一切扱わない
 */
export function resolveListCreateParams(
  raw: ListCreateRouteParams,
): ResolvedListCreateParams {
  return {
    inventoryId: raw.inventoryId,
    raw,
  } as ResolvedListCreateParams;
}

export function canFetchListCreate(
  params: ResolvedListCreateParams,
): boolean {
  return Boolean(
    params.inventoryId,
  );
}

export function buildListCreateFetchInput(
  params: ResolvedListCreateParams,
): {
  inventoryId?: string;
} {
  if (!params.inventoryId) {
    return {
      inventoryId: undefined,
    };
  }

  return {
    inventoryId: params.inventoryId,
  };
}

export function buildInventoryDetailPath(
  inventoryId: string,
): string {
  if (!inventoryId) {
    return "/inventory";
  }

  return `/inventory/detail/${encodeURIComponent(
    inventoryId,
  )}`;
}

export function buildBackPath(
  params: ResolvedListCreateParams,
): string {
  if (params.inventoryId) {
    return buildInventoryDetailPath(
      params.inventoryId,
    );
  }

  return "/inventory";
}

export function buildAfterCreatePath(
  params: ResolvedListCreateParams,
): string {
  if (params.inventoryId) {
    return buildInventoryDetailPath(
      params.inventoryId,
    );
  }

  return "/inventory";
}

export function extractDisplayStrings(
  dto: ListCreateDTO | null,
): {
  productBrandName: string;
  productName: string;
  tokenBrandName: string;
  tokenName: string;
} {
  return {
    productBrandName:
      dto?.productBrandName ?? "",

    productName:
      dto?.productName ?? "",

    tokenBrandName:
      dto?.tokenBrandName ?? "",

    tokenName:
      dto?.tokenName ?? "",
  };
}

function assertCompletedPriceRows(
  rows: PriceRow[],
): asserts rows is CompletedPriceRow[] {
  const hasMissingPrice =
    rows.length === 0 ||
    rows.some((row) => {
      return (
        typeof row.price !== "number" ||
        !Number.isFinite(row.price)
      );
    });

  if (hasMissingPrice) {
    throw new Error(
      PRICE_REQUIRED_MESSAGE,
    );
  }
}

export function buildCreateListInput(
  args: {
    params: ResolvedListCreateParams;
    listingTitle: string;
    description: string;
    priceRows: CompletedPriceRow[];
    status: ListStatus;
    assigneeId?: string;
  },
): CreateListInput {
  return {
    inventoryId:
      args.params.inventoryId,

    title:
      args.listingTitle,

    description:
      args.description,

    status:
      args.status,

    assigneeId:
      args.assigneeId,

    priceRows:
      args.priceRows.map(
        (row): ListPriceRow => ({
          modelId:
            row.modelId,

          price:
            row.price,
        }),
      ),
  };
}

export function validateCreateListInput(
  input: CreateListInput,
): void {
  if (!input.title) {
    throw new Error(
      "タイトルを入力してください。",
    );
  }

  if (input.priceRows.length === 0) {
    throw new Error(
      PRICE_REQUIRED_MESSAGE,
    );
  }

  const missingModelId =
    input.priceRows.some((row) => {
      return !row.modelId;
    });

  if (missingModelId) {
    throw new Error(
      "価格行に modelId が含まれていません。",
    );
  }

  const missingPrice =
    input.priceRows.some((row) => {
      return (
        typeof row.price !== "number" ||
        !Number.isFinite(row.price)
      );
    });

  if (missingPrice) {
    throw new Error(
      PRICE_REQUIRED_MESSAGE,
    );
  }
}

/**
 * 複数画像をFirebase Storageへ直接アップロード
 * → backendに画像レコード登録
 * → primary image設定
 *
 * Policy B:
 * - List作成後のlistIdを使ってFirebase Storageへupload
 * - Firebase Storage download URLを取得
 * - saveListImageFromFirebaseStorageHTTPでimage recordを登録
 *
 * primary:
 * - backendのList.imageIdはimages subcollectionのdocID
 * - objectPathではなくimageIdを渡す
 */
export async function uploadListImagesPolicyB(
  args: {
    listId: string;
    files: File[];
    mainImageIndex: number;
  },
): Promise<{
  registered: Array<{
    imageId: string;
    displayOrder: number;
  }>;
  primaryImageId?: string;
}> {
  const listId =
    String(
      args.listId ?? "",
    ).trim();

  if (!listId) {
    throw new Error(
      "invalid_list_id",
    );
  }

  if (args.files.length === 0) {
    throw new Error(
      IMAGE_REQUIRED_MESSAGE,
    );
  }

  const mainImageIndex =
    args.mainImageIndex >= 0 &&
    args.mainImageIndex <
      args.files.length
      ? args.mainImageIndex
      : 0;

  const registered: Array<{
    imageId: string;
    displayOrder: number;
  }> = [];

  for (
    let index = 0;
    index < args.files.length;
    index += 1
  ) {
    const file =
      args.files[index];

    if (!file) {
      continue;
    }

    const uploaded =
      await uploadListImageToFirebaseStorage({
        listId,
        file,
      });

    await saveListImageFromFirebaseStorageHTTP({
      listId,

      id:
        uploaded.imageId,

      url:
        uploaded.url,

      displayOrder:
        index,
    });

    registered.push({
      imageId:
        uploaded.imageId,

      displayOrder:
        index,
    });
  }

  const primary =
    registered.find(
      (item) =>
        item.displayOrder ===
        mainImageIndex,
    ) ??
    registered[0];

  if (primary?.imageId) {
    await setListPrimaryImageHTTP({
      listId,

      imageId:
        primary.imageId,
    });
  }

  return {
    registered,

    primaryImageId:
      primary?.imageId,
  };
}

export function _internal_getListIdFromListDTO(
  list: List,
): string {
  return list.id;
}

/**
 * ListCreateDTOを取得する。
 *
 * 方針:
 * - GET /inventory/list-create/{inventoryId}のresponseを唯一の正とする
 * - frontendではmodel variations APIを呼ばない
 * - priceRowsはbackend側で完成形になっている前提とする
 *
 * categoryごとの表示:
 * - apparel: modelNumber / size / color / rgb
 * - alcohol: modelNumber / volumeValue / volumeUnit
 */
export async function loadListCreateDTOFromParams(
  params: ResolvedListCreateParams,
): Promise<ListCreateDTO> {
  const input =
    buildListCreateFetchInput(
      params,
    );

  const raw =
    await getListCreateRaw(
      input,
    );

  return mapListCreateDTO(
    raw,
  );
}

/**
 * list作成（POST /lists）と画像登録。
 *
 * Policy B:
 * 1. 必須項目を検証する
 * 2. 画像なしでListを先に作成する
 * 3. 作成済みlistIdを使ってFirebase Storageへuploadする
 * 4. backendにimage recordを作成する
 * 5. primary imageを設定する
 *
 * List作成後に画像登録が失敗しても、
 * List作成自体は成功として扱う。
 */
export async function createListWithImages(
  args: {
    params: ResolvedListCreateParams;
    listingTitle: string;
    description: string;
    priceRows: PriceRow[];
    status: ListStatus;
    assigneeId?: string;

    images: File[];
    mainImageIndex: number;

    onImageUploadFailed?: (
      message: string,
      error: unknown,
    ) => void;
  },
): Promise<List> {
  if (!args.listingTitle) {
    throw new Error(
      "タイトルを入力してください。",
    );
  }

  if (args.images.length === 0) {
    throw new Error(
      IMAGE_REQUIRED_MESSAGE,
    );
  }

  assertCompletedPriceRows(
    args.priceRows,
  );

  const input =
    buildCreateListInput({
      params:
        args.params,

      listingTitle:
        args.listingTitle,

      description:
        args.description,

      priceRows:
        args.priceRows,

      status:
        args.status,

      assigneeId:
        args.assigneeId,
    });

  validateCreateListInput(
    input,
  );

  const created =
    await createListHTTP(
      input as ListPostCreateListInput,
    );

  const listId =
    _internal_getListIdFromListDTO(
      created,
    );

  if (!listId) {
    throw new Error(
      "created_list_missing_id",
    );
  }

  try {
    await uploadListImagesPolicyB({
      listId,

      files:
        args.images,

      mainImageIndex:
        args.mainImageIndex,
    });
  } catch (error) {
    args.onImageUploadFailed?.(
      LIST_IMAGE_UPLOAD_FAILED_MESSAGE,
      error,
    );
  }

  return created;
}