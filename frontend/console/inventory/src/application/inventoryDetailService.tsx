// frontend/console/inventory/src/application/inventoryDetailService.tsx

import type { InventoryRow } from "../presentation/components/inventoryCard";
import {
  fetchInventoryIDsByProductAndTokenDTO,
  fetchInventoryDetailDTO,
  type InventoryDetailDTO,
  type ProductBlueprintPatchDTO,
} from "../infrastructure/http/inventoryRepositoryHTTP";

// ============================================================
// ViewModel (Screen-friendly shape)
// ============================================================

export type InventoryDetailViewModel = {
  /** 画面用の一意キー（pbId + tbId） */
  inventoryKey: string;

  /** 方針A: 詳細が対象とする inventoryId の集合 */
  inventoryIds: string[];

  tokenBlueprintId: string;
  productBlueprintId: string;

  /** 方針Aでは原則空 */
  modelId: string;

  productBlueprintPatch: ProductBlueprintPatchDTO;

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

function pickPatch(dtos: InventoryDetailDTO[]): ProductBlueprintPatchDTO {
  const found =
    dtos.find(
      (d) =>
        d?.productBlueprintPatch && Object.keys(d.productBlueprintPatch).length > 0,
    )?.productBlueprintPatch ?? {};
  return found as any;
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
    const rowTbId = asString(r?.tokenBlueprintId ?? r?.TokenBlueprintID ?? r?.token_blueprint_id);

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
): InventoryDetailViewModel {
  const patch = pickPatch(dtos);
  const maxUpdated = pickUpdatedAtMax(dtos);

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

    productBlueprintPatch: patch,
    rows: mergedRows,
    totalStock,

    updatedAt: maxUpdated,
  };
}

// ============================================================
// Query Request (Application Layer)
// - ✅ 方針Aのみ: pbId + tbId -> inventoryIds -> details -> merge
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
    throw new Error("inventoryIds is empty (no inventory for productBlueprintId + tokenBlueprintId)");
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
    throw new Error(`failed to fetch any inventory detail: ${failed.map((x) => x.id).join(", ")}`);
  }

  // ✅ backend DTO に inventoryIds が入ってくる場合は union
  const idsFromDTO: string[] = [];
  for (const d of ok as any[]) {
    const xs = Array.isArray(d?.inventoryIds) ? d.inventoryIds : [];
    for (const x of xs) idsFromDTO.push(asString(x));
  }

  const inventoryIds = uniqStrings([...idsFromResolver, ...idsFromDTO]);

  // ③ マージしてViewModel化
  return mergeDetailDTOs(pbId, tbId, inventoryIds, ok);
}
