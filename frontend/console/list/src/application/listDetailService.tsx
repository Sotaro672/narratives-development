// frontend/console/list/src/application/listDetailService.tsx

import type * as React from "react";

// repository（HTTP）はここから呼ぶ
import {
  fetchListByIdHTTP,
  fetchListsHTTP,
  updateListByIdHTTP,
  type ListDTO,
} from "../infrastructure/http/list";

import {
  buildListImagesForUpdate,
  buildPricesForUpdateFromPriceRows,
  computeListDetailPageTitle,
  deriveListDetail,
  formatYMDHM,
  normalizeDecision,
  normalizeImageUrls,
  normalizeListImages,
  normalizeListingDecisionNorm,
  normalizePriceRows,
  s,
  splitDraftImages,
  toDecisionForUpdate,
  updatePriceRowPrice,
  type ListingDecisionNorm,
  type ListImage,
} from "./listDetail/listDetailMapper";

export type { ListingDecisionNorm, ListImage };

export {
  buildListImagesForUpdate,
  buildPricesForUpdateFromPriceRows,
  computeListDetailPageTitle,
  deriveListDetail,
  formatYMDHM,
  normalizeDecision,
  normalizeImageUrls,
  normalizeListImages,
  normalizeListingDecisionNorm,
  normalizePriceRows,
  s,
  splitDraftImages,
  toDecisionForUpdate,
  updatePriceRowPrice,
};

// ---------------------------------------------------------
// Types (presentation hook から使う)
// ---------------------------------------------------------

export type ListDetailRouteParams = {
  listId?: string;
  id?: string;
  inventoryId?: string;
};

// hook 内では必要なフィールドだけ参照するので Record<any> でOK
export type ListDetailDTO = ListDTO;

/**
 * presentation hook が import しても落ちないように（ts2305 対策）
 * - 本来は hook 側の型でOKだが、移譲の過程で service から import しているケースがあるため export する
 */
export type UseListDetailResult = {
  pageTitle: string;
  onBack: () => void;

  // loading/error
  loading: boolean;
  error: string;

  // raw dto
  dto: ListDetailDTO | null;

  // listing (view)
  listingTitle: string;
  description: string;

  // decision/status (view)
  decision: "list" | "hold" | "" | string;

  // display strings (already trimmed)
  productBrandId: string;
  productBrandName: string;
  productName: string;

  tokenBrandId: string;
  tokenBrandName: string;
  tokenName: string;

  // images (view)
  imageUrls: string[];
  mainImageIndex: number;
  setMainImageIndex: React.Dispatch<React.SetStateAction<number>>;

  // price (PriceCard 用)
  priceRows: any[];
  priceCard: any;

  // admin (view)
  assigneeId: string;
  assigneeName: string;
  createdByName: string;
  createdAt: string;

  // 更新者/更新日時
  updatedByName: string;
  updatedAt: string;

  // edit helpers (optional)
  listId?: string;
  inventoryId?: string;
};

// ---------------------------------------------------------
// Route params
// ---------------------------------------------------------

export function resolveListDetailParams(params: ListDetailRouteParams | undefined) {
  // ルートパラメータ名の違い（listId / id）は吸収
  const listId = s(params?.listId || params?.id);
  const inventoryId = s(params?.inventoryId);

  return {
    listId,
    inventoryId,
    raw: params,
  };
}

// ---------------------------------------------------------
// Backend query(ListQuery.ListRows) 経由で補完
// ---------------------------------------------------------

async function fetchRowFromListRows(args: { listId: string }): Promise<any | null> {
  const id = s(args.listId);
  if (!id) return null;

  try {
    const rows = await fetchListsHTTP();
    const hit = Array.isArray(rows) ? rows.find((r: any) => s(r?.id) === id) : null;

    return hit || null;
  } catch {
    return null;
  }
}

// ---------------------------------------------------------
// Model metadata log
// ---------------------------------------------------------

