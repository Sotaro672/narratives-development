// frontend/console/inventory/src/application/listCreate/listCreateService.tsx

import type * as React from "react";
import type { RefObject } from "react";

import { getListCreateRaw } from "../../infrastructure/api/listCreateApi";
import type { ListCreateDTO } from "../../infrastructure/http/listCreateRepositoryHTTP.types";
import { mapListCreateDTO } from "../../infrastructure/http/listCreateRepositoryHTTP.mappers";

import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

import {
  createListHTTP,
  saveListImageFromFirebaseStorageHTTP,
  setListPrimaryImageHTTP,
} from "../../../../list/infrastructure/repository";

import type {
  CreateListInput as ListPostCreateListInput,
  ListDTO,
} from "../../../../list/infrastructure/dto";

import { uploadListImageToFirebaseStorage } from "../../../../list/infrastructure/firebase/listImageStorage";

/**
 * Hook 側で使う Ref 型（useRef<HTMLInputElement | null>(null) を許容）
 */
export type ImageInputRef = RefObject<HTMLInputElement | null>;

/**
 * List create route params
 *
 * UI ルートは inventoryId（= inventoryKey: "pb__tb"）のみを正とする。
 * productBlueprintId / tokenBlueprintId は互換用途では扱わない。
 */
export type ListCreateRouteParams = {
  inventoryId?: string;
};

export type ResolvedListCreateParams = {
  inventoryId: string;
  raw: ListCreateRouteParams;
};

/**
 * POST /lists の priceRows
 *
 * - modelId を識別子として使う
 * - 未入力 price は undefined のまま素通りさせる
 * - 明示的な未設定は null
 * - 入力済み価格は number
 */
export type CreateListPriceRow = {
  modelId: string;
  price?: number | null;
};

export type PriceCardMode = "view" | "edit";

export type PriceRowKind = "apparel" | "alcohol" | string;

/**
 * PriceCard 用 row
 *
 * backend response を正とする。
 *
 * - modelId を識別子として使う
 * - id / modelID / model_id などの名揺れは持たない
 * - React key は displayOrder ではなく modelId を使う
 * - displayOrder は重複/未設定があり得る
 * - 並び順は displayOrder 昇順のみ
 * - 未設定は null を保持し、UI 側で末尾扱いにする
 *
 * category ごとの表示:
 * - apparel: size / color / rgb
 * - alcohol: volumeValue / volumeUnit
 */
export type PriceRow = {
  modelId: string;

  /**
   * モデル種別。
   *
   * model domain の variation.kind 由来。
   * - apparel
   * - alcohol
   */
  kind?: PriceRowKind | null;

  /**
   * 並び順。
   * 未設定は null のまま保持する。
   */
  displayOrder?: number | null;

  /**
   * apparel category 用。
   */
  size?: string | null;

  /**
   * apparel category 用。
   */
  color?: string | null;

  /**
   * RGB。
   * backend response では number が基本。
   * 既存 UI 互換として "#RRGGBB" string も許容する。
   */
  rgb?: number | string | null;

  /**
   * alcohol category 用。
   *
   * 例: 720, 1000
   */
  volumeValue?: number | null;

  /**
   * alcohol category 用。
   *
   * 例: "ml", "L"
   */
  volumeUnit?: string | null;

  stock: number;

  /**
   * 価格。
   * 未入力は undefined、明示的な未設定は null。
   */
  price?: number | null;
};

export type PriceCardProps = {
  title?: string;
  rows: PriceRow[];
  className?: string;

  mode?: PriceCardMode;

  /**
   * ProductBlueprintCategory.code を渡す想定。
   *
   * 例:
   * - "apparel.tops"
   * - "alcohol.sake"
   */
  productBlueprintCategory?: string;

  /**
   * edit 時に価格を更新するコールバック。
   *
   * hook 内で displayOrder で並べ替えても、
   * index は「元の rows 配列の index」を返す。
   */
  onChangePrice?: (index: number, price: number | null, row: PriceRow) => void;

  currencySymbol?: string;
};

export type PriceRowVM = {
  /**
   * React key 用の識別子。
   */
  modelId: string;

  /**
   * モデル種別。
   *
   * PriceCard の category 表示分岐で使う。
   */
  kind?: PriceRowKind | null;

  /**
   * 並び順。
   * 未設定は null。
   */
  displayOrder: number | null;

  /**
   * apparel category 用。
   */
  size?: string | null;

  /**
   * apparel category 用。
   */
  color?: string | null;

  /**
   * alcohol category 用。
   */
  volumeValue?: number | null;

  /**
   * alcohol category 用。
   */
  volumeUnit?: string | null;

  stock: number;

  bgColor: string;
  rgbTitle: string;

  priceInputValue: string;
  priceDisplayText: string;

  onChangePriceInput: (e: React.ChangeEvent<HTMLInputElement>) => void;
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
  decision: "list" | "hold";
  assigneeId?: string;
  priceRows: CreateListPriceRow[];
};

