// frontend/console/list/src/application/listDetailService.tsx

import type * as React from "react";

// repository（HTTP）はここから呼ぶ
import {
  fetchListByIdHTTP,
  fetchListsHTTP,
  updateListByIdHTTP,
} from "../infrastructure/repository";

import type { ListDTO } from "../infrastructure/dto";

import {
  computeListDetailPageTitle,
  deriveListDetail,
  formatYMDHM,
  normalizeDecision,
  normalizeImageUrls,
  normalizeListingDecisionNorm,
  normalizePriceRows,
  toDecisionForUpdate,
  updatePriceRowPrice,
  type ListingDecisionNorm,
} from "./listDetail/listDetailMapper";

export type { ListingDecisionNorm };

export {
  computeListDetailPageTitle,
  deriveListDetail,
  formatYMDHM,
  normalizeDecision,
  normalizeImageUrls,
  normalizeListingDecisionNorm,
  normalizePriceRows,
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
  const listId = String(params?.listId || params?.id || "").trim();
  const inventoryId = String(params?.inventoryId ?? "").trim();

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
  const id = String(args.listId ?? "").trim();
  if (!id) return null;

  try {
    const rows = await fetchListsHTTP();
    const hit = Array.isArray(rows)
      ? rows.find((r: any) => String(r?.id ?? "").trim() === id)
      : null;

    return hit || null;
  } catch {
    return null;
  }
}

// ---------------------------------------------------------
// Model metadata log
// ---------------------------------------------------------

function logModelMetadataFromDetail(args: { listId: string; dto: any }) {
  const listId = String(args.listId ?? "").trim();
  const dto = args.dto;

  const rowsRaw = Array.isArray(dto?.priceRows) ? dto.priceRows : [];
  const count = rowsRaw.length;

  const sample = rowsRaw.slice(0, 4).map((r: any) => ({
    modelId: String(r?.modelId ?? "").trim(),
    displayOrder: r?.displayOrder ?? null,
    size: String(r?.size ?? "").trim(),
    color: String(r?.color ?? "").trim(),
    price:
      r?.price === null || r?.price === undefined
        ? null
        : Number.isFinite(Number(r?.price))
          ? Number(r?.price)
          : null,
    stock: Number.isFinite(Number(r?.stock)) ? Number(r?.stock) : 0,
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
  const listId = String(args.listId ?? "").trim();

  if (!listId) throw new Error("invalid_list_id");

  const [detail, row] = await Promise.all([
    fetchListByIdHTTP(listId),
    fetchRowFromListRows({ listId }),
  ]);

  logModelMetadataFromDetail({ listId, dto: detail });

  const merged: any = { ...(detail as any) };

  // assigneeId は detail を優先。無ければ row。
  if (!String(merged?.assigneeId ?? "").trim() && row) {
    merged.assigneeId = String(row?.assigneeId ?? "").trim();
  }

  // updatedAt / updatedBy も detail を優先。無ければ row（best-effort）
  if (!String(merged?.updatedAt ?? "").trim() && row) {
    merged.updatedAt = String(row?.updatedAt ?? "").trim();
  }

  if (!String(merged?.updatedBy ?? "").trim() && row) {
    merged.updatedBy =
      String(row?.updatedBy ?? "").trim() ||
      String(row?.updatedByName ?? "").trim();
  }

  // 表示名/ブランド/商品名/トークン名/ステータスは row があれば補完
  if (row) {
    if (!String(merged?.productName ?? "").trim()) {
      merged.productName = String(row?.productName ?? "").trim();
    }

    if (!String(merged?.tokenName ?? "").trim()) {
      merged.tokenName = String(row?.tokenName ?? "").trim();
    }

    if (!String(merged?.assigneeName ?? "").trim()) {
      merged.assigneeName = String(row?.assigneeName ?? "").trim();
    }

    if (
      !String(merged?.status ?? "").trim() &&
      String(row?.status ?? "").trim()
    ) {
      merged.status = String(row?.status ?? "").trim();
    }

    if (!String(merged?.productBrandId ?? "").trim()) {
      merged.productBrandId = String(row?.productBrandId ?? "").trim();
    }

    if (!String(merged?.productBrandName ?? "").trim()) {
      merged.productBrandName = String(row?.productBrandName ?? "").trim();
    }

    if (!String(merged?.tokenBrandId ?? "").trim()) {
      merged.tokenBrandId = String(row?.tokenBrandId ?? "").trim();
    }

    if (!String(merged?.tokenBrandName ?? "").trim()) {
      merged.tokenBrandName = String(row?.tokenBrandName ?? "").trim();
    }
  }

  // id を正規化して持っておく
  if (!String(merged?.id ?? "").trim()) {
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
}): Promise<ListDTO> {
  const listId = String(args.listId ?? "").trim();
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