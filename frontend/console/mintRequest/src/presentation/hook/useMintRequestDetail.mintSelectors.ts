// frontend/console/mintRequest/src/presentation/hook/useMintRequestDetail.mintSelectors.ts

import * as React from "react";

import type { InspectionBatchDTO, MintDTO } from "../../infrastructure/api/mintRequestApi";
import { asNonEmptyString } from "../../application/mapper/modelInspectionMapper";

import {
  extractMintInfoFromBatch,
  extractMintInfoFromMintDTO,
  type MintInfo,
} from "../../application/mapper/mintInfoMapper";

import type { ProductBlueprintPatchDTO } from "../../infrastructure/dto/mintRequestLocal.dto";

export function useMintInfo(params: {
  mintDTO: MintDTO | null;
  inspectionBatch: InspectionBatchDTO | null;
  pbPatch: ProductBlueprintPatchDTO | null;
}) {
  const { mintDTO, inspectionBatch, pbPatch } = params;

  const mint: MintInfo | null = React.useMemo(() => {
    const fromDTO = extractMintInfoFromMintDTO(mintDTO as any);
    if (fromDTO) return fromDTO;

    const fromBatch = extractMintInfoFromBatch(inspectionBatch as any);
    return fromBatch;
  }, [mintDTO, inspectionBatch]);

  const hasMint = React.useMemo(() => !!mint, [mint]);

  // minted=true のときのみ非表示判定（= mint 完了扱い）
  const isMintRequested = React.useMemo(() => {
    return Boolean(mint?.minted === true);
  }, [mint]);

  // ✅ requestedByName（表示名）
  // - mintInfo が requestedByName を持つ場合はそれを最優先
  // - 次に createdByName
  // - 最後に createdBy（id）
  const requestedByName: string | null = React.useMemo(() => {
    const a = asNonEmptyString((mint as any)?.requestedByName);
    if (a) return a;

    const b = asNonEmptyString((mint as any)?.createdByName);
    if (b) return b;

    const c = asNonEmptyString((mint as any)?.createdBy);
    return c ? c : null;
  }, [mint]);

  const mintRequestedTokenBlueprintId = React.useMemo(() => {
    const v = asNonEmptyString(mint?.tokenBlueprintId);
    return v ? v : "";
  }, [mint]);

  const mintRequestedBrandId = React.useMemo(() => {
    // mint.brandId を最優先。無ければ pbPatch.brandId を fallback
    const fromMint = asNonEmptyString(mint?.brandId);
    if (fromMint) return fromMint;
    const fromPatch = asNonEmptyString((pbPatch as any)?.brandId);
    return fromPatch ? fromPatch : "";
  }, [mint, pbPatch]);

  return {
    mint,
    hasMint,
    isMintRequested,
    requestedByName,
    mintRequestedTokenBlueprintId,
    mintRequestedBrandId,
  };
}
