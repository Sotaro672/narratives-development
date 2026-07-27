// frontend/console/shell/src/features/mintRequest/presentation/hook/useMintRequestDetail.mintSelectors.ts

import * as React from "react";

import type { MintInfo } from "../../application/mapper/mintInfoMapper";
import { asNonEmptyString } from "../../application/util/primitive";

import type { ProductBlueprintPatchDTO } from "../../infrastructure/dto/mintRequestLocal.dto";

type UseMintInfoParams = {
  /**
   * Application層でMintDTOから変換済みのMint情報。
   *
   * Presentation層ではMintDTOを直接扱わない。
   */
  mint: MintInfo | null;

  /**
   * MintにbrandIdが存在しない場合に使用する
   * Product Blueprint側の補助情報。
   */
  pbPatch: ProductBlueprintPatchDTO | null;
};

export function useMintInfo({
  mint,
  pbPatch,
}: UseMintInfoParams) {
  const hasMint = mint !== null;

  /**
   * Mintが存在し、まだMINTEDではない場合は
   * 画面上「ミント中」として扱う。
   */
  const isMinting =
    hasMint &&
    mint.status !== "MINTED";

  /**
   * Backendの正規状態に合わせ、
   * status === "MINTED"のみを
   * 画面上「ミント完了」として扱う。
   */
  const isMintCompleted =
    mint?.status === "MINTED";

  /**
   * mintsドキュメントを作成した人の表示値。
   *
   * 優先順位:
   * 1. createdByName
   * 2. createdBy
   *
   * requestedBy系からのfallbackは行わない。
   */
  const createdByName:
    | string
    | null = React.useMemo(() => {
    const creatorName =
      asNonEmptyString(
        mint?.createdByName,
      );

    if (creatorName) {
      return creatorName;
    }

    const creatorId =
      asNonEmptyString(
        mint?.createdBy,
      );

    return creatorId || null;
  }, [mint]);

  /**
   * Mint申請ボタンを押した人の表示値。
   *
   * 優先順位:
   * 1. requestedByName
   * 2. requestedBy
   *
   * createdBy系からのfallbackは行わない。
   */
  const requestedByName:
    | string
    | null = React.useMemo(() => {
    const requesterName =
      asNonEmptyString(
        mint?.requestedByName,
      );

    if (requesterName) {
      return requesterName;
    }

    const requesterId =
      asNonEmptyString(
        mint?.requestedBy,
      );

    return requesterId || null;
  }, [mint]);

  /**
   * Mintで選択されたToken Blueprint ID。
   */
  const mintRequestedTokenBlueprintId =
    React.useMemo(() => {
      return asNonEmptyString(
        mint?.tokenBlueprintId,
      );
    }, [mint]);

  /**
   * Mintで選択されたブランドID。
   *
   * Mint側にbrandIdが存在しない場合だけ、
   * Product Blueprint側のbrandIdを使用する。
   */
  const mintRequestedBrandId =
    React.useMemo(() => {
      const brandIdFromMint =
        asNonEmptyString(
          mint?.brandId,
        );

      if (brandIdFromMint) {
        return brandIdFromMint;
      }

      return asNonEmptyString(
        pbPatch?.brandId,
      );
    }, [
      mint,
      pbPatch,
    ]);

  return {
    mint,
    hasMint,
    isMinting,
    isMintCompleted,
    createdByName,
    requestedByName,
    mintRequestedTokenBlueprintId,
    mintRequestedBrandId,
  };
}