function logModelMetadataFromDetail(args: { listId: string; dto: any }) {
  const listId = s(args.listId);
  const dto = args.dto;

  const rowsRaw = Array.isArray(dto?.priceRows) ? dto.priceRows : [];
  const count = rowsRaw.length;

  const sample = rowsRaw.slice(0, 4).map((r: any) => ({
    modelId: s(r?.modelId),
    displayOrder: r?.displayOrder ?? null,
    size: s(r?.size),
    color: s(r?.color),
    kind: s(r?.kind),
    volumeValue:
      r?.volumeValue === null || r?.volumeValue === undefined
        ? null
        : Number.isFinite(Number(r?.volumeValue))
          ? Number(r?.volumeValue)
          : null,
    volumeUnit: s(r?.volumeUnit),
    rgb: Number.isFinite(Number(r?.rgb)) ? Number(r?.rgb) : null,
    stock: Number.isFinite(Number(r?.stock)) ? Number(r?.stock) : 0,
    price:
      r?.price === null || r?.price === undefined
        ? null
        : Number.isFinite(Number(r?.price))
          ? Number(r?.price)
          : null,
  }));

  // eslint-disable-next-line no-console
  console.log("[console/list/modelMetadata] priceRows(model metadata) resolved", {
    listId,
    count,
    sample,
  });
}

// ---------------------------------------------------------
// Service API
// ---------------------------------------------------------

export async function loadListDetailDTO(args: {
  listId: string;
  inventoryIdHint?: string;
}): Promise<ListDetailDTO> {
  const listId = s(args.listId);

  if (!listId) throw new Error("invalid_list_id");

  const [detail, row] = await Promise.all([
    fetchListByIdHTTP(listId),
    fetchRowFromListRows({ listId }),
  ]);

  logModelMetadataFromDetail({ listId, dto: detail });

  const merged: any = { ...(detail as any) };

  // assigneeId は detail を優先。無ければ row。
  if (!s(merged?.assigneeId) && row) {
    merged.assigneeId = s(row?.assigneeId);
  }

  // updatedAt / updatedBy も detail を優先。無ければ row（best-effort）
  if (!s(merged?.updatedAt) && row) {
    merged.updatedAt = s(row?.updatedAt);
  }

  if (!s(merged?.updatedBy) && row) {
    merged.updatedBy = s(row?.updatedBy) || s(row?.updatedByName);
  }

  // 表示名/ブランド/商品名/トークン名/ステータスは row があれば補完
  if (row) {
    if (!s(merged?.productName)) {
      merged.productName = s(row?.productName);
    }

    if (!s(merged?.tokenName)) {
      merged.tokenName = s(row?.tokenName);
    }

    if (!s(merged?.assigneeName)) {
      merged.assigneeName = s(row?.assigneeName);
    }

    if (!s(merged?.status) && s(row?.status)) {
      merged.status = s(row?.status);
    }

    if (!s(merged?.productBrandId)) {
      merged.productBrandId = s(row?.productBrandId);
    }

    if (!s(merged?.productBrandName)) {
      merged.productBrandName = s(row?.productBrandName);
    }

    if (!s(merged?.tokenBrandId)) {
      merged.tokenBrandId = s(row?.tokenBrandId);
    }

    if (!s(merged?.tokenBrandName)) {
      merged.tokenBrandName = s(row?.tokenBrandName);
    }

    // listImages が row にある場合だけ補完（detail を正とする）
    if (!Array.isArray(merged?.listImages) && Array.isArray((row as any)?.listImages)) {
      merged.listImages = (row as any).listImages;
    }

    if (!Array.isArray(merged?.listImage) && Array.isArray((row as any)?.listImage)) {
      merged.listImage = (row as any).listImage;
    }
  }

  // id を正規化して持っておく
  if (!s(merged?.id)) {
    merged.id = listId;
  }

  return merged as ListDetailDTO;
}

// ---------------------------------------------------------
// Update API
// - repositoryHTTP 側が priceRows を受けて prices を正規化するので、service からは priceRows を渡す
// ---------------------------------------------------------

export async function updateListDetailDTO(args: {
  listId: string;

  title?: string;
  description?: string;

  // PriceCard rows（id/modelId = modelId）
  priceRows?: any[];

  decision?: "list" | "hold";
  assigneeId?: string;

  updatedBy?: string;

  // listImages?: ListImage[]; // 必要になったら
}): Promise<ListDTO> {
  const listId = s(args.listId);
  if (!listId) throw new Error("invalid_list_id");

  return await updateListByIdHTTP({
    listId,
    title: args.title,
    description: args.description,

    // prices ではなく priceRows を渡す
    priceRows: args.priceRows,

    decision: args.decision,
    assigneeId: args.assigneeId,
    updatedBy: args.updatedBy,
  });
}