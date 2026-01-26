// frontend/console/mintRequest/src/application/mapper/mintInfoMapper.ts

import type {
  InspectionBatchDTO,
  MintDTO,
} from "../../infrastructure/api/mintRequestApi";

// ============================================================
// Types
// ============================================================

export type MintInfo = {
  id: string;

  brandId: string;
  tokenBlueprintId: string;

  createdBy: string;
  createdByName?: string | null;
  createdAt: string | null;

  minted: boolean;
  mintedAt?: string | null;
  onChainTxSignature?: string | null;
  scheduledBurnDate?: string | null;
};

// ============================================================
// helpers
// ============================================================

export function asNonEmptyString(v: any): string {
  return typeof v === "string" && v.trim() ? v.trim() : "";
}

export function asMaybeISO(v: any): string {
  if (!v) return "";
  if (typeof v === "string") return v;
  if (v instanceof Date) return v.toISOString();
  return String(v);
}

// ============================================================
// mapper
// ============================================================

/**
 * MintDTO（優先）から MintInfo を抽出。
 */
export function extractMintInfoFromMintDTO(m: MintDTO | any): MintInfo | null {
  if (!m) return null;

  const id = asNonEmptyString((m as any).id ?? (m as any).mintId);
  if (!id) return null;

  const tokenBlueprintId = asNonEmptyString((m as any).tokenBlueprintId);
  const brandId = asNonEmptyString((m as any).brandId);

  const createdBy = asNonEmptyString((m as any).createdBy);
  const createdByName = asNonEmptyString((m as any).createdByName);

  const createdAtStr = asNonEmptyString(asMaybeISO((m as any).createdAt));
  const createdAt = createdAtStr ? createdAtStr : null;

  const mintedAtStr = asNonEmptyString(asMaybeISO((m as any).mintedAt));
  const minted =
    typeof (m as any).minted === "boolean" ? (m as any).minted : Boolean(mintedAtStr);

  const onChainTxSignature = asNonEmptyString((m as any).onChainTxSignature);
  const scheduledBurnDate = asNonEmptyString(asMaybeISO((m as any).scheduledBurnDate));

  return {
    id,
    brandId,
    tokenBlueprintId,
    createdBy,
    createdByName: createdByName ? createdByName : null,
    createdAt,
    minted,
    mintedAt: mintedAtStr ? mintedAtStr : null,
    onChainTxSignature: onChainTxSignature ? onChainTxSignature : null,
    scheduledBurnDate: scheduledBurnDate ? scheduledBurnDate : null,
  };
}

/**
 * InspectionBatchDTO 内に埋め込まれている mint 互換フィールドから MintInfo を抽出。
 * - batch.mint / batch.mintRequest などの揺れを吸収
 */
export function extractMintInfoFromBatch(
  batch: InspectionBatchDTO | any,
): MintInfo | null {
  if (!batch) return null;

  const mintObj = (batch as any).mint ?? (batch as any).mintRequest ?? null;
  if (!mintObj) return null;

  return extractMintInfoFromMintDTO(mintObj as any);
}
