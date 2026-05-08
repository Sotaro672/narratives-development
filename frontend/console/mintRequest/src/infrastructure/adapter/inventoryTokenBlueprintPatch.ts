// frontend/console/mintRequest/src/infrastructure/adapter/inventoryTokenBlueprintPatch.ts

import {
  fetchTokenBlueprintPatchDTO,
  type TokenBlueprintPatchDTO,
} from "../../../../inventory/src/infrastructure/http/inventoryRepositoryHTTP";

/**
 * Inventory 側の tokenBlueprintPatch 取得を MintRequest モジュールにアダプトする。
 * - application 層から "inventory 側の import" を排除するための薄い wrapper
 */
export async function fetchInventoryTokenBlueprintPatch(
  tokenBlueprintId: string,
): Promise<TokenBlueprintPatchDTO | null> {
  const id = String(tokenBlueprintId ?? "").trim();
  if (!id) return null;

  const patch = await fetchTokenBlueprintPatchDTO(id);
  return (patch ?? null) as TokenBlueprintPatchDTO | null;
}

export type { TokenBlueprintPatchDTO };
