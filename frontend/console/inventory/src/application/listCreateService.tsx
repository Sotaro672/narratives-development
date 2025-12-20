// frontend/console/inventory/src/application/listCreateService.tsx

import type { RefObject } from "react";

import type { PriceRow } from "../../../list/src/presentation/hook/usePriceCard";

import {
  fetchListCreateDTO,
  type ListCreateDTO,
} from "../infrastructure/http/inventoryRepositoryHTTP";

// ✅ NEW: list create (POST /lists)
import {
  createListHTTP,
  type CreateListInput,
  type ListDTO,
} from "../../../list/src/infrastructure/http/listRepositoryHTTP";

function s(v: unknown): string {
  return String(v ?? "").trim();
}

// ✅ Hook 側で使う Ref 型（useRef<HTMLInputElement | null>(null) を許容）
export type ImageInputRef = RefObject<HTMLInputElement | null>;

export type ListCreateRouteParams = {
  inventoryId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
};

export type ResolvedListCreateParams = {
  inventoryId: string;
  productBlueprintId: string;
  tokenBlueprintId: string;
  raw: ListCreateRouteParams;
};

export function resolveListCreateParams(
  raw: ListCreateRouteParams,
): ResolvedListCreateParams {
  return {
    inventoryId: s(raw?.inventoryId),
    productBlueprintId: s(raw?.productBlueprintId),
    tokenBlueprintId: s(raw?.tokenBlueprintId),
    raw,
  };
}

export function computeListCreateTitle(inventoryId: string): string {
  return inventoryId ? `出品作成（inventoryId: ${inventoryId}）` : "出品作成";
}

export function canFetchListCreate(p: ResolvedListCreateParams): boolean {
  return (
    Boolean(p.inventoryId) ||
    (Boolean(p.productBlueprintId) && Boolean(p.tokenBlueprintId))
  );
}

export function buildListCreateFetchInput(p: ResolvedListCreateParams): {
  inventoryId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
} {
  return {
    inventoryId: p.inventoryId || undefined,
    productBlueprintId: p.productBlueprintId || undefined,
    tokenBlueprintId: p.tokenBlueprintId || undefined,
  };
}

export function getInventoryIdFromDTO(dto: ListCreateDTO | null | undefined): string {
  return s((dto as any)?.inventoryId ?? (dto as any)?.InventoryID);
}

export function shouldRedirectToInventoryIdRoute(args: {
  currentInventoryId: string;
  gotInventoryId: string;
  alreadyRedirected: boolean;
}): boolean {
  return !args.alreadyRedirected && !args.currentInventoryId && Boolean(args.gotInventoryId);
}

export function buildInventoryDetailPath(pbId: string, tbId: string): string {
  const pb = s(pbId);
  const tb = s(tbId);
  if (!pb || !tb) return "/inventory";
  return `/inventory/detail/${encodeURIComponent(pb)}/${encodeURIComponent(tb)}`;
}

export function buildInventoryListCreatePath(inventoryId: string): string {
  const id = s(inventoryId);
  if (!id) return "/inventory/list/create";
  return `/inventory/list/create/${encodeURIComponent(id)}`;
}

export function buildBackPath(p: ResolvedListCreateParams): string {
  // ✅ 詳細へは pb/tb で戻す
  if (p.productBlueprintId && p.tokenBlueprintId) {
    return buildInventoryDetailPath(p.productBlueprintId, p.tokenBlueprintId);
  }
  return "/inventory";
}

export function buildAfterCreatePath(p: ResolvedListCreateParams): string {
  // ✅ 作成後も pb/tb があれば detail へ
  if (p.productBlueprintId && p.tokenBlueprintId) {
    return buildInventoryDetailPath(p.productBlueprintId, p.tokenBlueprintId);
  }
  return "/inventory";
}

export function extractDisplayStrings(dto: ListCreateDTO | null): {
  productBrandName: string;
  productName: string;
  tokenBrandName: string;
  tokenName: string;
} {
  return {
    productBrandName: s(dto?.productBrandName),
    productName: s(dto?.productName),
    tokenBrandName: s(dto?.tokenBrandName),
    tokenName: s(dto?.tokenName),
  };
}

/**
 * ✅ backend の ListCreateDTO.priceRows を PriceCard 用 PriceRow[] に変換
 * - dto 側に priceRows が無ければ []
 *
 * ※ ここは UI 用のため、size/color/stock/rgb を残してOK
 *    （POST /lists に送る形は buildCreateListInput で「modelId+price のみ」に射影する）
 */
export function mapDTOToPriceRows(dto: ListCreateDTO | null): PriceRow[] {
  const rowsAny: any[] = Array.isArray((dto as any)?.priceRows)
    ? ((dto as any).priceRows as any[])
    : Array.isArray((dto as any)?.PriceRows)
      ? ((dto as any).PriceRows as any[])
      : [];

  return rowsAny.flatMap((r: any) => {
    const size = s(r?.size ?? r?.Size) || "-";
    const color = s(r?.color ?? r?.Color) || "-";
    const stock = Number(r?.stock ?? r?.Stock ?? 0);
    const rgb = r?.rgb ?? r?.RGB; // number|string|null|undefined 想定
    const price = r?.price ?? r?.Price;

    const safeStock = Number.isFinite(stock) ? stock : 0;

    const row: PriceRow = {
      size,
      color,
      stock: safeStock,
      rgb: rgb as any,
      price: price === undefined ? null : (price as any),
    };

    return [row];
  });
}

