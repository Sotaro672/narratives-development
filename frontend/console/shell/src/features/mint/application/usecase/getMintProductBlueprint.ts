// frontend/console/shell/src/features/mint/application/usecase/getMintProductBlueprint.ts

import type { MintRequestRepository } from "../port/MintRequestRepository";
import { asNonEmptyString } from "../../../../shared/util/primitive";
import type { MintProductBlueprintDTO } from "../../infrastructure/dto/mintRequestLocal.dto";

export type MintProductBlueprintRepository = Pick<
  MintRequestRepository,
  "fetchMintProductBlueprint"
>;

/**
 * Mint申請詳細で使用するProduct Blueprint情報を取得する。
 *
 * productBlueprintIdが空の場合は通信を行わずnullを返す。
 * Repositoryから発生した通信エラーは呼び出し元へthrowする。
 */
export async function getMintProductBlueprint(
  repository: MintProductBlueprintRepository,
  productBlueprintId: string | null | undefined,
): Promise<MintProductBlueprintDTO | null> {
  const normalizedProductBlueprintId = asNonEmptyString(productBlueprintId);

  if (!normalizedProductBlueprintId) {
    return null;
  }

  return repository.fetchMintProductBlueprint(normalizedProductBlueprintId);
}