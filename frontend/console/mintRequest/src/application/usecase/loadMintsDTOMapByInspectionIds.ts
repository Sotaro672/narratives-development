// frontend/console/mintRequest/src/application/usecase/loadMintsDTOMapByInspectionIds.ts

import type { MintDTO } from "../../infrastructure/api/mintRequestApi";
import { fetchMintsByInspectionIdsHTTP } from "../../infrastructure/repository";

/**
 * MintDTO を inspectionIds (= productionIds) でまとめて取得する。
 * - 詳細画面や “mint存在判定以外の情報” が必要になった場合のため
 */
export async function loadMintsDTOMapByInspectionIds(
  inspectionIds: string[],
): Promise<Record<string, MintDTO>> {
  const ids = (inspectionIds ?? [])
    .map((s) => String(s ?? "").trim())
    .filter((s) => !!s);

  if (ids.length === 0) return {};

  const m = await fetchMintsByInspectionIdsHTTP(ids);
  return (m ?? {}) as Record<string, MintDTO>;
}
