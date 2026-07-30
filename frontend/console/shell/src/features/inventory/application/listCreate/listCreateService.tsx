// frontend/console/shell/src/features/inventory/application/listCreate/listCreateService.tsx

import type * as React from "react";

import { getListCreateRaw } from "../../infrastructure/api/listCreateApi";
import type { ListCreateDTO } from "../../infrastructure/http/listCreateRepositoryHTTP.types";
import { mapListCreateDTO } from "../../infrastructure/http/listCreateRepositoryHTTP.mappers";

import { auth } from "../../../../auth/infrastructure/config/firebaseClient";

import type {
  List,
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

/**
 * POST /listsのpriceRows。
 *
 * - modelIdを識別子として使う
 * - 未入力priceはundefinedのまま素通りさせる
 * - 明示的な未設定はnull
 * - 入力済み価格はnumber
 */
export type CreateListPriceRow = {
  modelId: string;
  price?: number | null;
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
 * - id / modelID / model_idなどの名揺れは持たない
 * - React keyはdisplayOrderではなくmodelIdを使う
 * - displayOrderは重複または未設定があり得る
 * - 並び順はdisplayOrder昇順のみ
 * - 未設定はnullを保持し、UI側で末尾扱いにする
 *
 * categoryごとの表示:
 * - apparel: size / color / rgb
 * - alcohol: volumeValue / volumeUnit
 */
export type PriceRow = {
  modelId: string;

  /**
   * モデル種別。
   *
   * model domainのvariation.kind由来。
   */
  kind?: PriceRowKind | null;

  /**
   * 並び順。
   * 未設定はnullのまま保持する。
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
   * 未入力はundefined、明示的な未設定はnull。
   */
  price?: number | null;
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
   * hook内でdisplayOrderにより並べ替えても、
   * indexは元のrows配列のindexを返す。
   */
  onChangePrice?: (
    index: number,
    price: number | null,
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
   * 未設定はnull。
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

export type CreateListInput = {
  inventoryId: string;
  title: string;
  description: string;
  status: ListStatus;
  assigneeId?: string;
  priceRows: CreateListPriceRow[];
};

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

export function normalizeCreateListPriceRows(
  rows: unknown[],
): CreateListPriceRow[] {
  const sourceRows =
    Array.isArray(rows)
      ? rows
      : [];

  return sourceRows.map(
    (rawRow) => {
      const row =
        rawRow as {
          modelId: string;
          price?: number | null;
        };

      return {
        modelId:
          row.modelId,

        price:
          row.price,
      };
    },
  );
}

export function buildCreateListInput(
  args: {
    params: ResolvedListCreateParams;
    listingTitle: string;
    description: string;
    priceRows: unknown[];
    status: ListStatus;
    assigneeId?: string;
  },
): CreateListInput {
  const priceRows =
    normalizeCreateListPriceRows(
      args.priceRows,
    );

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

    priceRows,
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

  const rows =
    Array.isArray(
      input.priceRows,
    )
      ? input.priceRows
      : [];

  if (rows.length === 0) {
    throw new Error(
      "価格が未設定です（価格行がありません）。",
    );
  }

  const missingModelId =
    rows.find(
      (row) => {
        return !row.modelId;
      },
    );

  if (missingModelId) {
    throw new Error(
      "価格行に modelId が含まれていません。",
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
    createdBy?: string;
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

  const files =
    Array.isArray(
      args.files,
    )
      ? args.files
      : [];

  const requestedMainImageIndex =
    Number.isFinite(
      Number(
        args.mainImageIndex,
      ),
    )
      ? Number(
          args.mainImageIndex,
        )
      : 0;

  const mainImageIndex =
    requestedMainImageIndex >= 0 &&
    requestedMainImageIndex <
      files.length
      ? requestedMainImageIndex
      : 0;

  if (!listId) {
    throw new Error(
      "invalid_list_id",
    );
  }

  if (files.length === 0) {
    return {
      registered: [],
    };
  }

  const uid =
    args.createdBy ||
    auth.currentUser?.uid ||
    "system";

  const now =
    new Date().toISOString();

  const registered: Array<{
    imageId: string;
    displayOrder: number;
  }> = [];

  for (
    let index = 0;
    index < files.length;
    index += 1
  ) {
    const file =
      files[index];

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

      createdBy:
        uid,

      createdAt:
        now,
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

export const LIST_IMAGE_UPLOAD_FAILED_MESSAGE =
  "画像アップロードに失敗しました。後から追加できます。";

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
 * 1. 画像なしでListを先に作成する
 * 2. 作成済みlistIdを使ってFirebase Storageへuploadする
 * 3. backendにimage recordを作成する
 * 4. primary imageを設定する
 *
 * List作成後に画像登録が失敗しても、
 * List作成自体は成功として扱う。
 */
export async function createListWithImages(
  args: {
    params: ResolvedListCreateParams;
    listingTitle: string;
    description: string;
    priceRows: any[];
    status: ListStatus;
    assigneeId?: string;

    images?: File[];
    mainImageIndex?: number;

    onImageUploadFailed?: (
      message: string,
      error: unknown,
    ) => void;
  },
): Promise<List> {
  const images =
    Array.isArray(
      args.images,
    )
      ? args.images
      : [];

  const mainImageIndex =
    Number.isFinite(
      Number(
        args.mainImageIndex,
      ),
    )
      ? Number(
          args.mainImageIndex,
        )
      : 0;

  const input:
    CreateListInput =
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

  if (images.length > 0) {
    try {
      await uploadListImagesPolicyB({
        listId,

        files:
          images,

        mainImageIndex,

        createdBy:
          auth.currentUser?.uid,
      });
    } catch (error) {
      args.onImageUploadFailed?.(
        LIST_IMAGE_UPLOAD_FAILED_MESSAGE,
        error,
      );
    }
  }

  return created;
}