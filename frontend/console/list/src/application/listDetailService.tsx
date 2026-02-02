// frontend/console/list/src/application/listDetailService.tsx

import type * as React from "react";

// ✅ repository（HTTP）はここから呼ぶ
import {
  fetchListByIdHTTP,
  fetchListsHTTP,
  updateListByIdHTTP,
  type ListDTO,
} from "../infrastructure/http/list";

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
 * ✅ (2) decision normalize types (moved from hook)
 */
export type ListingDecisionNorm = "listing" | "holding" | "";

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

  // 更新者/更新日時
  updatedByName: string;
  updatedAt: string;

  // edit helpers (optional)
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
  // ルートパラメータ名の違い（listId / id）は吸収
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

// ---------------------------------------------------------
// ✅ (2) moved from hook: ListingDecisionNorm helpers
// ---------------------------------------------------------

export function normalizeListingDecisionNorm(v: unknown): ListingDecisionNorm {
  const x = s(v).toLowerCase();
  if (x === "listing" || x === "list") return "listing";
  if (x === "holding" || x === "hold") return "holding";
  return "";
}

export function toDecisionForUpdate(v: unknown): "list" | "hold" | undefined {
  const x = normalizeListingDecisionNorm(v);
  if (x === "listing") return "list";
  if (x === "holding") return "hold";
  return undefined;
}

// ---------------------------------------------------------
// ✅ (1) moved from hook: datetime format helper
// ---------------------------------------------------------

function pad2(n: number): string {
  return String(n).padStart(2, "0");
}

/**
 * ✅ yyyy/mm/dd/hh/mm 形式（入力が不正ならそのまま返す）
 */
export function formatYMDHM(v: unknown): string {
  const raw = s(v);
  if (!raw) return "";

  const d = new Date(raw);
  if (!Number.isFinite(d.getTime())) return raw;

  const yyyy = d.getFullYear();
  const mm = pad2(d.getMonth() + 1);
  const dd = pad2(d.getDate());
  const hh = pad2(d.getHours());
  const mi = pad2(d.getMinutes());

  return `${yyyy}/${mm}/${dd}/${hh}/${mi}`;
}

// ---------------------------------------------------------
// ✅ listImage helpers
// ---------------------------------------------------------

export type ListImage = {
  url: string;
  objectPath?: string;
};

// DraftImage（presentation hook 側）互換を緩く受ける
type DraftImageLike = {
  url?: unknown;
  isNew?: unknown;
  file?: unknown;
  objectPath?: unknown;
};

function dedupeUrlsKeepOrder(urls: string[]): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const u of urls) {
    const x = s(u);
    if (!x) continue;
    if (seen.has(x)) continue;
    seen.add(x);
    out.push(x);
  }
  return out;
}

function toListImage(x: any): ListImage | null {
  // string のみでも受ける（url として扱う）
  if (typeof x === "string") {
    const u = s(x);
    return u ? { url: u } : null;
  }

  const url = s(x?.url);
  if (!url) return null;

  const objectPath = s(x?.objectPath);
  return objectPath ? { url, objectPath } : { url };
}

/**
 * dto から listImages を正として読む
 */
export function normalizeListImages(dto: any): ListImage[] {
  const arr =
    (Array.isArray(dto?.listImages) ? dto.listImages : null) ??
    (Array.isArray(dto?.listImage) ? dto.listImage : null) ??
    [];

  const mapped: ListImage[] = [];
  for (const x of arr) {
    const li = toListImage(x);
    if (!li) continue;
    mapped.push(li);
  }

  // url 重複排除（順序維持）
  const urls = dedupeUrlsKeepOrder(mapped.map((x) => x.url));
  const urlSet = new Set(urls);

  // 先に dedupe した url の順序に沿って、最初に出現した objectPath を保持
  const firstByUrl = new Map<string, ListImage>();
  for (const x of mapped) {
    const u = x.url;
    if (!urlSet.has(u)) continue;
    if (!firstByUrl.has(u)) firstByUrl.set(u, x);
  }

  return urls.map((u) => firstByUrl.get(u) ?? { url: u });
}

