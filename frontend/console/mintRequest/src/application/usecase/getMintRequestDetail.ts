// frontend/console/mintRequest/src/application/usecase/getMintRequestDetail.ts

import type { MintRequestRepository } from "../port/MintRequestRepository";
import { asNonEmptyString } from "../util/primitive";

async function resolveProductBlueprintId(
  repo: MintRequestRepository,
  productionId: string,
): Promise<string> {
  const productBlueprintId =
    await repo.fetchProductBlueprintIdByProductionId(productionId);

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
      mintDTO: null,
      productBlueprintId: "",
    };
  }

  const [inspectionBatch, mintDTO, productBlueprintId] = await Promise.all([
    repo.fetchInspectionByProductionId(pid),
    repo.fetchMintByProductionId(pid),
    resolveProductBlueprintId(repo, pid),
  ]);

  return {
    inspectionBatch,
    mintDTO,
    productBlueprintId: productBlueprintId || "",
  };
}