/**
 * - UI ルートは inventoryId（= inventoryKey: "pb__tb"）のみを正とする
 * - backend fetch も inventoryId のみを使う（/inventory/list-create/:inventoryId）
 * - productBlueprintId / tokenBlueprintId は一切扱わない（互換も廃止）
 */
export function resolveListCreateParams(
  raw: ListCreateRouteParams,
): ResolvedListCreateParams {
  return {
    inventoryId: raw.inventoryId,
    raw,
  } as ResolvedListCreateParams;
}

export function canFetchListCreate(p: ResolvedListCreateParams): boolean {
  return Boolean(p.inventoryId);
}

export function buildListCreateFetchInput(p: ResolvedListCreateParams): {
  inventoryId?: string;
} {
  if (!p.inventoryId) {
    return { inventoryId: undefined };
  }

  return {
    inventoryId: p.inventoryId,
  };
}

export function getInventoryIdFromDTO(
  dto: ListCreateDTO | null | undefined,
): string {
  return dto?.inventoryId ?? "";
}

/**
 * リダイレクトは不要
 */
export function shouldRedirectToInventoryIdRoute(_: {
  currentInventoryId: string;
  gotInventoryId: string;
  alreadyRedirected: boolean;
}): boolean {
  return false;
}

export function buildInventoryDetailPath(inventoryId: string): string {
  if (!inventoryId) return "/inventory";
  return `/inventory/detail/${encodeURIComponent(inventoryId)}`;
}

export function buildInventoryListCreatePath(inventoryId: string): string {
  if (!inventoryId) return "/inventory/list/create";
  return `/inventory/list/create/${encodeURIComponent(inventoryId)}`;
}

export function buildBackPath(p: ResolvedListCreateParams): string {
  if (p.inventoryId) return buildInventoryDetailPath(p.inventoryId);
  return "/inventory";
}

export function buildAfterCreatePath(p: ResolvedListCreateParams): string {
  if (p.inventoryId) return buildInventoryDetailPath(p.inventoryId);
  return "/inventory";
}

export function extractDisplayStrings(dto: ListCreateDTO | null): {
  productBrandName: string;
  productName: string;
  tokenBrandName: string;
  tokenName: string;
} {
  return {
    productBrandName: dto?.productBrandName ?? "",
    productName: dto?.productName ?? "",
    tokenBrandName: dto?.tokenBrandName ?? "",
    tokenName: dto?.tokenName ?? "",
  };
}

/**
 * backend の ListCreateDTO.priceRows を PriceCard 用 PriceRow[] に変換
 * - 期待値: inventory/application の PriceRow を正とする
 * - 識別子は modelId を正とする
 * - 並び順は displayOrder（未設定は null を保持）
 * - 名揺れ補完はしない
 */
export function mapDTOToPriceRows(dto: ListCreateDTO | null): PriceRow[] {
  const rows = Array.isArray(dto?.priceRows) ? dto.priceRows : [];

  return rows.map((r: any) => {
    const displayOrderRaw = r.displayOrder;
    const displayOrder =
      displayOrderRaw === null || displayOrderRaw === undefined
        ? null
        : Number(displayOrderRaw);

    const stockRaw = Number(r.stock ?? 0);
    const stock = Number.isFinite(stockRaw) ? stockRaw : 0;

    const row: PriceRow = {
      modelId: r.modelId,
      kind: r.kind ?? null,
      displayOrder,
      size: r.size,
      color: r.color,
      stock,
      rgb: r.rgb as any,
      volumeValue: r.volumeValue ?? null,
      volumeUnit: r.volumeUnit ?? null,
      price: r.price === undefined ? null : (r.price as any),
    };

    return row;
  });
}

/**
 * 初期表示用: PriceRow[] を返す
 */
export function initPriceRowsFromDTO(dto: ListCreateDTO | null): PriceRow[] {
  return mapDTOToPriceRows(dto);
}

export function normalizeCreateListPriceRows(
  rows: unknown[],
): CreateListPriceRow[] {
  const arr = Array.isArray(rows) ? rows : [];

  return arr.map((r) => {
    const row = r as {
      modelId: string;
      price?: number | null;
    };

    return {
      modelId: row.modelId,
      price: row.price,
    };
  });
}

export function buildCreateListInput(args: {
  params: ResolvedListCreateParams;
  listingTitle: string;
  description: string;
  priceRows: unknown[];
  decision: "list" | "hold";
  assigneeId?: string;
}): CreateListInput {
  const priceRows = normalizeCreateListPriceRows(args.priceRows);

  return {
    // inventoryId(pb__tb) をそのまま送る
    inventoryId: args.params.inventoryId,
    title: args.listingTitle,
    description: args.description,
    decision: args.decision,
    assigneeId: args.assigneeId,
    priceRows,
  };
}

export function validateCreateListInput(input: CreateListInput): void {
  if (!input.title) {
    throw new Error("タイトルを入力してください。");
  }

  const rows = Array.isArray(input.priceRows) ? input.priceRows : [];

  if (rows.length === 0) {
    throw new Error("価格が未設定です（価格行がありません）。");
  }

  const missingModelId = rows.find((r) => {
    return !r.modelId;
  });

  if (missingModelId) {
    throw new Error("価格行に modelId が含まれていません。");
  }
}

