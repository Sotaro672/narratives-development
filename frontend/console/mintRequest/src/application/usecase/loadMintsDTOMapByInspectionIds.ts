// frontend/console/mintRequest/src/application/usecase/loadMintsDTOMapByInspectionIds.ts

import type { MintDTO } from "../../infrastructure/api/mintRequestApi";
import { fetchMintsByProductionIdsHTTP } from "../../infrastructure/repository";

/**
 * MintDTO を productionIds でまとめて取得する。
 * - 詳細画面や “mint存在判定以外の情報” が必要になった場合のため
 *
 * NOTE:
 * 旧 inspectionIds 名は使わず、productionIds を正とする。
 */
export async function loadMintsDTOMapByProductionIds(
  productionIds: string[],
): Promise<Record<string, MintDTO>> {
  const ids = (productionIds ?? [])
    .map((s) => String(s ?? "").trim())
    .filter((s) => !!s);

  if (ids.length === 0) return {};

  const m = await fetchMintsByProductionIdsHTTP(ids);
  return (m ?? {}) as Record<string, MintDTO>;
}