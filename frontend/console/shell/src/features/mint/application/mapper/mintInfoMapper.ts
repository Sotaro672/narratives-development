// frontend/console/shell/src/features/mintRequest/application/mapper/mintInfoMapper.ts

import type { InspectionBatchDTO } from "../../../../shared/types/inspections";
import type { MintStatus } from "../../../../shared/types/mints";
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

  /**
   * mintsドキュメントを作成したmemberId。
   */
  createdBy: string;

  /**
   * createdByに対応する表示名。
   */
  createdByName: string | null;

  /**
   * mintsドキュメントの作成日時。
   */
  createdAt: string | null;

  /**
   * Mint申請ボタンを押したmemberId。
   *
   * Mint未申請の場合はnull。
   */
  requestedBy: string | null;

  /**
   * requestedByに対応する表示名。
   *
   * Mint未申請の場合、または表示名を取得できない場合はnull。
   */
  requestedByName: string | null;

  mintedAt: string | null;
  onChainTxSignature: string | null;
  scheduledBurnDate: string | null;
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
 * createdByとrequestedByは、それぞれ独立したデータとして扱う。
 *
 * - createdBy:
 *   mintsドキュメントを作成したmemberId
 *
 * - requestedBy:
 *   Mint申請ボタンを押したmemberId
 *
 * createdBy系とrequestedBy系の間でfallbackや値の補完は行わない。
 *
 * Mint状態についても、このmapperでは再解釈せず、
 * Backendと共通のstatusをそのまま使用する。
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

  const createdBy = asNonEmptyString(
    mintDTO.createdBy,
  );

  const createdByName =
    asNonEmptyString(
      mintDTO.createdByName,
    ) || null;

  const requestedBy =
    asNonEmptyString(
      mintDTO.requestedBy,
    ) || null;

  const requestedByName =
    asNonEmptyString(
      mintDTO.requestedByName,
    ) || null;

  const createdAt =
    asNonEmptyString(
      asMaybeISO(
        mintDTO.createdAt,
      ),
    ) || null;

  const mintedAt =
    asNonEmptyString(
      asMaybeISO(
        mintDTO.mintedAt,
      ),
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

    createdBy,
    createdByName,
    createdAt,

    requestedBy,
    requestedByName,

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