/**
 * ✅ ListCreateDTO を取得する（Hook からはこれだけ呼ぶ）
 */
export async function loadListCreateDTOFromParams(
  p: ResolvedListCreateParams,
): Promise<ListCreateDTO> {
  const input = buildListCreateFetchInput(p);
  return await fetchListCreateDTO(input);
}

// ============================================================
// ✅ POST /lists: 期待値どおり「modelId + price のみ」
// ============================================================

export type CreateListPriceRow = {
  modelId: string;
  price: number | null;
};

function toNumberOrNull(v: unknown): number | null {
  if (v === null || v === undefined) return null;
  const n = typeof v === "number" ? v : Number(String(v).trim());
  if (!Number.isFinite(n)) return null;
  return Math.floor(n);
}

/**
 * ✅ Hook から渡された priceRows を「modelId+price」に正規化する
 * - Hook 側が PriceRowEx を渡してくる想定（modelId を含む）
 * - 互換のため ModelID なども拾う
 * - size/color/stock/rgb 等は POST には一切含めない
 */
export function normalizeCreateListPriceRows(rows: any[]): CreateListPriceRow[] {
  const arr = Array.isArray(rows) ? rows : [];
  return arr.map((r) => {
    const modelId = s((r as any)?.modelId ?? (r as any)?.ModelID);
    const price = toNumberOrNull((r as any)?.price);
    return { modelId, price };
  });
}

/**
 * ✅ POST /lists 用の payload を組み立てる
 * - 期待値：listRepositoryHTTP.tsx へは {modelId, price} のみ渡す
 */
export function buildCreateListInput(args: {
  params: ResolvedListCreateParams;
  listingTitle: string;
  description: string;
  // ✅ Hook 側が PriceRowEx（modelId含む）を渡してくるので any[] で受ける
  priceRows: any[];
  decision: "list" | "hold";
  assigneeId?: string;
}): CreateListInput {
  const title = s(args.listingTitle);
  const desc = s(args.description);

  const priceRows = normalizeCreateListPriceRows(args.priceRows);

  return {
    inventoryId: s(args.params.inventoryId) || undefined,
    title,
    description: desc,
    decision: args.decision,
    assigneeId: s(args.assigneeId) || undefined,
    // ✅ ここが重要：modelId と price 以外は送らない
    priceRows: priceRows as any,
  } as CreateListInput;
}

/**
 * ✅ 入力バリデーション（UI 側の要件）
 * - title が空欄 → エラー
 * - modelId が欠ける行がある → エラー
 * - price が 0（または未入力/0のみ） → エラー
 */
export function validateCreateListInput(input: CreateListInput): void {
  const title = s((input as any)?.title);
  if (!title) {
    throw new Error("タイトルを入力してください。");
  }

  const rows = Array.isArray((input as any)?.priceRows) ? (input as any).priceRows : [];
  if (rows.length === 0) {
    throw new Error("価格が未設定です（価格行がありません）。");
  }

  const missingModelId = rows.find((r: any) => !s(r?.modelId ?? r?.ModelID));
  if (missingModelId) {
    throw new Error("価格行に modelId が含まれていません。");
  }

  // 価格が1つも入っていない or 0 しか無い場合は NG
  const hasPositivePrice = rows.some((r: any) => {
    const n = toNumberOrNull(r?.price);
    return n !== null && n > 0;
  });
  if (!hasPositivePrice) {
    throw new Error("価格を入力してください。（0 円は指定できません）");
  }

  // 念のため「0円」の行が混ざっていたらエラー
  const hasZeroPrice = rows.some((r: any) => {
    const n = toNumberOrNull(r?.price);
    return n !== null && n === 0;
  });
  if (hasZeroPrice) {
    throw new Error("価格に 0 円が含まれています。0 円は指定できません。");
  }
}

/**
 * ✅ list 作成（POST /lists）
 * - listRepositoryHTTP.tsx 経由で Backend へ POST
 */
export async function postCreateList(input: CreateListInput): Promise<ListDTO> {
  // eslint-disable-next-line no-console
  console.log("[inventory/listCreateService] postCreateList (before validate)", {
    inventoryId: (input as any).inventoryId,
    title: (input as any).title,
    descriptionLen: String((input as any).description ?? "").length,
    decision: (input as any).decision,
    priceRowsCount: Array.isArray((input as any).priceRows) ? (input as any).priceRows.length : 0,
    // ✅ payload は大きいので、先頭だけ確認できるようにする
    priceRowsSample: Array.isArray((input as any).priceRows) ? (input as any).priceRows.slice(0, 5) : [],
  });

  // ✅ validate
  validateCreateListInput(input);

  // eslint-disable-next-line no-console
  console.log("[inventory/listCreateService] postCreateList (validated)", {
    inventoryId: (input as any).inventoryId,
    title: (input as any).title,
    decision: (input as any).decision,
    priceRowsCount: Array.isArray((input as any).priceRows) ? (input as any).priceRows.length : 0,
  });

  return await createListHTTP(input);
}
