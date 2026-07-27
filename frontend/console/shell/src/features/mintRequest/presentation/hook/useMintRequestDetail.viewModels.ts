// frontend/console/shell/src/features/mintRequest/presentation/hook/useMintRequestDetail.viewModels.ts

import {
  safeDateLabelJa,
  safeDateTimeLabelJa,
} from "../../../../shared/util/dateJa";

import type {
  BrandSummary,
  TokenBlueprintSummary,
} from "../../application/port/MintRequestRepository";
import type { MintInfo } from "../../application/mapper/mintInfoMapper";
import { asNonEmptyString } from "../../application/util/primitive";

import type { ProductBlueprintPatchDTO } from "../../infrastructure/dto/mintRequestLocal.dto";

import type {
  ProductBlueprintCardVM as ProductBlueprintCardViewModel,
  TokenBlueprintCardHandlersVM as TokenBlueprintCardHandlers,
  TokenBlueprintCardVM as TokenBlueprintCardViewModel,
} from "../viewModel/mintRequestDetail.vm";

export function buildProductBlueprintCardView(
  pbPatch: ProductBlueprintPatchDTO | null,
): ProductBlueprintCardViewModel | null {
  if (!pbPatch) {
    return null;
  }

  return {
    productName:
      asNonEmptyString(
        pbPatch.productName,
      ) || undefined,

    brandName:
      asNonEmptyString(
        pbPatch.brandName,
      ) || undefined,

    productBlueprintCategory:
      pbPatch.productBlueprintCategory ??
      null,
  };
}

export function buildTokenBlueprintCardVm(params: {
  selectedTokenBlueprint:
    | TokenBlueprintSummary
    | null;
  tokenBlueprintIdForPatch: string;
  selectedBrandName: string;
  pbPatch: ProductBlueprintPatchDTO | null;
  brandOptions: BrandSummary[];
}): TokenBlueprintCardViewModel | null {
  const {
    selectedTokenBlueprint,
    tokenBlueprintIdForPatch,
    selectedBrandName,
    pbPatch,
    brandOptions,
  } = params;

  const tokenBlueprintId =
    asNonEmptyString(
      selectedTokenBlueprint?.id,
    ) ||
    asNonEmptyString(
      tokenBlueprintIdForPatch,
    );

  if (!tokenBlueprintId) {
    return null;
  }

  const tokenName =
    asNonEmptyString(
      selectedTokenBlueprint?.tokenName,
    ) ||
    tokenBlueprintId;

  const symbol =
    asNonEmptyString(
      selectedTokenBlueprint?.symbol,
    );

  const brandId =
    asNonEmptyString(
      selectedTokenBlueprint?.brandId,
    );

  const brandName =
    asNonEmptyString(
      selectedBrandName,
    ) ||
    asNonEmptyString(
      selectedTokenBlueprint?.brandName,
    ) ||
    asNonEmptyString(
      pbPatch?.brandName,
    );

  const description =
    asNonEmptyString(
      selectedTokenBlueprint?.description,
    );

  const iconUrl =
    asNonEmptyString(
      selectedTokenBlueprint?.iconUrl,
    ) || undefined;

  return {
    id: tokenBlueprintId,
    tokenName,
    symbol,
    brandId,
    brandName,
    description,
    iconUrl,
    isEditMode: false,
    brandOptions,
  };
}

export function buildTokenBlueprintCardHandlers(
  iconUrl?: string,
): TokenBlueprintCardHandlers {
  return {
    onPreview: () => {
      if (!iconUrl) {
        return;
      }

      window.open(
        iconUrl,
        "_blank",
        "noopener,noreferrer",
      );
    },
  };
}

export function buildMintLabels(params: {
  mint: MintInfo | null;
  createdByName: string | null;
}) {
  const {
    mint,
    createdByName,
  } = params;

  const mintCreatedAtLabel =
    safeDateTimeLabelJa(
      mint?.createdAt ?? null,
      "（未登録）",
    );

  /**
   * mintsドキュメントを作成した人の表示値。
   *
   * 優先順位:
   * 1. createdByName
   * 2. createdBy
   *
   * requestedBy系からのfallbackは行わない。
   */
  const mintCreatedByLabel = (() => {
    const creatorName =
      asNonEmptyString(
        createdByName,
      );

    if (creatorName) {
      return creatorName;
    }

    const creatorId =
      asNonEmptyString(
        mint?.createdBy,
      );

    return creatorId || "（不明）";
  })();

  const mintScheduledBurnDateLabel =
    safeDateLabelJa(
      mint?.scheduledBurnDate ?? null,
      "（未設定）",
    );

  const mintMintedAtLabel =
    safeDateTimeLabelJa(
      mint?.mintedAt ?? null,
      "（未完了）",
    );

  const onChainTxSignature =
    asNonEmptyString(
      mint?.onChainTxSignature,
    );

  return {
    mintCreatedAtLabel,
    mintCreatedByLabel,
    mintScheduledBurnDateLabel,
    mintMintedAtLabel,
    onChainTxSignature,
  };
}