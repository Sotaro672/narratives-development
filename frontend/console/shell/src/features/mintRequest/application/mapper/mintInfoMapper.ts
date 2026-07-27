// frontend/console/shell/src/features/mintRequest/application/mapper/mintInfoMapper.ts

import type { MintStatus } from "../../domain/mints";
import type { InspectionBatchDTO } from "../../domain/inspections";
import type { MintDTO } from "../../infrastructure/dto/mint.dto";

import {
  asMaybeISO,
  asNonEmptyString,
} from "../util/primitive";

// ============================================================
// Types
// ============================================================

export type MintInfo = {
  id: string;

  brandId: string;
  tokenBlueprintId: string;

  status: MintStatus;

  requestedByName?: string | null;

  createdBy: string;
  createdByName?: string | null;
  createdAt: string | null;

  mintedAt?: string | null;
  onChainTxSignature?: string | null;
  scheduledBurnDate?: string | null;
};

/**
 * 詳細APIのinspection内にMint情報が含まれる場合の型。
 *
 * InspectionBatchDTO本体にはMint情報を持たせず、
 * APIレスポンス上の追加情報として扱う。
 */
type InspectionBatchWithMintDTO =
  InspectionBatchDTO & {
    mint?: MintDTO | null;
  };

// ============================================================
// mapper
// ============================================================

/**
 * MintDTOから画面表示用のMintInfoを生成する。
 *
 * ミント状態はmintedフラグを再生成せず、
 * Backendと共通のstatusを正とする。
 *
 * - status !== "MINTED": ミント中
 * - status === "MINTED": ミント完了
 */
export function extractMintInfoFromMintDTO(
  mintDTO: MintDTO | null | undefined,
): MintInfo | null {
  if (!mintDTO) {
    return null;
  }

  const id = asNonEmptyString(
    mintDTO.id,
  );

  if (!id) {
    return null;
  }

  const brandId = asNonEmptyString(
    mintDTO.brandId,
  );

  const tokenBlueprintId =
    asNonEmptyString(
      mintDTO.tokenBlueprintId,
    );

  const requestedByName =
    asNonEmptyString(
      mintDTO.requestedByName,
    );

  const createdBy = asNonEmptyString(
    mintDTO.createdBy,
  );

  const createdByName =
    asNonEmptyString(
      mintDTO.createdByName,
    );

  const createdAt =
    asNonEmptyString(
      asMaybeISO(mintDTO.createdAt),
    ) || null;

  const mintedAt =
    asNonEmptyString(
      asMaybeISO(mintDTO.mintedAt),
    ) || null;

  const onChainTxSignature =
    asNonEmptyString(
      mintDTO.onChainTxSignature,
    ) || null;

  const scheduledBurnDate =
    asNonEmptyString(
      asMaybeISO(
        mintDTO.scheduledBurnDate,
      ),
    ) || null;

  return {
    id,

    brandId,
    tokenBlueprintId,

    status: mintDTO.status,

    requestedByName:
      requestedByName || null,

    createdBy,
    createdByName:
      createdByName || null,
    createdAt,

    mintedAt,
    onChainTxSignature,
    scheduledBurnDate,
  };
}

/**
 * InspectionBatchDTO内に埋め込まれているMint情報から
 * 画面表示用のMintInfoを生成する。
 */
export function extractMintInfoFromBatch(
  batch:
    | InspectionBatchWithMintDTO
    | null
    | undefined,
): MintInfo | null {
  if (!batch?.mint) {
    return null;
  }

  return extractMintInfoFromMintDTO(
    batch.mint,
  );
}