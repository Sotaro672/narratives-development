//frontend\console\mintRequest\src\application\usecase\listBrandsForMint.ts
import type { MintRequestRepository } from "../port/MintRequestRepository";

export async function listBrandsForMint(repo: MintRequestRepository) {
  return await repo.fetchBrandsForMint();
}
