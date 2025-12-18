// frontend/console/inventory/src/application/inventoryDetailService.tsx

import type { InventoryRow } from "../presentation/components/inventoryCard";
import {
  fetchInventoryIDsByProductAndTokenDTO,
  fetchInventoryDetailDTO,
  // ✅ NEW: tokenBlueprint patch を取得する関数
  fetchTokenBlueprintPatchDTO,
  type InventoryDetailDTO,
  type ProductBlueprintPatchDTO,
  // ✅ NEW: tokenBlueprint patch DTO 型
  type TokenBlueprintPatchDTO,
} from "../infrastructure/http/inventoryRepositoryHTTP";

// ============================================================
// ViewModel (Screen-friendly shape)
// ============================================================

// DTO 側に brandName が増えても落とさないための拡張型（UIで参照しやすくする）
export type ProductBlueprintPatchDTOEx = ProductBlueprintPatchDTO & {
  brandId?: string;
  brandName?: string;
  productName?: string;
};

// ✅ NEW: tokenBlueprint patch を ViewModel に保持できるようにする
export type TokenBlueprintPatchDTOEx = TokenBlueprintPatchDTO & {
  tokenName?: string;
  brandId?: string;
  brandName?: string;
};

export type InventoryDetailViewModel = {
  /** 画面用の一意キー（pbId + tbId） */
  inventoryKey: string;

  /** 方針A: 詳細が対象とする inventoryId の集合 */
  inventoryIds: string[];

  tokenBlueprintId: string;
  productBlueprintId: string;

  /** 方針Aでは原則空 */
  modelId: string;

  // ✅ 画面でそのまま表示できるように ViewModel 直下にも持つ（重要）
  productName?: string;
  brandId?: string;
  brandName?: string;

  // ✅ NEW: tokenBlueprint patch（token名など）
  tokenBlueprintPatch?: TokenBlueprintPatchDTOEx;

  // 元データも保持（編集フォームなどで利用する想定）
  productBlueprintPatch: ProductBlueprintPatchDTOEx;

  rows: InventoryRow[];
  totalStock: number;

  /** max(updatedAt) */
  updatedAt?: string;
};

// ============================================================
// helpers
// ============================================================

function asString(v: unknown): string {
  return String(v ?? "").trim();
}

function uniqStrings(xs: string[]): string[] {
  const out: string[] = [];
  const seen = new Set<string>();
  for (const x of xs) {
    const s = asString(x);
    if (!s) continue;
    if (seen.has(s)) continue;
    seen.add(s);
    out.push(s);
  }
  return out;
}

function pickPatch(dtos: InventoryDetailDTO[]): ProductBlueprintPatchDTOEx {
  const found =
    dtos.find(
      (d) =>
        d?.productBlueprintPatch && Object.keys(d.productBlueprintPatch).length > 0,
    )?.productBlueprintPatch ?? {};

  // brandName 等の “増えがちなフィールド” を any 経由でも保持する
  return (found ?? {}) as any;
}

// ✅ NEW: tokenBlueprint patch を DTO群 or 外部取得結果から拾う
function pickTokenBlueprintPatch(
  dtos: InventoryDetailDTO[],
  external?: TokenBlueprintPatchDTO | null,
): TokenBlueprintPatchDTOEx | undefined {
  // 1) DTO 内（embedded）を優先…ただし「実体が薄い / 期待キーが無い」場合があるので注意して扱う
  const embedded =
    (dtos.find(
      (d: any) => d?.tokenBlueprintPatch && Object.keys(d.tokenBlueprintPatch).length > 0,
    ) as any)?.tokenBlueprintPatch ?? undefined;

  const embeddedTokenName = asString((embedded as any)?.tokenName ?? (embedded as any)?.name);
  const embeddedSymbol = asString((embedded as any)?.symbol);

  // embedded が “あるが中身が空っぽ” っぽい場合は external を優先
  const shouldPreferExternal =
    (!!embedded && !embeddedTokenName && !embeddedSymbol) || embedded === null;

  const base = (shouldPreferExternal ? (external ?? embedded) : (embedded ?? external)) as any;
  if (!base) return undefined;

  return {
    ...base,
    tokenName: asString(base?.tokenName ?? base?.name) || undefined,
    brandId: asString(base?.brandId) || undefined,
    brandName: asString(base?.brandName) || undefined,
  } as any;
}

