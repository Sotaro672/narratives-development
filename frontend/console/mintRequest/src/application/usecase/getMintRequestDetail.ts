// frontend/console/mintRequest/src/application/usecase/getMintRequestDetail.ts

import type { MintRequestRepository } from "../port/MintRequestRepository";
import { asNonEmptyString } from "../mapper/modelInspectionMapper";
import { extractProductBlueprintIdFromBatch } from "../mapper/productBlueprintIdMapper";

async function resolveProductBlueprintId(
  repo: MintRequestRepository,
  productionId: string,
  batch: unknown,
): Promise<string> {
  const productBlueprintIdFromBatch = extractProductBlueprintIdFromBatch(batch);

  if (productBlueprintIdFromBatch) {
    return productBlueprintIdFromBatch;
  }

  const productBlueprintIdFromProduction =
    await repo.fetchProductBlueprintIdByProductionId(productionId);

  return asNonEmptyString(productBlueprintIdFromProduction);
}

export async function getMintRequestDetail(
  repo: MintRequestRepository,
  requestId: string,
) {
  const rid = String(requestId ?? "").trim();

  if (!rid) {
    return {
      inspectionBatch: null,
      mintDTO: null,
      productBlueprintId: "",
    };
  }

  const [inspectionBatch, mintDTO] = await Promise.all([
    repo.fetchInspectionByProductionId(rid),
    repo.fetchMintByInspectionId(rid),
  ]);

  const productBlueprintId = await resolveProductBlueprintId(
    repo,
    rid,
    inspectionBatch,
  );

  return {
    inspectionBatch,
    mintDTO,
    productBlueprintId: productBlueprintId || "",
  };
}