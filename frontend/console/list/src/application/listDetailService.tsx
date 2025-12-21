// frontend/console/list/src/application/listDetailService.tsx

import * as React from "react";

// ✅ repository（HTTP）はここから呼ぶ
import {
  fetchListByIdHTTP,
  fetchListsHTTP,
  updateListByIdHTTP,
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

  // ✅ NEW: edit helpers (optional)
  listId?: string;
  inventoryId?: string;
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

/**
 * ✅ decision は UI で "list" | "hold" を使うため、backend の status/decision を最小限変換する
 * - "listing" => "list"（出品）
 * - "hold"    => "hold"（保留）
 * - それ以外はそのまま返す
 */
export function normalizeDecision(dto: any): string {
  const raw = (s(dto?.decision) || s(dto?.status)).toLowerCase();

  if (raw === "listing") return "list";
  if (raw === "hold") return "hold";

  return raw;
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

function toInt(v: unknown): number {
  const n = Number(v);
  if (!Number.isFinite(n)) return 0;
  return Math.trunc(n);
}

function toNumberOrNull(v: unknown): number | null {
  if (v === null || v === undefined) return null;
  const n = Number(v);
  if (!Number.isFinite(n)) return null;
  return n;
}

/**
 * ✅ priceRows は dto.priceRows のみ採用（名揺れ吸収しない）
 *
 * 重要:
 * - PriceCard の PriceRow 型は "modelId" を持たないため、ここでは絶対に返さない
 * - UI 側では id を正として扱う（id = modelId）
 * - ts2353（'modelId' does not exist in type 'PriceRow'）回避のため、
 *   返却オブジェクトは any/unknown 経由でキャストして返す
 */
export function normalizePriceRows<TRow extends Record<string, any> = any>(dto: any): TRow[] {
  const rowsRaw = Array.isArray(dto?.priceRows) ? dto.priceRows : [];

  return rowsRaw.map((r: any, idx: number) => {
    const modelId = s(r?.modelId);

    const size = s(r?.size);
    const color = s(r?.color);

    const stock = toInt(r?.stock);
    const price = toNumberOrNull(r?.price);

    const rgbNum = toNumberOrNull(r?.rgb);
    const rgb = rgbNum === null ? undefined : rgbNum;

    // ✅ PriceRow 互換（modelId を含めない）
    // id が空になると行が消える実装があり得るので、最後に idx フォールバック
    const rowAny = {
      id: modelId || String(idx),
      size,
      color,
      rgb,
      stock,
      price,
    };

    // ✅ generic を指定されても excess property check を発生させない
    return rowAny as unknown as TRow;
  });
}

/**
 * ✅ draft(priceRows) -> backend update payload の prices
 * - backend が受け取るのは prices: [{modelId, price}] を想定
 * - PriceCard row は modelId を持たないので、id を modelId として扱う
 * - price が null の行は送らない（= 変更無し扱いにしたい）
 */
export function buildPricesForUpdateFromPriceRows(
  rows: any[] | null | undefined,
): Array<{ modelId: string; price: number }> {
  const rr = Array.isArray(rows) ? rows : [];
  const out: Array<{ modelId: string; price: number }> = [];

  for (const r of rr) {
    const modelId = s(r?.modelId) || s(r?.id); // ✅ id = modelId
    if (!modelId) continue;

    const price = toNumberOrNull(r?.price);
    if (price === null) continue;

    out.push({ modelId, price });
  }

  return out;
}

// ---------------------------------------------------------
// ✅ Backend query(ListQuery.ListRows) 経由で product/token/assignee/brand を補完する
// - GET /lists は ListQuery.ListRows を通る想定
// ---------------------------------------------------------
async function fetchRowFromListRows(args: { listId: string }): Promise<any | null> {
  const id = s(args.listId);
  if (!id) return null;

  try {
    const rows = await fetchListsHTTP();
    const hit = Array.isArray(rows) ? rows.find((r: any) => s(r?.id) === id) : null;
    return hit || null;
  } catch {
    // ✅ 失敗しても detail は返せるので静かに null
    return null;
  }
}

// ---------------------------------------------------------
// ✅ Model metadata ログ（取得できたことが分かるログ）
// - 取得タイミング: loadListDetailDTO の A) /lists/{id} 取得直後
// ---------------------------------------------------------
function logModelMetadataFromDetail(args: { listId: string; dto: any }) {
  const listId = s(args.listId);
  const dto = args.dto;

  const rowsRaw = Array.isArray(dto?.priceRows) ? dto.priceRows : [];
  const count = rowsRaw.length;

  const sample = rowsRaw.slice(0, 4).map((r: any) => ({
    modelId: s(r?.modelId),
    size: s(r?.size),
    color: s(r?.color),
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
// Service API (hook から呼ぶ)
// ---------------------------------------------------------
export async function loadListDetailDTO(args: {
  listId: string;
  inventoryIdHint?: string; // ✅ ルートから来る想定
}): Promise<ListDetailDTO> {
  const listId = s(args.listId);
  const inventoryIdHint = s(args.inventoryIdHint);

  if (!listId) throw new Error("invalid_list_id");

  // A) 詳細：GET /lists/{id} （ListDetailDTO を返す想定）
  // B) rows：GET /lists（ListQuery.ListRows）
  const [detail, row] = await Promise.all([
    fetchListByIdHTTP(listId),
    fetchRowFromListRows({ listId }),
  ]);

  // ✅ Model metadata が取れていることが分かるログ（ここだけ残す）
  logModelMetadataFromDetail({ listId, dto: detail });

  const merged: any = { ...(detail as any) };

  // ✅ inventoryId は detail を優先。無ければ hint、最後に row。
  if (!s(merged?.inventoryId)) merged.inventoryId = inventoryIdHint;
  if (!s(merged?.inventoryId) && row) merged.inventoryId = s(row?.inventoryId);

  // ✅ assigneeId は detail を優先。無ければ row。
  if (!s(merged?.assigneeId) && row) merged.assigneeId = s(row?.assigneeId);

  // ✅ 表示名/ブランド/商品名/トークン名/ステータスは row があれば補完
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

  // ✅ 一応 id を正規化して持っておく（view 側で拾えるように）
  if (!s(merged?.id)) merged.id = listId;

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

  // ✅ priceRows は detail の priceRows を読む（id=modelId, size/color/rgb/stock/price）
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
// ✅ NEW: Update API (hook / page から呼べる)
// - list_handler の PUT /lists/{id} を叩くための入口
// - ここで prices を正規化して送る（id=modelId）
// ---------------------------------------------------------
export async function updateListDetailDTO(args: {
  listId: string;

  // editable fields
  title?: string;
  description?: string;

  // PriceCard rows（id = modelId）
  priceRows?: any[];

  // audit
  updatedBy?: string;
}): Promise<ListDTO> {
  const listId = s(args.listId);
  if (!listId) throw new Error("invalid_list_id");

  const payload: any = {};

  if (args.title !== undefined) payload.title = String(args.title ?? "");
  if (args.description !== undefined) payload.description = String(args.description ?? "");

  // ✅ backend が受けるのは prices（modelId+price）想定
  if (args.priceRows !== undefined) {
    payload.prices = buildPricesForUpdateFromPriceRows(args.priceRows);
  }

  if (args.updatedBy !== undefined) payload.updatedBy = s(args.updatedBy) || undefined;

  // ✅ ここが「list_handler が叩かれる」唯一の確実な入口になる
  return await updateListByIdHTTP({
    listId,
    ...payload,
  });
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
