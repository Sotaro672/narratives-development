// frontend/console/inventory/src/application/inventoryDetail/inventoryDetail.query.ts

import {
  fetchInventoryIDsByProductAndTokenDTO,
  fetchInventoryDetailDTO,
  fetchTokenBlueprintPatchDTO,
  type InventoryDetailDTO,
  type TokenBlueprintPatchDTO,
} from "../../infrastructure/http/inventoryRepositoryHTTP";

import type { InventoryDetailViewModel } from "./inventoryDetail.types";
import { asString, uniqStrings } from "./inventoryDetail.utils";
import { mergeDetailDTOs } from "./inventoryDetail.mapper";

// ============================================================
// Query Request (Application Layer)
// - ✅ 方針Aのみ: pbId + tbId -> inventoryIds -> details -> merge
// - ✅ tokenBlueprint patch を追加で取得して ViewModel に載せる
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

  // ② 各 inventoryId の詳細を並列取得
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

  // ✅ tokenBlueprint patch を追加で取得（失敗しても detail 自体は返す）
  let tokenBlueprintPatch: TokenBlueprintPatchDTO | null = null;
  try {
    tokenBlueprintPatch = await fetchTokenBlueprintPatchDTO(tbId);
  } catch {
    tokenBlueprintPatch = null;
  }

  // ③ マージしてViewModel化
  return mergeDetailDTOs(pbId, tbId, inventoryIds, ok, tokenBlueprintPatch);
}
