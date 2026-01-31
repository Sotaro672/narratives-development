// frontend/console/mintRequest/src/presentation/hook/useMintRequestDetail.viewModels.ts

import { safeDateLabelJa, safeDateTimeLabelJa } from "../../../../shell/src/shared/util/dateJa";
import { asNonEmptyString } from "../../application/mapper/modelInspectionMapper";

import type { ProductBlueprintPatchDTO } from "../../infrastructure/dto/mintRequestLocal.dto";
import type { MintInfo } from "../../application/mapper/mintInfoMapper";

import type {
  ProductBlueprintCardVM as ProductBlueprintCardViewModel,
  TokenBlueprintCardVM as TokenBlueprintCardViewModel,
  TokenBlueprintCardHandlersVM as TokenBlueprintCardHandlers,
  BrandOptionVM as BrandOption,
  TokenBlueprintOptionVM as TokenBlueprintOption,
} from "../viewModel/mintRequestDetail.vm";

import type { TokenBlueprintPatchDTO } from "../../infrastructure/adapter/inventoryTokenBlueprintPatch";

export function buildProductBlueprintCardView(
  pbPatch: ProductBlueprintPatchDTO | null,
): ProductBlueprintCardViewModel | null {
  if (!pbPatch) return null;

  return {
    productName: (pbPatch as any)?.productName ?? undefined,
    brand: (pbPatch as any)?.brandName ?? undefined,
    itemType: (pbPatch as any)?.itemType ?? undefined,
    fit: (pbPatch as any)?.fit ?? undefined,
    materials: (pbPatch as any)?.material ?? undefined,
    weight: (pbPatch as any)?.weight ?? undefined,
    washTags: (pbPatch as any)?.qualityAssurance ?? undefined,
    productIdTag: (pbPatch as any)?.productIdTag?.type ?? undefined,
  };
}

export function buildTokenBlueprintCardVm(params: {
  selectedTokenBlueprint: TokenBlueprintOption | null;
  tokenBlueprintIdForPatch: string;
  selectedBrandName: string;
  tokenBlueprintPatch: TokenBlueprintPatchDTO | null;
  pbPatch: ProductBlueprintPatchDTO | null;
  brandOptions: BrandOption[];
}): TokenBlueprintCardViewModel | null {
  const {
    selectedTokenBlueprint,
    tokenBlueprintIdForPatch,
    selectedBrandName,
    tokenBlueprintPatch,
    pbPatch,
    brandOptions,
  } = params;

  const tbId =
    asNonEmptyString(selectedTokenBlueprint?.id) || asNonEmptyString(tokenBlueprintIdForPatch);
  if (!tbId) return null;

  const brandName =
    selectedBrandName ||
    asNonEmptyString((tokenBlueprintPatch as any)?.brandName) ||
    asNonEmptyString((pbPatch as any)?.brandName) ||
    "";

  const name =
    asNonEmptyString((tokenBlueprintPatch as any)?.tokenName) ||
    asNonEmptyString(selectedTokenBlueprint?.name);

  const symbol =
    asNonEmptyString((tokenBlueprintPatch as any)?.symbol) ||
    asNonEmptyString(selectedTokenBlueprint?.symbol);

  const description = asNonEmptyString((tokenBlueprintPatch as any)?.description);

  const iconUrl =
    asNonEmptyString((tokenBlueprintPatch as any)?.iconUrl) ||
    asNonEmptyString(selectedTokenBlueprint?.iconUrl) ||
    undefined;

  return {
    id: tbId,
    name: name || tbId,
    symbol: symbol || "",
    brandId: "",
    brandName,
    description: description || "",
    iconUrl,
    isEditMode: false,
    brandOptions: brandOptions.map((b) => ({ id: b.id, name: b.name })),
  };
}

export function buildTokenBlueprintCardHandlers(iconUrl?: string): TokenBlueprintCardHandlers {
  return {
    onPreview: () => {
      const url = iconUrl;
      if (url) window.open(url, "_blank", "noopener,noreferrer");
    },
  };
}

export function buildMintLabels(params: {
  mint: MintInfo | null;
  requestedByName: string | null;
}) {
  const { mint, requestedByName } = params;

  const mintCreatedAtLabel = safeDateTimeLabelJa(mint?.createdAt ?? null, "（未登録）");

  const mintCreatedByLabel = (() => {
    const name = asNonEmptyString(requestedByName);
    if (name) return name;

    const fallback = asNonEmptyString(mint?.createdBy);
    return fallback ? fallback : "（不明）";
  })();

  const mintScheduledBurnDateLabel = safeDateLabelJa(mint?.scheduledBurnDate ?? null, "（未設定）");

  const mintMintedAtLabel = safeDateTimeLabelJa(mint?.mintedAt ?? null, "（未完了）");

  const onChainTxSignature = asNonEmptyString(mint?.onChainTxSignature);

  return {
    mintCreatedAtLabel,
    mintCreatedByLabel,
    mintScheduledBurnDateLabel,
    mintMintedAtLabel,
    onChainTxSignature,
  };
}