function pickTokenNameFromDTO(dto: any): string {
  return (
    asString(dto?.tokenBlueprint?.name) ||
    asString(dto?.TokenBlueprint?.name) ||
    asString(dto?.tokenBlueprintName) ||
    ""
  );
}

function pickUpdatedAtMax(dtos: InventoryDetailDTO[]): string | undefined {
  let maxUpdated: string | undefined = undefined;
  for (const d of dtos as any[]) {
    const t = d?.updatedAt ? String(d.updatedAt) : "";
    if (!t) continue;
    if (!maxUpdated || t > maxUpdated) maxUpdated = t;
  }
  return maxUpdated;
}

// Patch から “ありがちな揺れ” を吸収して brandId/brandName を抜く
function pickBrandId(patch: any): string {
  return (
    asString(patch?.brandId) ||
    asString(patch?.BrandID) ||
    asString(patch?.BrandId) ||
    ""
  );
}
function pickBrandName(patch: any): string {
  return asString(patch?.brandName) || asString(patch?.BrandName) || "";
}
function pickProductName(patch: any): string {
  return asString(patch?.productName) || asString(patch?.ProductName) || "";
}

// ============================================================
// Mapper (DTO row -> InventoryRow)
// ============================================================

function mapDtoToRows(
  dto: InventoryDetailDTO,
  opts?: { expectedTokenBlueprintId?: string; fallbackTokenName?: string },
): InventoryRow[] {
  const expectedTbId = asString(opts?.expectedTokenBlueprintId);
  const fallbackToken = asString(opts?.fallbackTokenName);

  const rowsRaw: any[] = Array.isArray((dto as any)?.rows) ? ((dto as any).rows as any[]) : [];
  const out: InventoryRow[] = [];

  for (const r of rowsRaw) {
    // row 側に tbId が入っていない実装も多いので、ある時だけフィルタする
    const rowTbId = asString(
      r?.tokenBlueprintId ?? r?.TokenBlueprintID ?? r?.token_blueprint_id,
    );
    if (expectedTbId && rowTbId && rowTbId !== expectedTbId) continue;

    const token = asString(r?.token ?? r?.Token) || fallbackToken || "-";

    out.push({
      token,
      modelNumber: asString(r?.modelNumber ?? r?.ModelNumber ?? ""),
      size: asString(r?.size ?? r?.Size ?? ""),
      color: asString(r?.color ?? r?.Color ?? ""),
      rgb: (r?.rgb ?? r?.RGB ?? null) as any,
      stock: Number(r?.stock ?? r?.Stock ?? 0),
    });
  }

  return out;
}

// ============================================================
// Aggregator (multiple DTOs -> one ViewModel)
// ============================================================

function mergeDetailDTOs(
  pbId: string,
  tbId: string,
  inventoryIds: string[],
  dtos: InventoryDetailDTO[],
  tokenBlueprintPatchExternal?: TokenBlueprintPatchDTO | null,
): InventoryDetailViewModel {
  const patch = pickPatch(dtos);
  const maxUpdated = pickUpdatedAtMax(dtos);

  // ✅ 画面で表示しやすいように直下にも展開
  const brandId = pickBrandId(patch);
  const brandName = pickBrandName(patch);
  const productName = pickProductName(patch);

  // ✅ NEW: tokenBlueprint patch を確定
  const tokenBlueprintPatch = pickTokenBlueprintPatch(dtos, tokenBlueprintPatchExternal);

  // rows を結合 → 同一キーで合算（表示安定）
  const agg = new Map<
    string,
    { token?: string; modelNumber: string; size: string; color: string; rgb: any; stock: number }
  >();

  for (const d of dtos as any[]) {
    const fallbackTokenName = pickTokenNameFromDTO(d);

    for (const r of mapDtoToRows(d as any, { expectedTokenBlueprintId: tbId, fallbackTokenName })) {
      const token = asString(r.token) || "-";
      const modelNumber = asString(r.modelNumber) || "-";
      const size = asString(r.size) || "-";
      const color = asString(r.color) || "-";
      const rgbKey = r.rgb == null ? "nil" : String(r.rgb);

      const key = `${token}__${modelNumber}__${size}__${color}__${rgbKey}`;

      const cur = agg.get(key);
      if (!cur) {
        agg.set(key, {
          token,
          modelNumber,
          size,
          color,
          rgb: r.rgb ?? null,
          stock: Number(r.stock ?? 0),
        });
      } else {
        cur.stock += Number(r.stock ?? 0);
      }
    }
  }

  const mergedRows: InventoryRow[] = Array.from(agg.values()).map((v) => ({
    token: v.token,
    modelNumber: v.modelNumber,
    size: v.size,
    color: v.color,
    rgb: v.rgb,
    stock: v.stock,
  }));

  const totalStock = mergedRows.reduce((sum, r) => sum + Number(r.stock ?? 0), 0);

  mergedRows.sort((a, b) => {
    const as = (x: any) => String(x ?? "");
    if (as(a.token) !== as(b.token)) return as(a.token).localeCompare(as(b.token));
    if (as(a.modelNumber) !== as(b.modelNumber))
      return as(a.modelNumber).localeCompare(as(b.modelNumber));
    if (as(a.size) !== as(b.size)) return as(a.size).localeCompare(as(b.size));
    if (as(a.color) !== as(b.color)) return as(a.color).localeCompare(as(b.color));
    return as(a.rgb).localeCompare(as(b.rgb));
  });

  return {
    inventoryKey: `${pbId}__${tbId}`,
    inventoryIds,

    tokenBlueprintId: tbId,
    productBlueprintId: pbId,
    modelId: "",

    productName: productName || undefined,
    brandId: brandId || undefined,
    brandName: brandName || undefined,

    tokenBlueprintPatch,

    productBlueprintPatch: patch,
    rows: mergedRows,
    totalStock,

    updatedAt: maxUpdated,
  };
}

