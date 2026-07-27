// frontend/console/mintRequest/src/infrastructure/dto/mint.dto.ts

import type { Mint } from "../../domain/mints";

export type MintDTO = Mint & {
  createdByName?: string | null;
  requestedByName?: string | null;
  onChainTxSignature?: string | null;
};
