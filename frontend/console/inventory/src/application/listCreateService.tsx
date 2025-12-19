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
    const rgb = r?.rgb ?? r?.RGB; // number|null|undefined 想定
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

/**
 * ✅ POST /lists 用の payload を組み立てる（inventory の listCreate 画面入力 → list 作成）
 */
export function buildCreateListInput(args: {
  params: ResolvedListCreateParams;
  listingTitle: string;
  description: string;
  priceRows: PriceRow[];
  decision: "list" | "hold";
  assigneeId?: string;
}): CreateListInput {
  const title = s(args.listingTitle);
  const desc = s(args.description);

  return {
    inventoryId: s(args.params.inventoryId) || undefined,
    title,
    description: desc,
    decision: args.decision,
    assigneeId: s(args.assigneeId) || undefined,
    priceRows: (args.priceRows ?? []).map((r) => ({
      size: s(r.size) || "-",
      color: s(r.color) || "-",
      stock: Number.isFinite(Number(r.stock)) ? Number(r.stock) : 0,
      price: r.price === undefined ? null : (r.price as any),
      rgb: (r as any).rgb ?? null,
    })),
  };
}

/**
 * ✅ 入力バリデーション（UI 側の要件）
 * - title が空欄 → エラー
 * - price が 0（または未入力/0のみ） → エラー
 */
export function validateCreateListInput(input: CreateListInput): void {
  const title = s(input.title);
  if (!title) {
    throw new Error("タイトルを入力してください。");
  }

  const rows = Array.isArray(input.priceRows) ? input.priceRows : [];
  // 価格が1つも入っていない or 0 しか無い場合は NG
  const hasPositivePrice = rows.some((r: any) => {
    const p = r?.price;
    const n = typeof p === "number" ? p : Number(p);
    return Number.isFinite(n) && n > 0;
  });
  if (!hasPositivePrice) {
    throw new Error("価格を入力してください。（0 円は指定できません）");
  }

  // 念のため「0円」の行が混ざっていたらエラーにする（在庫>0 の行だけ見る等にしたければここを調整）
  const hasZeroPrice = rows.some((r: any) => {
    const p = r?.price;
    const n = typeof p === "number" ? p : Number(p);
    return Number.isFinite(n) && n === 0;
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
    inventoryId: input.inventoryId,
    title: input.title,
    descriptionLen: String(input.description ?? "").length,
    decision: input.decision,
    priceRowsCount: Array.isArray(input.priceRows) ? input.priceRows.length : 0,
    payload: input,
  });

  // ✅ validate
  validateCreateListInput(input);

  // eslint-disable-next-line no-console
  console.log("[inventory/listCreateService] postCreateList (validated)", {
    inventoryId: input.inventoryId,
    title: input.title,
    decision: input.decision,
    priceRowsCount: Array.isArray(input.priceRows) ? input.priceRows.length : 0,
  });

  return await createListHTTP(input);
}