/**
 * UI 用の imageUrls を生成
 * - 優先: dto.listImages / dto.listImage
 * - fallback: dto.imageUrls（旧）
 */
export function normalizeImageUrls(dto: any): string[] {
  const listImages = normalizeListImages(dto);
  if (listImages.length > 0) {
    return dedupeUrlsKeepOrder(listImages.map((x) => x.url));
  }

  const direct = Array.isArray(dto?.imageUrls) ? dto.imageUrls : [];
  const urls = direct.map((u: any) => s(u)).filter(Boolean);
  return dedupeUrlsKeepOrder(urls);
}

/**
 * hook の draftImages から「既存URL」と「新規File」を取り出す
 */
export function splitDraftImages(args: {
  draftImages: DraftImageLike[] | null | undefined;
}): {
  existingUrls: string[];
  newFiles: File[];
  listImages: ListImage[];
} {
  const src = Array.isArray(args.draftImages) ? args.draftImages : [];

  const existingUrls: string[] = [];
  const newFiles: File[] = [];
  const listImages: ListImage[] = [];

  for (const x of src) {
    const url = s((x as any)?.url);
    const isNew = Boolean((x as any)?.isNew);

    const objectPath = s((x as any)?.objectPath);

    if (url) {
      // 既存URLとして保持（isNew=true の blob: は existingUrls には入れない）
      if (!isNew && !url.startsWith("blob:")) {
        existingUrls.push(url);
        listImages.push(objectPath ? { url, objectPath } : { url });
      }
    }

    if (isNew) {
      const f = (x as any)?.file;
      if (f && typeof (f as any).name === "string") {
        newFiles.push(f as File);
      }
    }
  }

  const existingUrlsD = dedupeUrlsKeepOrder(existingUrls);

  // listImages も url ベースで dedupe（順序は existingUrlsD に合わせる）
  const first = new Map<string, ListImage>();
  for (const li of listImages) {
    if (!first.has(li.url)) first.set(li.url, li);
  }

  return {
    existingUrls: existingUrlsD,
    newFiles,
    listImages: existingUrlsD.map((u) => first.get(u) ?? { url: u }),
  };
}

/**
 * 更新payloadへ入れる listImages を作る（UI側が string[] でも DraftImage[] でもOK）
 */
export function buildListImagesForUpdate(input: {
  imageUrls?: string[] | null;
  draftImages?: DraftImageLike[] | null;
}): ListImage[] {
  const urls = Array.isArray(input.imageUrls) ? input.imageUrls : [];
  if (urls.length > 0) {
    return dedupeUrlsKeepOrder(urls.map((u) => s(u)).filter(Boolean)).map((u) => ({
      url: u,
    }));
  }

  const { listImages } = splitDraftImages({ draftImages: input.draftImages });
  return listImages;
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
 * priceRows は dto.priceRows のみ採用
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

    const rowAny = {
      id: modelId || String(idx),
      size,
      color,
      rgb,
      stock,
      price,
    };

    return rowAny as unknown as TRow;
  });
}

/**
 * draft(priceRows) -> backend update payload の prices（必要なら使う）
 */
