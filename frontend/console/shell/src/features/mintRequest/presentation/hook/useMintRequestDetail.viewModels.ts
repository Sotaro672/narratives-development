// frontend/console/shell/src/features/mintRequest/presentation/hook/useMintRequestDetail.viewModels.ts

import {
  safeDateLabelJa,
  safeDateTimeLabelJa,
} from "../../../../shared/util/dateJa";

import type { MintInfo } from "../../application/mapper/mintInfoMapper";
import type {
  BrandSummary,
  TokenBlueprintSummary,
} from "../../application/port/MintRequestRepository";

import type { ProductBlueprintPatchDTO } from "../../infrastructure/dto/mintRequestLocal.dto";

import type { ProductBlueprintCardProps } from "../../../productBlueprint/presentation/cards/productBlueprintForm/productBlueprintCard";
import type { TokenBlueprintCardViewModel } from "../../../tokenBlueprint/presentation/components/tokenBlueprintCard";

type ProductBlueprintCardViewModel =
  Pick<
    ProductBlueprintCardProps,
    | "productName"
    | "brandName"
    | "productBlueprintCategory"
  >;

export function buildProductBlueprintCardView(
  pbPatch: ProductBlueprintPatchDTO | null,
): ProductBlueprintCardViewModel | null {
  if (!pbPatch) {
    return null;
  }

  return {
    productName:
      pbPatch.productName ??
      undefined,

    brandName:
      pbPatch.brandName ??
      undefined,

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
    selectedTokenBlueprint?.id ||
    tokenBlueprintIdForPatch;

  if (!tokenBlueprintId) {
    return null;
  }

  const tokenName =
    selectedTokenBlueprint?.tokenName ||
    tokenBlueprintId;

  const symbol =
    selectedTokenBlueprint?.symbol ??
    "";

  const brandId =
    selectedTokenBlueprint?.brandId ??
    "";

  const brandName =
    selectedBrandName ||
    selectedTokenBlueprint?.brandName ||
    pbPatch?.brandName ||
    "";

  const description =
    selectedTokenBlueprint?.description ??
    "";

  const iconUrl =
    selectedTokenBlueprint?.iconUrl;

  return {
    id: tokenBlueprintId,
    name: tokenName,
    symbol,
    brandId,
    brandName,
    description,
    iconUrl,
    minted:
      selectedTokenBlueprint?.minted ??
      false,
    iconFile: null,
    isEditMode: false,
    brandOptions,
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
   * createdByNameはselector側で、
   * createdByNameからcreatedByへのfallbackまで完了している。
   *
   * MintInfo由来の値はApplication層で正規化済みのため、
   * Presentation層では再正規化しない。
   *
   * requestedBy系からのfallbackは行わない。
   */
  const mintCreatedByLabel =
    createdByName || "（不明）";

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

  /**
   * MintInfoへの変換時に正規化済み。
   */
  const onChainTxSignature =
    mint?.onChainTxSignature ?? "";

  return {
    mintCreatedAtLabel,
    mintCreatedByLabel,
    mintScheduledBurnDateLabel,
    mintMintedAtLabel,
    onChainTxSignature,
  };
}