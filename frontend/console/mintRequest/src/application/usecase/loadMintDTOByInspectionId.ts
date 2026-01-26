// frontend/console/mintRequest/src/application/usecase/loadMintDTOByInspectionId.ts

import type { MintDTO } from "../../infrastructure/api/mintRequestApi";
import { fetchMintByInspectionIdHTTP } from "../../infrastructure/repository";

/**
 * MintDTO を inspectionId (= productionId) で 1 件取得する。
 */
export async function loadMintDTOByInspectionId(
  inspectionId: string,
): Promise<MintDTO | null> {
  const iid = String(inspectionId ?? "").trim();
  if (!iid) return null;

  const m = await fetchMintByInspectionIdHTTP(iid);
  return (m ?? null) as MintDTO | null;
}
