// frontend/console/list/src/application/listDetailService.tsx

import * as React from "react";

// ✅ repository（HTTP）はここから呼ぶ
import {
  fetchListByIdHTTP,
  fetchListsHTTP,
  type ListDTO,
} from "../infrastructure/http/listRepositoryHTTP";

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
 * ✅ presentation hook が import しても落ちないように（ts2305 対策）
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
};

// ---------------------------------------------------------
// Helpers (shared)
// ---------------------------------------------------------
export function s(v: unknown): string {
  return String(v ?? "").trim();
}

export function resolveListDetailParams(params: ListDetailRouteParams | undefined) {
  // ✅ ルートパラメータ名の違い（listId / id）は吸収（DTO の名揺れではない）
  const listId = s(params?.listId || params?.id);
  const inventoryId = s(params?.inventoryId);

  return {
    listId,
    inventoryId,
    raw: params,
  };
}

// ✅ decision は backend の status/decision をそのまま使う（名揺れ吸収しない）
export function normalizeDecision(dto: any): string {
  return s(dto?.decision) || s(dto?.status);
}

// ✅ imageUrls は dto.imageUrls のみ採用（名揺れ吸収しない）
export function normalizeImageUrls(dto: any): string[] {
  const direct = Array.isArray(dto?.imageUrls) ? dto.imageUrls : [];
  const urls = direct.map((u: any) => s(u)).filter(Boolean);

  // dedupe (keep order)
  const seen = new Set<string>();
  const out: string[] = [];
  for (const u of urls) {
    if (seen.has(u)) continue;
    seen.add(u);
    out.push(u);
  }
  return out;
}

// ✅ prices は dto.prices のみ採用（名揺れ吸収しない）
export function normalizePriceRows<TRow extends Record<string, any> = any>(dto: any): TRow[] {
  const rowsRaw = Array.isArray(dto?.prices) ? dto.prices : [];

  return rowsRaw.map((r: any, idx: number) => {
    const modelId = s(r?.modelId);
    const size = s(r?.size);
    const color = s(r?.color);

    const stockRaw = r?.stock;
    const stock = Number.isFinite(Number(stockRaw)) ? Number(stockRaw) : 0;

    const priceRaw = r?.price;
    const price =
      priceRaw === null || priceRaw === undefined
        ? null
        : Number.isFinite(Number(priceRaw))
          ? Number(priceRaw)
          : null;

    const rgbRaw = r?.rgb;
    const rgb = Number.isFinite(Number(rgbRaw)) ? Number(rgbRaw) : null;

    const row: any = {
      modelId,
      size,
      color,
      rgb: rgb === null ? undefined : rgb,
      stock,
      price,
      _idx: idx,
      _raw: r,
    };

    return row as TRow;
  });
}

// ---------------------------------------------------------
// ✅ Backend query(ListQuery.ListRows) 経由で product/token/assignee/brand を補完する
// - GET /lists は ListQuery.ListRows を通る想定（ログの通り）
// ---------------------------------------------------------
async function fetchRowFromListRows(args: { listId: string }): Promise<any | null> {
  const id = s(args.listId);
  if (!id) return null;

  try {
    const rows = await fetchListsHTTP();
    const hit = Array.isArray(rows) ? rows.find((r: any) => s(r?.id) === id) : null;

    // eslint-disable-next-line no-console
    console.log("[console/list/listDetailService] row resolved via /lists(ListRows)", {
      listId: id,
      found: Boolean(hit),
      inventoryId: s(hit?.inventoryId),
      assigneeId: s(hit?.assigneeId),
      productBrandId: s(hit?.productBrandId),
      productBrandName: s(hit?.productBrandName),
      productName: s(hit?.productName),
      tokenBrandId: s(hit?.tokenBrandId),
      tokenBrandName: s(hit?.tokenBrandName),
      tokenName: s(hit?.tokenName),
      assigneeName: s(hit?.assigneeName),
      status: s(hit?.status),
    });

    return hit || null;
  } catch (e) {
    // eslint-disable-next-line no-console
    console.warn("[console/list/listDetailService] fetchRowFromListRows failed", {
      listId: id,
      error: String(e instanceof Error ? e.message : e),
      raw: e,
    });
    return null;
  }
}

