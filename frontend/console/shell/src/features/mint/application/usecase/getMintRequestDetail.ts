// frontend/console/shell/src/features/mintRequest/application/usecase/getMintRequestDetail.ts

import type { MintRequestRepository } from "../port/MintRequestRepository";
import { asNonEmptyString } from "../../../../shared/util/primitive";

async function resolveProductBlueprintId(
  repo: MintRequestRepository,
  productionId: string,
): Promise<string> {
  const productBlueprintId =
    await repo.fetchProductBlueprintIdByProductionId(
      productionId,
    );

  return asNonEmptyString(productBlueprintId);
}

export async function getMintRequestDetail(
  repo: MintRequestRepository,
  productionId: string,
) {
  const pid = String(productionId ?? "").trim();

  if (!pid) {
    return {
      inspectionBatch: null,
      mintRequestRow: null,
      productBlueprintId: "",
    };
  }

  const [
    inspectionBatch,
    mintRequestRow,
    productBlueprintId,
  ] = await Promise.all([
    repo.fetchInspectionByProductionId(pid),
    repo.fetchMintRequestRowByProductionId(pid),
    resolveProductBlueprintId(repo, pid),
  ]);

  return {
    inspectionBatch,
    mintRequestRow,
    productBlueprintId: productBlueprintId || "",
  };
}