// frontend/console/shell/src/features/mintRequest/infrastructure/dto/mint.dto.ts

import type { MintTaskProgress } from "../../application/port/MintRequestRepository";
import type { Mint } from "../../domain/mints";

export type MintDTO = Mint & {
  createdByName?: string | null;
  requestedByName?: string | null;
  onChainTxSignature?: string | null;
  mintProgress?: MintTaskProgress | null;
};