/**
 * 複数画像を Firebase Storage へ直接アップロード
 * → backend にメタ情報登録
 * → primary image 設定
 *
 * Policy B:
 * - List 作成後の listId を使って Firebase Storage へ upload
 * - Firebase Storage download URL を取得
 * - saveListImageFromFirebaseStorageHTTP で image record を登録
 *
 * primary:
 * - backend の List.imageId は images subcollection docID
 * - objectPath ではなく imageId を渡す
 */
export async function uploadListImagesPolicyB(args: {
  listId: string;
  files: File[];
  mainImageIndex: number;
  createdBy?: string;
}): Promise<{
  registered: Array<{ imageId: string; displayOrder: number }>;
  primaryImageId?: string;
}> {
  const listId = String(args.listId ?? "").trim();
  const files = Array.isArray(args.files) ? args.files : [];

  const requestedMainImageIndex = Number.isFinite(Number(args.mainImageIndex))
    ? Number(args.mainImageIndex)
    : 0;

  const mainImageIndex =
    requestedMainImageIndex >= 0 && requestedMainImageIndex < files.length
      ? requestedMainImageIndex
      : 0;

  if (!listId) throw new Error("invalid_list_id");
  if (files.length === 0) return { registered: [] };

  const uid = args.createdBy || auth.currentUser?.uid || "system";
  const now = new Date().toISOString();

  const registered: Array<{ imageId: string; displayOrder: number }> = [];

  for (let i = 0; i < files.length; i++) {
    const file = files[i];
    if (!file) continue;

    const uploaded = await uploadListImageToFirebaseStorage({
      listId,
      file,
    });

    await saveListImageFromFirebaseStorageHTTP({
      listId,
      id: uploaded.imageId,
      url: uploaded.url,
      objectPath: uploaded.objectPath,
      size: Number(file.size || 0),
      displayOrder: i,
      fileName: file.name,
      contentType: file.type || "application/octet-stream",
      createdBy: uid,
      createdAt: now,
    });

    registered.push({
      imageId: uploaded.imageId,
      displayOrder: i,
    });
  }

  const primary =
    registered.find((x) => x.displayOrder === mainImageIndex) || registered[0];

  if (primary?.imageId) {
    await setListPrimaryImageHTTP({
      listId,
      imageId: primary.imageId,
      updatedBy: uid,
      now,
    });
  }

  return {
    registered,
    primaryImageId: primary?.imageId,
  };
}

export function _internal_getListIdFromListDTO(dto: ListDTO): string {
  return dto.id;
}

export const LIST_IMAGE_UPLOAD_FAILED_MESSAGE =
  "画像アップロードに失敗しました。後から追加できます。";

/**
 * ListCreateDTO を取得する（Hook からはこれだけ呼ぶ）
 *
 * 方針:
 * - GET /inventory/list-create/{inventoryId} の response を唯一の正とする。
 * - frontend では model variations API を呼ばない。
 * - priceRows は backend 側で productCategory / model kind に応じた完成形になっている前提。
 *
 * category ごとの表示:
 * - apparel: priceRows[].modelNumber / size / color / rgb
 * - alcohol: priceRows[].modelNumber / volumeValue / volumeUnit
 */
export async function loadListCreateDTOFromParams(
  p: ResolvedListCreateParams,
): Promise<ListCreateDTO> {
  const input = buildListCreateFetchInput(p);

  const raw = await getListCreateRaw(input);
  return mapListCreateDTO(raw);
}

/**
 * list 作成（POST /lists） + 画像（Policy B）
 *
 * Policy B:
 * 1. 画像なしで List を先に作成する
 * 2. 作成済み listId を使って Firebase Storage へ upload する
 * 3. backend に /lists/{listId}/images として image record を作成する
 * 4. primary image を設定する
 *
 * 重要:
 * - List 作成後に画像 upload / image record 登録 / primary image 設定が失敗しても、
 *   List 作成自体は成功として返す。
 * - UI には onImageUploadFailed で
 *   「画像アップロードに失敗しました。後から追加できます。」
 *   を表示する。
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

  onImageUploadFailed?: (message: string, error: unknown) => void;
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
  const created = await createListHTTP(input as ListPostCreateListInput);

  const listId = _internal_getListIdFromListDTO(created);

  if (!listId) {
    throw new Error("created_list_missing_id");
  }

  // 3) images (Policy B)
  //
  // List 作成後の画像失敗は List 作成を失敗扱いにしない。
  // 画面側で「画像アップロードに失敗しました。後から追加できます。」を出せるようにする。
  if (images.length > 0) {
    try {
      await uploadListImagesPolicyB({
        listId,
        files: images,
        mainImageIndex,
        createdBy: auth.currentUser?.uid,
      });
    } catch (error) {
      args.onImageUploadFailed?.(LIST_IMAGE_UPLOAD_FAILED_MESSAGE, error);
    }
  }

  return created;
}