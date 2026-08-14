// frontend/console/shell/src/features/mint/application/usecase/getMintRequestDetail.ts

import type { MintRequestRepository } from "../port/MintRequestRepository";
import type { MintRequestDetailDTO } from "../../infrastructure/dto/mintRequestLocal.dto";

export type MintRequestDetailRepository = Pick<
  MintRequestRepository,
  "fetchMintRequestDetail"
>;

/**
 * productionIdに紐づくMint詳細BFFを取得する。
 *
 * BackendのGET /mint/inspections/{productionId} responseを正とし、
 * Application層ではinspectionBatchやmintRequestRowへの再構築を行わない。
 */
export async function getMintRequestDetail(
  repo: MintRequestDetailRepository,
  productionId: string,
): Promise<MintRequestDetailDTO | null> {
  const pid = String(productionId ?? "").trim();
  if (!pid) return null;

  return repo.fetchMintRequestDetail(pid);
}