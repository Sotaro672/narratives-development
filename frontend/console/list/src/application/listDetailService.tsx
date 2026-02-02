// frontend/console/list/src/application/listDetailService.tsx

import * as React from "react";

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

  // ✅ NEW: 更新者/更新日時（AdminCard などで表示するため）
  updatedByName: string;
  updatedAt: string;

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

// ---------------------------------------------------------
// ✅ listImage helpers（NEW）
// - backend が listImages (推奨) / listImage を返す or 受け取るケースに備える
// - UI は imageUrls を使うので、ここで url 配列へも変換できるようにする
// ---------------------------------------------------------

export type ListImage = {
  url: string;
  objectPath?: string; // GCS 等の objectPath を持たせたい場合の拡張（無くてもOK）
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
 * ✅ dto から listImages を正として読む
 * - listImages が無ければ listImage を見る（配列想定）
 * - それも無ければ []（imageUrls は別関数が fallback する）
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
 * ✅ UI 用の imageUrls を生成
 * - 優先: dto.listImages / dto.listImage から url を作る
 * - fallback: dto.imageUrls（旧フィールド）
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
 * ✅ hook の draftImages から「既存URL」と「新規File」を取り出す
 * - backend 実装（署名URL発行→PUT→attach）で使える形
 */
export function splitDraftImages(args: {
  draftImages: DraftImageLike[] | null | undefined;
}): {
  existingUrls: string[];
  newFiles: File[];
  listImages: ListImage[]; // 既存URLを listImages へ正規化（objectPathは持っていれば保持）
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
      // File 判定はブラウザ依存なのでゆるく
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
 * ✅ 更新payloadへ入れる listImages を作る（UI側が string[] でも DraftImage[] でもOK）
 * - 現段階では updateListByIdHTTP が listImages を受けるか不明なので、
 *   「payload を作る関数」だけ提供する（service からの呼び出しは後で）
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
 *
 * NOTE:
 * - 現在の update は repositoryHTTP 側で priceRows -> prices 正規化するため、
 *   service で prices を作って渡す必要はありません。
 * - ただし他用途で使えるので関数自体は残しておく。
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

  // ✅ updatedAt / updatedBy も detail を優先。無ければ row（best-effort）
  if (!s(merged?.updatedAt) && row) merged.updatedAt = s(row?.updatedAt);
  if (!s(merged?.updatedBy) && row) {
    merged.updatedBy = s(row?.updatedBy) || s(row?.updatedByName);
  }

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

    // ✅ listImages が row にある場合だけ補完（detail を正とする）
    if (!Array.isArray(merged?.listImages) && Array.isArray((row as any)?.listImages)) {
      merged.listImages = (row as any).listImages;
    }
    if (!Array.isArray(merged?.listImage) && Array.isArray((row as any)?.listImage)) {
      merged.listImage = (row as any).listImage;
    }
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

  // ✅ NEW: updatedByName / updatedAt
  // - 基本は dto.updatedBy / dto.updatedAt
  // - もし updatedByName が別フィールドで来ても表示できるように best-effort で拾う
  const updatedByName = s(dto?.updatedBy) || s((dto as any)?.updatedByName);
  const updatedAt = s(dto?.updatedAt);

  // ✅ listImages 対応：優先して listImages/listImage から作る（無ければ imageUrls）
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

    // ✅ NEW
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
// ✅ Update API (hook / page から呼べる)
// - 重要: repositoryHTTP 側が priceRows を受けて prices を正規化するので、
//         service からは priceRows を渡す（prices を渡すと落ちる）
// ---------------------------------------------------------
export async function updateListDetailDTO(args: {
  listId: string;

  // editable fields
  title?: string;
  description?: string;

  // PriceCard rows（id = modelId）
  priceRows?: any[];

  // decision (optional)
  decision?: "list" | "hold";

  // assignee (optional)
  assigneeId?: string;

  // audit
  updatedBy?: string;

  // ✅ NEW: listImages (optional)
  // NOTE: repositoryHTTP がまだ受けない場合があるので、必要になったら実装側で対応。
  // listImages?: ListImage[];
}): Promise<ListDTO> {
  const listId = s(args.listId);
  if (!listId) throw new Error("invalid_list_id");

  // ✅ repositoryHTTP が期待する UpdateListInput に合わせる
  return await updateListByIdHTTP({
    listId,
    title: args.title,
    description: args.description,

    // ✅ ここが修正点: prices ではなく priceRows を渡す
    priceRows: args.priceRows,

    decision: args.decision,
    assigneeId: args.assigneeId,
    updatedBy: args.updatedBy,
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
