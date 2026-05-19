// frontend/console/mintRequest/src/infrastructure/dto/mint.dto.ts

import type { Mint } from "../../domain/entity/mints";

export type MintDTO = Mint & {
  /**
   * Backend management row では mint:boolean として返る。
   * Frontend 内部の判定では minted:boolean を使うため、
   * repository / mapper 側で mint -> minted に正規化する。
   */
  mint?: boolean | null;

  productionId?: string | null;
  inspectionId?: string | null;

  brandId?: string | null;
  tokenBlueprintId?: string | null;
  tokenName?: string | null;

  createdBy?: string | null;
  createdByName?: string | null;
  requestedBy?: string | null;
  requestedByName?: string | null;

  createdAt?: string | null;
  mintedAt?: string | null;
  scheduledBurnDate?: string | null;

  products?: string[];

  onChainTxSignature?: string | null;
};

export type MintListRowDTO = {
  productionId?: string | null;
  mintId?: string | null;

  /**
   * Backend management/list row 互換。
   */
  mint?: boolean | null;

  tokenBlueprintId?: string | null;
  tokenName: string;

  createdByName?: string | null;
  requestedByName?: string | null;

  createdAt?: string | null;
  mintedAt?: string | null;
};