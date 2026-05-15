// frontend/console/list/src/application/listDetail/listDetailMapper.ts

import { safeDateTimeLabelJa } from "../../../../shell/src/shared/util/dateJa";

export type ListingDecisionNorm = "listing" | "holding" | "";

export type ListImage = {
  url: string;
  objectPath?: string;
};

// DraftImage（presentation hook 側）互換を緩く受ける
export type DraftImageLike = {
  url?: unknown;
  isNew?: unknown;
  file?: unknown;
  objectPath?: unknown;
};

// ---------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------

export function s(v: unknown): string {
  return String(v ?? "").trim();
}

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

function toDisplayOrderOrNull(v: unknown): number | null {
  if (v === null || v === undefined) return null;

  const n = Number(v);
  if (!Number.isFinite(n)) return null;

  const x = Math.trunc(n);

  // 互換: 0 は未設定扱いに寄せる
  if (x === 0) return null;

  return x;
}

// ---------------------------------------------------------
// Decision helpers
// ---------------------------------------------------------

/**
 * decision は UI で "list" | "hold" を使うため、backend の status/decision を最小限変換する
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
// Datetime format helper
// ---------------------------------------------------------

/**
 * yyyy/mm/dd hh:mm:ss 形式
 * - shell の safeDateTimeLabelJa を共通利用
 * - 空/不正な値でも落ちず、fallback は空文字
 */
export function formatYMDHM(v: unknown): string {
  return safeDateTimeLabelJa(s(v), "");
}

// ---------------------------------------------------------
// listImage helpers
// ---------------------------------------------------------

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
 * dto から listImages を読む
 *
 * 現在の表示正は dto.imageUrls だが、既存互換として listImages/listImage も正規化可能にしておく。
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
 *
 * 正:
 * - GET /lists/{id} response の imageUrls: string[] を採用
 * - GCS / bucket / storagePath 由来の組み立てはしない
 */
export function normalizeImageUrls(dto: any): string[] {
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

// ---------------------------------------------------------
// priceRows helpers
// ---------------------------------------------------------

/**
 * priceRows は dto.priceRows のみ採用
 *
 * PriceCard 正:
 * - apparel: size / color / stock / price
 * - alcohol: volumeValue / volumeUnit / stock / price
 *
 * 注意:
 * - kind / volumeValue / volumeUnit を落とさず PriceRow へ渡す
 */
export function normalizePriceRows<TRow extends Record<string, any> = any>(
  dto: any,
): TRow[] {
  const rowsRaw = Array.isArray(dto?.priceRows) ? dto.priceRows : [];

  return rowsRaw.map((r: any, idx: number) => {
    const modelId = s(r?.modelId);
    const displayOrder = toDisplayOrderOrNull(r?.displayOrder);

    const size = s(r?.size);
    const color = s(r?.color);

    const stock = toInt(r?.stock);
    const price = toNumberOrNull(r?.price);

    const rgbNum = toNumberOrNull(r?.rgb);
    const rgb = rgbNum === null ? undefined : rgbNum;

    const kind = s(r?.kind);
    const volumeValue = toNumberOrNull(r?.volumeValue ?? r?.volume?.value);
    const volumeUnit = s(r?.volumeUnit ?? r?.volume?.unit);

    const rowAny = {
      id: modelId || String(idx),
      modelId,
      displayOrder,
      size,
      color,
      rgb,
      stock,
      price,
      kind,
      volumeValue,
      volumeUnit,
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
    const modelId = s(r?.modelId) || s(r?.id);
    if (!modelId) continue;

    const price = toNumberOrNull(r?.price);
    if (price === null) continue;

    out.push({ modelId, price });
  }

  return out;
}

/**
 * PriceCard 編集時、price だけ更新する。
 * kind / volumeValue / volumeUnit / size / color / stock は row spread で保持する。
 */
export function updatePriceRowPrice<TRow extends Record<string, any>>(
  rows: TRow[] | null | undefined,
  index: number,
  price: number | null,
): TRow[] {
  const src = Array.isArray(rows) ? rows : [];

  return src.map((row, i) => {
    if (i !== index) return row;
    return { ...row, price };
  });
}

// ---------------------------------------------------------
// detail mapper
// ---------------------------------------------------------

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
  const createdAt = safeDateTimeLabelJa(s(dto?.createdAt), "");

  const updatedByName = s(dto?.updatedBy) || s((dto as any)?.updatedByName);
  const updatedAt = safeDateTimeLabelJa(s(dto?.updatedAt), "");

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

export function computeListDetailPageTitle(args: {
  listId?: string;
  listingTitle?: string;
}) {
  const id = s(args.listId);
  const t = s(args.listingTitle) || "出品詳細";

  return id ? `${t}（listId: ${id}）` : t;
}