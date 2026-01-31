//frontend\console\mintRequest\src\application\usecase\listTokenBlueprintsByBrand.ts
import type { MintRequestRepository } from "../port/MintRequestRepository";

export async function listTokenBlueprintsByBrand(repo: MintRequestRepository, brandId: string) {
  const id = String(brandId ?? "").trim();
  if (!id) return [];
  return await repo.fetchTokenBlueprintsByBrand(id);
}
