// frontend/console/inventory/src/application/inventoryDetail/inventoryDetail.mapper.ts

import type { InventoryRow } from "../inventoryTypes";
import type {
  InventoryDetailDTO,
  TokenBlueprintPatchDTO,
} from "../../infrastructure/http/inventoryRepositoryHTTP";
import type { InventoryDetailViewModel } from "./inventoryDetail.types";
import { asString } from "./inventoryDetail.utils";
import {
  pickPatch,
  pickTokenBlueprintPatch,
  pickUpdatedAtMax,
  pickBrandId,
  pickBrandName,
  pickProductName,
} from "./inventoryDetail.pickers";

// ============================================================
// Mapper (DTO row -> InventoryRow)
// ============================================================

export function mapDtoToRows(
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

export function mergeDetailDTOs(
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

  // ✅ tokenBlueprint patch を確定
  const tokenBlueprintPatch = pickTokenBlueprintPatch(dtos, tokenBlueprintPatchExternal);

  // ✅ token 名の fallback は tokenBlueprintPatch を使う（picker縮小に合わせて）
  const fallbackTokenName = asString((tokenBlueprintPatch as any)?.tokenName);

  // rows を結合 → 同一キーで合算（表示安定）
  const agg = new Map<
    string,
    { token?: string; modelNumber: string; size: string; color: string; rgb: any; stock: number }
  >();

  for (const d of dtos as any[]) {
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
