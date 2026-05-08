// frontend/console/mintRequest/src/infrastructure/normalizers/mintListRow.ts

import type { MintListRowDTO } from "../api/mintRequestApi";
import { asMaybeString } from "./string";

/**
 * list row normalize（hook 側の “正” を前提に最小限）
 * - inspectionId を “正” として揃える（productionId/id 揺れは rowKey 側で吸収）
 */
export function normalizeMintListRow(v: any): MintListRowDTO {
  const inspectionId =
    asMaybeString(v?.inspectionId ?? v?.productionId ?? v?.id) ?? null;

  const mintId = asMaybeString(v?.mintId ?? v?.id) ?? null;

  // ✅ tokenBlueprintId は lowerCamel を正として扱う（名揺れ吸収を削減）
  const tokenBlueprintId = asMaybeString(v?.tokenBlueprintId) ?? null;

  // ✅ tokenName も “tokenName” を正とする
  const tokenName = asMaybeString(v?.tokenName) ?? null;

  const createdByName = asMaybeString(v?.createdByName) ?? null;

  const mintedAt =
    typeof v?.mintedAt === "string" && v.mintedAt.trim() ? v.mintedAt.trim() : null;

  const minted = typeof v?.minted === "boolean" ? v.minted : Boolean(mintedAt);

  return {
    inspectionId,
    mintId,
    tokenBlueprintId,
    tokenName,
    createdByName,
    mintedAt,
    minted,
  } as any;
}
