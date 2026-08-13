// frontend/console/shell/src/features/mintRequest/presentation/viewModel/mintRequestDetailViewModel.ts

import type {
  BrandSummary,
  TokenBlueprintSummary,
} from "../../application/port/MintRequestRepository";

import type {
  ProductBlueprintPatchDTO,
} from "../../infrastructure/dto/mintRequestLocal.dto";

import type {
  ProductBlueprintCardProps,
} from "../../../productBlueprint/presentation/cards/productBlueprintForm/productBlueprintCard";

import type {
  TokenBlueprintCardViewModel,
} from "../../../tokenBlueprint/presentation/components/tokenBlueprintCard";

export type ProductBlueprintCardViewModel =
  Pick<
    ProductBlueprintCardProps,
    | "productName"
    | "brandName"
    | "productBlueprintCategory"
  >;

export type BuildTokenBlueprintCardVmInput = {
  selectedTokenBlueprint:
    | TokenBlueprintSummary
    | null;

  tokenBlueprintIdForPatch:
    string;

  selectedBrandName:
    string;

  pbPatch:
    | ProductBlueprintPatchDTO
    | null;

  brandOptions:
    BrandSummary[];
};

export function buildProductBlueprintCardView(
  pbPatch:
    | ProductBlueprintPatchDTO
    | null,
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

export function buildTokenBlueprintCardVm(
  input:
    BuildTokenBlueprintCardVmInput,
): TokenBlueprintCardViewModel | null {
  const {
    selectedTokenBlueprint,
    tokenBlueprintIdForPatch,
    selectedBrandName,
    pbPatch,
    brandOptions,
  } = input;

  const tokenBlueprintId =
    selectedTokenBlueprint?.id ??
    tokenBlueprintIdForPatch;

  if (!tokenBlueprintId) {
    return null;
  }

  const tokenName =
    selectedTokenBlueprint?.tokenName ??
    "";

  const symbol =
    selectedTokenBlueprint?.symbol ??
    "";

  const brandId =
    selectedTokenBlueprint?.brandId ??
    pbPatch?.brandId ??
    "";

  const brandName =
    selectedBrandName ||
    pbPatch?.brandName ||
    "";

  const description =
    selectedTokenBlueprint?.description ??
    "";

  const iconUrl =
    selectedTokenBlueprint?.iconUrl;

  return {
    id:
      tokenBlueprintId,

    name:
      tokenName,

    symbol,
    brandId,
    brandName,
    description,
    iconUrl,

    minted:
      selectedTokenBlueprint?.minted ??
      false,

    iconFile:
      null,

    isEditMode:
      false,

    brandOptions,
  };
}