// ---------------------------------------------------------
// Service API (hook から呼ぶ)
// ---------------------------------------------------------
export async function loadListDetailDTO(args: {
  listId: string;
  inventoryIdHint?: string; // ✅ ルートから来る想定（ログ通り）
}): Promise<ListDetailDTO> {
  const listId = s(args.listId);
  const inventoryIdHint = s(args.inventoryIdHint);

  if (!listId) throw new Error("invalid_list_id");

  // eslint-disable-next-line no-console
  console.log("[console/list/listDetailService] loadListDetailDTO start", {
    listId,
    inventoryIdHint,
  });

  // A) 詳細：GET /lists/{id}
  // B) rows：GET /lists（ListQuery.ListRows）
  const [detail, row] = await Promise.all([fetchListByIdHTTP(listId), fetchRowFromListRows({ listId })]);

  const merged: any = { ...(detail as any) };

  // ✅ inventoryId は detail を優先。無ければ hint、最後に row。
  if (!s(merged?.inventoryId)) merged.inventoryId = inventoryIdHint;
  if (!s(merged?.inventoryId) && row) merged.inventoryId = s(row?.inventoryId);

  // ✅ assigneeId は detail を優先。無ければ row。
  if (!s(merged?.assigneeId) && row) merged.assigneeId = s(row?.assigneeId);

  // ✅ 表示名/ブランド/商品名/トークン名/ステータスは row が正（ログ通り）なので、row があれば入れる
  if (row) {
    if (!s(merged?.productName)) merged.productName = s(row?.productName);
    if (!s(merged?.tokenName)) merged.tokenName = s(row?.tokenName);
    if (!s(merged?.assigneeName)) merged.assigneeName = s(row?.assigneeName);
    if (!s(merged?.status) && s(row?.status)) merged.status = s(row?.status);

    if (!s(merged?.productBrandId)) merged.productBrandId = s(row?.productBrandId);
    if (!s(merged?.productBrandName)) merged.productBrandName = s(row?.productBrandName);

    if (!s(merged?.tokenBrandId)) merged.tokenBrandId = s(row?.tokenBrandId);
    if (!s(merged?.tokenBrandName)) merged.tokenBrandName = s(row?.tokenBrandName);
  }

  // eslint-disable-next-line no-console
  console.log("[console/list/listDetailService] loadListDetailDTO end", {
    listId,
    inventoryId: s(merged?.inventoryId),
    assigneeId: s(merged?.assigneeId),
    assigneeName: s(merged?.assigneeName),
    productBrandId: s(merged?.productBrandId),
    productBrandName: s(merged?.productBrandName),
    productName: s(merged?.productName),
    tokenBrandId: s(merged?.tokenBrandId),
    tokenBrandName: s(merged?.tokenBrandName),
    tokenName: s(merged?.tokenName),
  });

  return merged as ListDetailDTO;
}

export function deriveListDetail<TRow extends Record<string, any> = any>(dto: any) {
  const listingTitle = s(dto?.title);
  const description = s(dto?.description);
  const decision = normalizeDecision(dto);

  // ✅ 右カラム：商品/トークン（ブランド含む）
  const productBrandId = s(dto?.productBrandId);
  const productBrandName = s(dto?.productBrandName);
  const productName = s(dto?.productName);

  const tokenBrandId = s(dto?.tokenBrandId);
  const tokenBrandName = s(dto?.tokenBrandName);
  const tokenName = s(dto?.tokenName);

  // ✅ 担当者（ID + Name）
  const assigneeId = s(dto?.assigneeId);
  const assigneeName = s(dto?.assigneeName) || "未設定";

  // ✅ createdBy は dto.createdBy をそのまま使う（名揺れ吸収しない）
  const createdByName = s(dto?.createdBy);
  const createdAt = s(dto?.createdAt);

  const imageUrls = normalizeImageUrls(dto);
  const priceRows = normalizePriceRows<TRow>(dto);

  return {
    listingTitle,
    description,
    decision,

    productBrandId,
    productBrandName,
    productName,

    tokenBrandId,
    tokenBrandName,
    tokenName,

    imageUrls,
    priceRows,

    assigneeId,
    assigneeName,

    createdByName,
    createdAt,
  };
}

export function computeListDetailPageTitle(args: { listId?: string; listingTitle?: string }) {
  const id = s(args.listId);
  const t = s(args.listingTitle) || "出品詳細";
  return id ? `${t}（listId: ${id}）` : t;
}

// ---------------------------------------------------------
// Hook helpers (presentation hook から使う)
// ---------------------------------------------------------
export function useCancelledRef() {
  const cancelledRef = React.useRef(false);
  React.useEffect(() => {
    cancelledRef.current = false;
    return () => {
      cancelledRef.current = true;
    };
  }, []);
  return cancelledRef;
}

export function useMainImageIndexGuard(args: {
  imageUrls: string[];
  mainImageIndex: number;
  setMainImageIndex: React.Dispatch<React.SetStateAction<number>>;
}) {
  const { imageUrls, mainImageIndex, setMainImageIndex } = args;

  React.useEffect(() => {
    if (imageUrls.length === 0) {
      if (mainImageIndex !== 0) setMainImageIndex(0);
      return;
    }
    if (mainImageIndex < 0 || mainImageIndex > imageUrls.length - 1) {
      setMainImageIndex(0);
    }
  }, [imageUrls.length, mainImageIndex, setMainImageIndex]);
}
