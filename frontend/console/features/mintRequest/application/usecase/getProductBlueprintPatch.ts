//frontend\console\mintRequest\src\application\usecase\getProductBlueprintPatch.ts
import type { MintRequestRepository } from "../port/MintRequestRepository";

export async function getProductBlueprintPatch(repo: MintRequestRepository, productBlueprintId: string) {
  const id = String(productBlueprintId ?? "").trim();
  if (!id) return null;
  return await repo.fetchProductBlueprintPatch(id);
}
