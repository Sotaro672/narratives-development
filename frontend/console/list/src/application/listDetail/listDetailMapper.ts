// frontend/console/list/src/application/listDetail/listDetailMapper.ts

import { safeDateTimeLabelJa } from "../../../../shell/src/shared/util/dateJa";

export type ListingDecisionNorm = "listing" | "holding" | "";

// ---------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------

function dedupeUrlsKeepOrder(urls: string[]): string[] {
  const seen = new Set<string>();
  const out: string[] = [];

  for (const url of urls) {
    const normalizedUrl = String(url ?? "").trim();
    if (!normalizedUrl) continue;
    if (seen.has(normalizedUrl)) continue;

    seen.add(normalizedUrl);
    out.push(normalizedUrl);
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

  return Math.trunc(n);
}

// ---------------------------------------------------------
// Decision helpers
// ---------------------------------------------------------

/**
 * decision は backend response の decision を正とする。
 * - "listing" => "list"（出品）
 * - "hold"    => "hold"（保留）
 */
export function normalizeDecision(dto: any): string {
  const raw = String(dto?.decision ?? "").trim().toLowerCase();

  if (raw === "listing") return "list";
  if (raw === "hold") return "hold";

  return raw;
}

export function normalizeListingDecisionNorm(v: unknown): ListingDecisionNorm {
  const x = String(v ?? "").trim().toLowerCase();

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
  return safeDateTimeLabelJa(String(v ?? "").trim(), "");
}

// ---------------------------------------------------------
// imageUrls helpers
// ---------------------------------------------------------

/**
 * UI 用の imageUrls を生成
 *
 * 正:
 * - GET /lists/{id} response の imageUrls: string[] を採用
 * - listImages / listImage / objectPath 互換は扱わない
 */
export function normalizeImageUrls(dto: any): string[] {
  const imageUrls = Array.isArray(dto?.imageUrls) ? dto.imageUrls : [];

  return dedupeUrlsKeepOrder(
    imageUrls.map((url: any) => String(url ?? "").trim()).filter(Boolean),
  );
}

// ---------------------------------------------------------
// priceRows helpers
// ---------------------------------------------------------

/**
 * priceRows は dto.priceRows のみ採用
 *
 * 正:
 * - modelId
 * - displayOrder
 * - stock
 * - size
 * - color
 * - price
 */
export function normalizePriceRows<TRow extends Record<string, any> = any>(
  dto: any,
): TRow[] {
  const rowsRaw = Array.isArray(dto?.priceRows) ? dto.priceRows : [];

  return rowsRaw.map((r: any, idx: number) => {
    const modelId = String(r?.modelId ?? "").trim();
    const displayOrder = toDisplayOrderOrNull(r?.displayOrder);

    const size = String(r?.size ?? "").trim();
    const color = String(r?.color ?? "").trim();

    const stock = toInt(r?.stock);
    const price = toNumberOrNull(r?.price);

    const rowAny = {
      id: modelId || String(idx),
      modelId,
      displayOrder,
      stock,
      size,
      color,
      price,
    };

    return rowAny as unknown as TRow;
  });
}

/**
 * PriceCard 編集時、price だけ更新する。
 * size / color / stock は row spread で保持する。
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
  const listingTitle = String(dto?.title ?? "").trim();
  const description = String(dto?.description ?? "").trim();
  const decision = normalizeDecision(dto);

  const productBrandId = String(dto?.productBrandId ?? "").trim();
  const productBrandName = String(dto?.productBrandName ?? "").trim();
  const productName = String(dto?.productName ?? "").trim();

  const tokenBrandId = String(dto?.tokenBrandId ?? "").trim();
  const tokenBrandName = String(dto?.tokenBrandName ?? "").trim();
  const tokenName = String(dto?.tokenName ?? "").trim();

  const assigneeId = String(dto?.assigneeId ?? "").trim();
  const assigneeName = String(dto?.assigneeName ?? "").trim() || "未設定";

  const createdByName = String(dto?.createdByName ?? "").trim();
  const createdAt = safeDateTimeLabelJa(
    String(dto?.createdAt ?? "").trim(),
    "",
  );

  const updatedByName = String(dto?.updatedByName ?? "").trim();
  const updatedAt = safeDateTimeLabelJa(
    String(dto?.updatedAt ?? "").trim(),
    "",
  );

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
  const id = String(args.listId ?? "").trim();
  const title = String(args.listingTitle ?? "").trim() || "出品詳細";

  return id ? `${title}（listId: ${id}）` : title;
}