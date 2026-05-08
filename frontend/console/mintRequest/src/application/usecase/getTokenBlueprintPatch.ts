//frontend\console\mintRequest\src\application\usecase\getTokenBlueprintPatch.ts
import type { MintRequestRepository } from "../port/MintRequestRepository";

export async function getTokenBlueprintPatch(repo: MintRequestRepository, tokenBlueprintId: string) {
  const id = String(tokenBlueprintId ?? "").trim();
  if (!id) return null;
  return await repo.fetchTokenBlueprintPatch(id);
}
