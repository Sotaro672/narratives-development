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

function resolveTokenBlueprintId(input: unknown): string {
  return (
    asNonEmptyString((input as any)?.tokenBlueprintId) ||
    asNonEmptyString((input as any)?.tokenBlueprintID) ||
    ""
  );
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
      productBlueprintPatch: null,
      tokenBlueprintPatch: null,
    };
  }

  const [inspectionBatch, mintDTO, productBlueprintId] = await Promise.all([
    repo.fetchInspectionByProductionId(pid),
    repo.fetchMintByProductionId(pid),
    resolveProductBlueprintId(repo, pid),
  ]);

  const pbId = productBlueprintId || "";

  const productBlueprintPatch = pbId
    ? await repo.fetchProductBlueprintPatch(pbId)
    : null;

  const tokenBlueprintId = resolveTokenBlueprintId(mintDTO);

  const tokenBlueprintPatch = tokenBlueprintId
    ? await repo.fetchTokenBlueprintPatch(tokenBlueprintId)
    : null;

  return {
    inspectionBatch,
    mintDTO,
    productBlueprintId: pbId,
    productBlueprintPatch,
    tokenBlueprintPatch,
  };
}