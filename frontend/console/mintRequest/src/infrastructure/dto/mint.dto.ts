// infrastructure/dto/mint.dto.ts
import type { Mint } from "../../domain/entity/mints";

export type MintDTO = Mint & {
  createdByName?: string | null;
  requestedByName?: string | null;
  onChainTxSignature?: string | null;
};

export type MintListRowDTO = {
  productionId?: string | null;
  mintId?: string | null;
  tokenBlueprintId?: string | null;
  tokenName: string;
  createdByName?: string | null;
  requestedByName?: string | null;
  mintedAt?: string | null;
  minted?: boolean;
};