// ============================================================
// Query Request (Application Layer)
// - ✅ 方針Aのみ: pbId + tbId -> inventoryIds -> details -> merge
// - ✅ NEW: tokenBlueprint patch を追加で取得して ViewModel に載せる
// ============================================================

export async function queryInventoryDetailByProductAndToken(
  productBlueprintId: string,
  tokenBlueprintId: string,
): Promise<InventoryDetailViewModel> {
  const pbId = asString(productBlueprintId);
  const tbId = asString(tokenBlueprintId);

  if (!pbId) throw new Error("productBlueprintId is empty");
  if (!tbId) throw new Error("tokenBlueprintId is empty");

  // ① inventoryIds 解決
  const idsDto = await fetchInventoryIDsByProductAndTokenDTO(pbId, tbId);
  const idsFromResolver = Array.isArray((idsDto as any)?.inventoryIds)
    ? (idsDto as any).inventoryIds.map((x: unknown) => asString(x)).filter(Boolean)
    : [];

  if (idsFromResolver.length === 0) {
    throw new Error(
      "inventoryIds is empty (no inventory for productBlueprintId + tokenBlueprintId)",
    );
  }

  // ② 各 inventoryId の詳細を並列取得（HTTPは repository 側へ移譲済み）
  const results: PromiseSettledResult<InventoryDetailDTO>[] = await Promise.allSettled(
    idsFromResolver.map(async (id: string): Promise<InventoryDetailDTO> => {
      return await fetchInventoryDetailDTO(id);
    }),
  );

  const ok: InventoryDetailDTO[] = [];
  const failed: Array<{ id: string; reason: string }> = [];

  results.forEach((r, idx) => {
    const id = idsFromResolver[idx];
    if (r.status === "fulfilled") ok.push(r.value);
    else failed.push({ id, reason: String((r.reason as any)?.message ?? r.reason) });
  });

  if (ok.length === 0) {
    throw new Error(
      `failed to fetch any inventory detail: ${failed.map((x) => x.id).join(", ")}`,
    );
  }

  // ✅ backend DTO に inventoryIds が入ってくる場合は union
  const idsFromDTO: string[] = [];
  for (const d of ok as any[]) {
    const xs = Array.isArray(d?.inventoryIds) ? d.inventoryIds : [];
    for (const x of xs) idsFromDTO.push(asString(x));
  }

  const inventoryIds = uniqStrings([...idsFromResolver, ...idsFromDTO]);

  // ✅ NEW: tokenBlueprint patch を追加で取得
  // - endpoint が未実装/失敗しても inventory detail 自体は返す（optional）
  let tokenBlueprintPatch: TokenBlueprintPatchDTO | null = null;
  try {
    tokenBlueprintPatch = await fetchTokenBlueprintPatchDTO(tbId);
  } catch {
    tokenBlueprintPatch = null;
  }

  // ③ マージしてViewModel化
  const vm = mergeDetailDTOs(pbId, tbId, inventoryIds, ok, tokenBlueprintPatch);

  return vm;
}
