// frontend/console/shell/src/features/mintRequest/application/usecase/getMintRequestProductBlueprintPatch.ts

import type {
  MintRequestRepository,
} from "../port/MintRequestRepository";

import {
  asNonEmptyString,
} from "../../../../shared/util/primitive";

import type {
  ProductBlueprintPatchDTO,
} from "../../infrastructure/dto/mintRequestLocal.dto";

export type MintRequestProductBlueprintPatchRepository =
  Pick<
    MintRequestRepository,
    "fetchProductBlueprintPatch"
  >;

/**
 * Mint申請詳細で使用するProduct Blueprint情報を取得する。
 *
 * productBlueprintIdが空の場合は通信を行わずnullを返す。
 * Repositoryから発生した通信エラーは呼び出し元へthrowする。
 */
export async function getMintRequestProductBlueprintPatch(
  repository:
    MintRequestProductBlueprintPatchRepository,
  productBlueprintId:
    | string
    | null
    | undefined,
): Promise<
  ProductBlueprintPatchDTO | null
> {
  const normalizedProductBlueprintId =
    asNonEmptyString(
      productBlueprintId,
    );

  if (
    !normalizedProductBlueprintId
  ) {
    return null;
  }

  return repository
    .fetchProductBlueprintPatch(
      normalizedProductBlueprintId,
    );
}