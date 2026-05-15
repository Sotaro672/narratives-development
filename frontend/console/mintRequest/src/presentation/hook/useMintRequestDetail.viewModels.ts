// frontend/console/mintRequest/src/presentation/hook/useMintRequestDetail.viewModels.ts

import {
  safeDateLabelJa,
  safeDateTimeLabelJa,
} from "../../../../shell/src/shared/util/dateJa";
import { asNonEmptyString } from "../../application/util/primitive";

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

const toDisplayText = (value: unknown): string => {
  if (value === null || value === undefined) return "";

  if (typeof value === "string") {
    return value.trim();
  }

  if (typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }

  return "";
};

const toProductIdTagLabel = (
  productIdTag: ProductBlueprintPatchDTO["productIdTag"],
): string | undefined => {
  if (!productIdTag) return undefined;

  const value =
    asNonEmptyString(productIdTag.type) ||
    asNonEmptyString(productIdTag.Type);

  return value || undefined;
};

const buildCategoryFieldRows = (
  pbPatch: ProductBlueprintPatchDTO,
): { label: string; value: string }[] => {
  const fields = pbPatch.categoryFields;
  if (!fields) return [];

  const rows: { label: string; value: string }[] = [];

  const addRow = (key: string, label: string, suffix = "") => {
    const value = toDisplayText(fields[key]);
    if (!value) return;

    rows.push({
      label,
      value: suffix ? `${value}${suffix}` : value,
    });
  };

  /**
   * alcohol category fields
   */
  addRow("vintage", "ヴィンテージ");
  addRow("region", "地域");
  addRow("material", "原材料");
  addRow("alcoholContent", "アルコール度数", "%");

  /**
   * 未定義カテゴリや今後追加される categoryFields 用。
   * 既知 key は上で表示済みなので除外する。
   */
  const knownKeys = new Set([
    "vintage",
    "region",
    "material",
    "alcoholContent",
  ]);

  Object.entries(fields).forEach(([key, value]) => {
    if (knownKeys.has(key)) return;

    const displayValue = toDisplayText(value);
    if (!displayValue) return;

    rows.push({
      label: key,
      value: displayValue,
    });
  });

  return rows;
};

export function buildProductBlueprintCardView(
  pbPatch: ProductBlueprintPatchDTO | null,
): ProductBlueprintCardViewModel | null {
  if (!pbPatch) return null;

  const category = pbPatch.productBlueprintCategory ?? null;

  const productName = asNonEmptyString(pbPatch.productName) || "";
  const brand = asNonEmptyString(pbPatch.brandName) || "";

  const categoryName =
    asNonEmptyString(category?.nameJa) ||
    asNonEmptyString(category?.nameEn) ||
    asNonEmptyString(category?.code) ||
    "";

  const categoryCode = asNonEmptyString(category?.code) || "";
  const categoryKind = asNonEmptyString(category?.kind) || "";

  return {
    productName,
    brand,

    /**
     * ProductBlueprintCard は categoryName ではなく
     * productBlueprintCategory / productBlueprintPatch.productBlueprintCategory を見て
     * 商品カテゴリを表示する。
     */
    productBlueprintCategory: category,

    /**
     * mintRequest 側で補助表示・条件分岐に使う派生値。
     */
    categoryName,
    categoryCode,
    categoryKind,

    categoryFields: pbPatch.categoryFields ?? null,
    categoryFieldRows: buildCategoryFieldRows(pbPatch),

    productIdTag: toProductIdTagLabel(pbPatch.productIdTag),
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
    asNonEmptyString(selectedTokenBlueprint?.id) ||
    asNonEmptyString(tokenBlueprintIdForPatch);

  if (!tbId) return null;

  const brandName =
    selectedBrandName ||
    asNonEmptyString((tokenBlueprintPatch as any)?.brandName) ||
    asNonEmptyString(pbPatch?.brandName) ||
    "";

  const name =
    asNonEmptyString((tokenBlueprintPatch as any)?.tokenName) ||
    asNonEmptyString(selectedTokenBlueprint?.name);

  const symbol =
    asNonEmptyString((tokenBlueprintPatch as any)?.symbol) ||
    asNonEmptyString(selectedTokenBlueprint?.symbol);

  const description = asNonEmptyString(
    (tokenBlueprintPatch as any)?.description,
  );

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

export function buildTokenBlueprintCardHandlers(
  iconUrl?: string,
): TokenBlueprintCardHandlers {
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

  const mintCreatedAtLabel = safeDateTimeLabelJa(
    mint?.createdAt ?? null,
    "（未登録）",
  );

  const mintCreatedByLabel = (() => {
    const name = asNonEmptyString(requestedByName);
    if (name) return name;

    const fallback = asNonEmptyString(mint?.createdBy);
    return fallback ? fallback : "（不明）";
  })();

  const mintScheduledBurnDateLabel = safeDateLabelJa(
    mint?.scheduledBurnDate ?? null,
    "（未設定）",
  );

  const mintMintedAtLabel = safeDateTimeLabelJa(
    mint?.mintedAt ?? null,
    "（未完了）",
  );

  const onChainTxSignature = asNonEmptyString(mint?.onChainTxSignature);

  return {
    mintCreatedAtLabel,
    mintCreatedByLabel,
    mintScheduledBurnDateLabel,
    mintMintedAtLabel,
    onChainTxSignature,
  };
}