export function buildPricesForUpdateFromPriceRows(
  rows: any[] | null | undefined,
): Array<{ modelId: string; price: number }> {
  const rr = Array.isArray(rows) ? rows : [];
  const out: Array<{ modelId: string; price: number }> = [];

  for (const r of rr) {
    const modelId = s(r?.modelId) || s(r?.id); // id = modelId
    if (!modelId) continue;

    const price = toNumberOrNull(r?.price);
    if (price === null) continue;

    out.push({ modelId, price });
  }

  return out;
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
// Model metadata ログ
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
// Service API
// ---------------------------------------------------------
export async function loadListDetailDTO(args: {
  listId: string;
  inventoryIdHint?: string;
}): Promise<ListDetailDTO> {
  const listId = s(args.listId);
  const inventoryIdHint = s(args.inventoryIdHint);

  if (!listId) throw new Error("invalid_list_id");

  const [detail, row] = await Promise.all([fetchListByIdHTTP(listId), fetchRowFromListRows({ listId })]);

  logModelMetadataFromDetail({ listId, dto: detail });

  const merged: any = { ...(detail as any) };

  // inventoryId は detail を優先。無ければ hint、最後に row。
  if (!s(merged?.inventoryId)) merged.inventoryId = inventoryIdHint;
  if (!s(merged?.inventoryId) && row) merged.inventoryId = s(row?.inventoryId);

  // assigneeId は detail を優先。無ければ row。
  if (!s(merged?.assigneeId) && row) merged.assigneeId = s(row?.assigneeId);

  // updatedAt / updatedBy も detail を優先。無ければ row（best-effort）
  if (!s(merged?.updatedAt) && row) merged.updatedAt = s(row?.updatedAt);
  if (!s(merged?.updatedBy) && row) {
    merged.updatedBy = s(row?.updatedBy) || s(row?.updatedByName);
  }

  // 表示名/ブランド/商品名/トークン名/ステータスは row があれば補完
  if (row) {
    if (!s(merged?.productName)) merged.productName = s(row?.productName);
    if (!s(merged?.tokenName)) merged.tokenName = s(row?.tokenName);
    if (!s(merged?.assigneeName)) merged.assigneeName = s(row?.assigneeName);
    if (!s(merged?.status) && s(row?.status)) merged.status = s(row?.status);

    if (!s(merged?.productBrandId)) merged.productBrandId = s(row?.productBrandId);
    if (!s(merged?.productBrandName)) merged.productBrandName = s(row?.productBrandName);

    if (!s(merged?.tokenBrandId)) merged.tokenBrandId = s(row?.tokenBrandId);
    if (!s(merged?.tokenBrandName)) merged.tokenBrandName = s(row?.tokenBrandName);

    // listImages が row にある場合だけ補完（detail を正とする）
    if (!Array.isArray(merged?.listImages) && Array.isArray((row as any)?.listImages)) {
      merged.listImages = (row as any).listImages;
    }
    if (!Array.isArray(merged?.listImage) && Array.isArray((row as any)?.listImage)) {
      merged.listImage = (row as any).listImage;
    }
  }

  // id を正規化して持っておく
  if (!s(merged?.id)) merged.id = listId;

  return merged as ListDetailDTO;
}

export function deriveListDetail<TRow extends Record<string, any> = any>(dto: any) {
  const listingTitle = s(dto?.title);
  const description = s(dto?.description);
  const decision = normalizeDecision(dto);

  const productBrandId = s(dto?.productBrandId);
  const productBrandName = s(dto?.productBrandName);
  const productName = s(dto?.productName);

  const tokenBrandId = s(dto?.tokenBrandId);
  const tokenBrandName = s(dto?.tokenBrandName);
  const tokenName = s(dto?.tokenName);

  const assigneeId = s(dto?.assigneeId);
  const assigneeName = s(dto?.assigneeName) || "未設定";

  const createdByName = s(dto?.createdBy);
  const createdAt = s(dto?.createdAt);

  const updatedByName = s(dto?.updatedBy) || s((dto as any)?.updatedByName);
  const updatedAt = s(dto?.updatedAt);

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

    updatedByName,
    updatedAt,
  };
}

export function computeListDetailPageTitle(args: { listId?: string; listingTitle?: string }) {
  const id = s(args.listId);
  const t = s(args.listingTitle) || "出品詳細";
  return id ? `${t}（listId: ${id}）` : t;
}

// ---------------------------------------------------------
// Update API
// - repositoryHTTP 側が priceRows を受けて prices を正規化するので、service からは priceRows を渡す
// ---------------------------------------------------------
export async function updateListDetailDTO(args: {
  listId: string;

  title?: string;
  description?: string;

  // PriceCard rows（id = modelId）
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

    // ✅ prices ではなく priceRows を渡す
    priceRows: args.priceRows,

    decision: args.decision,
    assigneeId: args.assigneeId,
    updatedBy: args.updatedBy,
  });
}
