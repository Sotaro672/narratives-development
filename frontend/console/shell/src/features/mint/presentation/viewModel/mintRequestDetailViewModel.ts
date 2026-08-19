// frontend/console/shell/src/features/mint/presentation/viewModel/mintRequestDetailViewModel.ts

import type {
  BrandSummary,
  TokenBlueprintSummary,
} from "../../infrastructure/dto/MintRequestRepository";

import type {
  MintProductBlueprintDTO,
} from "../../infrastructure/dto/mintRequestLocal.dto";

import type {
  ProductBlueprintCardProps,
} from "../../../productBlueprint/presentation/cards/productBlueprintForm/productBlueprintCard";

import type {
  TokenBlueprintCardViewModel,
} from "../../../tokenBlueprint/presentation/components/tokenBlueprintCard";

export type ProductBlueprintCardViewModel = Pick<
  ProductBlueprintCardProps,
  "productName" | "brandName" | "productBlueprintCategoryPath"
>;

export type BuildTokenBlueprintCardVmInput = {
  selectedTokenBlueprint: TokenBlueprintSummary | null;
  displayTokenBlueprintId: string;
  selectedBrandName: string;
  productBlueprint: MintProductBlueprintDTO | null;
  brandOptions: BrandSummary[];
};

export function buildProductBlueprintCardView(
  productBlueprint: MintProductBlueprintDTO | null,
): ProductBlueprintCardViewModel | null {
  if (!productBlueprint) {
    return null;
  }

  return {
    productName: productBlueprint.productName ?? undefined,
    brandName: productBlueprint.brandName ?? undefined,
    productBlueprintCategoryPath:
      productBlueprint.productBlueprintCategoryPath ?? null,
  };
}

export function buildTokenBlueprintCardVm(
  input: BuildTokenBlueprintCardVmInput,
): TokenBlueprintCardViewModel | null {
  const {
    selectedTokenBlueprint,
    displayTokenBlueprintId,
    selectedBrandName,
    productBlueprint,
    brandOptions,
  } = input;

  const tokenBlueprintId =
    selectedTokenBlueprint?.id ??
    displayTokenBlueprintId;

  if (!tokenBlueprintId) {
    return null;
  }

  const tokenName =
    selectedTokenBlueprint?.tokenName ?? "";

  const symbol =
    selectedTokenBlueprint?.symbol ?? "";

  const brandId =
    selectedTokenBlueprint?.brandId ??
    productBlueprint?.brandId ??
    "";

  const brandName =
    selectedBrandName ||
    productBlueprint?.brandName ||
    "";

  const description =
    selectedTokenBlueprint?.description ?? "";

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
    minted: selectedTokenBlueprint?.minted ?? false,
    iconFile: null,
    isEditMode: false,
    brandOptions,
  };
}