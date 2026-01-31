// frontend/console/inventory/src/application/inventoryDetail/inventoryDetailService.tsx
// ✅ 統合版: query + barrel（既存 import を壊さない）
// ※ fallback / compatible は戻さず、トークンアイコン取得に必要な tokenBlueprintPatch だけ復旧

import {
  fetchInventoryIDsByProductAndTokenDTO,
  fetchInventoryDetailDTO,
  fetchTokenBlueprintPatchDTO,
  type InventoryDetailDTO,
  type TokenBlueprintPatchDTO,
} from "../../infrastructure/http/inventoryRepositoryHTTP";

import type { InventoryDetailViewModel } from "./inventoryDetail.types";
import { asString } from "./inventoryDetail.utils";
import { mergeDetailDTOs } from "./inventoryDetail.mapper";

// ------------------------------------------------------------
// Re-export types
// ------------------------------------------------------------
export type {
  ProductBlueprintPatchDTOEx,
  TokenBlueprintPatchDTOEx,
  InventoryDetailViewModel,
} from "./inventoryDetail.types";

// ============================================================
// Query Request (Application Layer)
// - pbId + tbId -> inventoryIds -> details -> merge
// - ✅ tokenBlueprintPatch を取得して iconUrl を拾えるようにする（最小復旧）
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
  const inventoryIds = Array.isArray((idsDto as any)?.inventoryIds)
    ? (idsDto as any).inventoryIds.map((x: unknown) => asString(x)).filter(Boolean)
    : [];

  if (inventoryIds.length === 0) {
    throw new Error(
      "inventoryIds is empty (no inventory for productBlueprintId + tokenBlueprintId)",
    );
  }

  // ② 各 inventoryId の詳細を並列取得
  const results: PromiseSettledResult<InventoryDetailDTO>[] = await Promise.allSettled(
    inventoryIds.map(async (id: string): Promise<InventoryDetailDTO> => {
      return await fetchInventoryDetailDTO(id);
    }),
  );

  const ok: InventoryDetailDTO[] = [];
  const failed: Array<{ id: string; reason: string }> = [];

  results.forEach((r, idx) => {
    const id = inventoryIds[idx];
    if (r.status === "fulfilled") ok.push(r.value);
    else failed.push({ id, reason: String((r.reason as any)?.message ?? r.reason) });
  });

  if (ok.length === 0) {
    throw new Error(
      `failed to fetch any inventory detail: ${failed.map((x) => x.id).join(", ")}`,
    );
  }

  // ✅ ここだけ復旧: tokenBlueprintPatch を取得（主に iconUrl 用）
  let tokenBlueprintPatch: TokenBlueprintPatchDTO | null = null;
  try {
    tokenBlueprintPatch = await fetchTokenBlueprintPatchDTO(tbId);
  } catch {
    tokenBlueprintPatch = null;
  }

  // ③ マージしてViewModel化（tokenBlueprintPatch を渡す）
  return mergeDetailDTOs(pbId, tbId, inventoryIds, ok, tokenBlueprintPatch);
}
