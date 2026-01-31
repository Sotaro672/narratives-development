//frontend\console\mintRequest\src\application\usecase\getMintRequestDetail.ts
import type { MintRequestRepository } from "../port/MintRequestRepository";
import { asNonEmptyString } from "../mapper/modelInspectionMapper"; // ここがapplicationにある前提ならOK。presentationにあるなら移動推奨。

function extractProductBlueprintIdFromBatch(batch: any): string {
  if (!batch) return "";
  const v = batch.productBlueprintId ?? batch.productBlueprint?.id ?? "";
  return asNonEmptyString(v);
}

async function resolveProductBlueprintId(
  repo: MintRequestRepository,
  productionId: string,
  batch: any,
): Promise<string> {
  const pbFromBatch = extractProductBlueprintIdFromBatch(batch);
  if (pbFromBatch) return pbFromBatch;

  const pbFromProduction = await repo.fetchProductBlueprintIdByProductionId(productionId);
  return asNonEmptyString(pbFromProduction);
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

  const productBlueprintId = await resolveProductBlueprintId(repo, rid, inspectionBatch);

  return {
    inspectionBatch,
    mintDTO,
    productBlueprintId: productBlueprintId || "",
  };
}
