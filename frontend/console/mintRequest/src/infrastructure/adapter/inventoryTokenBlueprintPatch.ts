// frontend/console/mintRequest/src/infrastructure/adapter/inventoryTokenBlueprintPatch.ts

import { fetchInventoryDetailDTO } from "../../../../inventory/src/infrastructure/http/inventoryRepositoryHTTP.fetchers";
import type { TokenBlueprintPatchDTO } from "../../../../inventory/src/infrastructure/http/inventoryRepositoryHTTP";

/**
 * Inventory 側の tokenBlueprintPatch 取得を MintRequest モジュールにアダプトする。
 * - application 層から "inventory 側の import" を排除するための薄い wrapper
 * - GET /inventory/{inventoryId} に統一する
 * - GET /token-blueprints/{tokenBlueprintId}/patch は呼ばない
 *
 * NOTE:
 * - inventoryId は `${productBlueprintId}__${tokenBlueprintId}` 形式を想定
 * - tokenBlueprintPatch は GET /inventory/{inventoryId} の response に含まれるものを正とする
 */
export async function fetchInventoryTokenBlueprintPatch(
  inventoryId: string,
): Promise<TokenBlueprintPatchDTO | null> {
  const id = String(inventoryId ?? "").trim();
  if (!id) return null;

  const detail = await fetchInventoryDetailDTO(id);
  return detail.tokenBlueprintPatch ?? null;
}

export type { TokenBlueprintPatchDTO };