// frontend/console/mintRequest/src/application/mapper/mintInfoMapper.ts

import type { InspectionBatchDTO } from "../../domain/entity/inspections";
import type { MintDTO } from "../../infrastructure/api/mintRequestApi";
import {
  asNonEmptyString,
  asMaybeISO,
} from "../util/primitive";

// ============================================================
// Types
// ============================================================

export type MintInfo = {
  id: string;

  brandId: string;
  tokenBlueprintId: string;
  requestedByName?: string | null;
  createdBy: string;
  createdByName?: string | null;
  createdAt: string | null;

  mint: boolean;
  mintedAt?: string | null;
  onChainTxSignature?: string | null;
  scheduledBurnDate?: string | null;
};

// ============================================================
// mapper
// ============================================================

/**
 * MintDTO（優先）から MintInfo を抽出。
 *
 * NOTE:
 * backend domain mint は minted:boolean、
 * /mint/requests?view=management の軽量 BFF row は mint:boolean を返す。
 * Frontend 内部では minted:boolean を正として扱うため、
 * mint:true も minted:true として正規化する。
 */
export function extractMintInfoFromMintDTO(m: MintDTO | any): MintInfo | null {
  if (!m) return null;

  const id =
    asNonEmptyString((m as any).id) ||
    asNonEmptyString((m as any).productionId) ||
    asNonEmptyString((m as any).inspectionId);

  if (!id) return null;

  const tokenBlueprintId = asNonEmptyString((m as any).tokenBlueprintId);
  const brandId = asNonEmptyString((m as any).brandId);

  const requestedByName = asNonEmptyString((m as any).requestedByName);

  const createdBy =
    asNonEmptyString((m as any).createdBy) ||
    asNonEmptyString((m as any).requestedBy);

  const createdByName = asNonEmptyString((m as any).createdByName);

  const createdAtStr = asNonEmptyString(asMaybeISO((m as any).createdAt));
  const createdAt = createdAtStr ? createdAtStr : null;

  const mintedAtStr = asNonEmptyString(asMaybeISO((m as any).mintedAt));

  const mint =
    typeof (m as any).minted === "boolean"
      ? (m as any).minted
      : typeof (m as any).mint === "boolean"
        ? (m as any).mint
        : Boolean(mintedAtStr);

  const onChainTxSignature = asNonEmptyString((m as any).onChainTxSignature);
  const scheduledBurnDate = asNonEmptyString(
    asMaybeISO((m as any).scheduledBurnDate),
  );

  return {
    id,
    brandId,
    tokenBlueprintId,
    requestedByName: requestedByName ? requestedByName : null,
    createdBy,
    createdByName: createdByName ? createdByName : null,
    createdAt,
    mint,
    mintedAt: mintedAtStr ? mintedAtStr : null,
    onChainTxSignature: onChainTxSignature ? onChainTxSignature : null,
    scheduledBurnDate: scheduledBurnDate ? scheduledBurnDate : null,
  };
}

/**
 * InspectionBatchDTO 内に埋め込まれている mint から MintInfo を抽出。
 */
export function extractMintInfoFromBatch(
  batch: InspectionBatchDTO | any,
): MintInfo | null {
  if (!batch) return null;

  const mintObj = (batch as any).mint ?? null;
  if (!mintObj) return null;

  return extractMintInfoFromMintDTO(mintObj as any);
}