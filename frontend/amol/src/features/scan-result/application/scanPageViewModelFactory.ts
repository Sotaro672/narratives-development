// frontend/amol/src/features/scan-result/application/scanPageViewModelFactory.ts

import { rgbToCssColor } from "../../../components/utils/color";
import type {
  MallOwnerInfo,
  PreviewState,
  ProductBlueprintPatch,
} from "../../shared/types/scanResult";
import {
  createScanAlcoholInfo,
  type ScanAlcoholInfo,
} from "./scanAlcoholInfoFactory";

export type ScanDisplayRowViewModel = {
  label: string;
  value: string;
};

export type ScanProductSectionViewModel = {
  productId: string;
  productBlueprintId: string;
  title: string;
  ownerLabel: string;
  brandId: string;
  brandName: string;
  hasBrandInfo: boolean;
  productBlueprintRows: ScanDisplayRowViewModel[];
  qualityAssuranceTabs: string[];
  modelNumber: string;
  size: string;
  color: string;
  swatch: string;
  measurementEntries: ScanDisplayRowViewModel[];
  alcoholInfo: ScanAlcoholInfo | null;
};

export type ScanTokenSectionViewModel = {
  tokenName: string;
  tokenIconUrl: string;
  tokenBrandName: string;
  tokenCompanyName: string;
  tokenDescription: string;
  assetId: string;
  canOpenTokenContents: boolean;
};

export type ScanResultPageViewModel = {
  product: ScanProductSectionViewModel;
  token: ScanTokenSectionViewModel | null;
};

export type CreateScanResultPageViewModelInput = {
  previewState: PreviewState | null;
  ownedByWallet: boolean | null;
};

function toDisplayText(value: unknown): string {
  if (typeof value === "string") return value;
  if (typeof value === "number" && Number.isFinite(value)) return String(value);
  if (typeof value === "boolean") return String(value);
  return "";
}

function resolveOwnerLabel(owner: MallOwnerInfo | null): string {
  if (!owner) return "-";
  if (owner.ownerType === "avatar") return owner.avatarName ?? "-";
  if (owner.ownerType === "brand") return owner.brandName ?? "-";
  return "-";
}

function createProductBlueprintRows(
  patch: ProductBlueprintPatch | null,
): ScanDisplayRowViewModel[] {
  if (!patch) return [];

  const categoryFields = patch.categoryFields ?? {};
  const rows: ScanDisplayRowViewModel[] = [
    {
      label: "種別",
      value: toDisplayText(categoryFields.itemType),
    },
    {
      label: "フィット",
      value: toDisplayText(categoryFields.fit),
    },
    {
      label: "素材",
      value: toDisplayText(categoryFields.material),
    },
    {
      label: "重量",
      value: toDisplayText(categoryFields.weight),
    },
    {
      label: "商品IDタグ",
      value: patch.productIdTag?.Type ?? "",
    },
  ];

  return rows.filter((row) => Boolean(row.value));
}

function createQualityAssuranceTabs(
  patch: ProductBlueprintPatch | null,
): string[] {
  const rawValue = patch?.categoryFields?.qualityAssurance;
  if (!Array.isArray(rawValue)) return [];

  return rawValue.map(toDisplayText).filter(Boolean);
}

function createMeasurementEntries(
  measurements: Record<string, number> | null,
): ScanDisplayRowViewModel[] {
  if (!measurements) return [];

  return Object.entries(measurements)
    .filter(([key, value]) => Boolean(key) && Number.isFinite(value))
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([label, value]) => ({
      label,
      value: `${value}cm`,
    }));
}

function createTokenViewModel(input: {
  previewState: PreviewState;
  ownedByWallet: boolean | null;
}): ScanTokenSectionViewModel | null {
  const preview = input.previewState.raw;
  const tokenBlueprintPatch = preview.tokenBlueprintPatch;

  if (!tokenBlueprintPatch) return null;

  const assetId = preview.token?.assetId ?? "";

  return {
    tokenName: tokenBlueprintPatch.tokenName,
    tokenIconUrl: tokenBlueprintPatch.tokenIcon,
    tokenBrandName: tokenBlueprintPatch.brandName,
    tokenCompanyName: tokenBlueprintPatch.companyName,
    tokenDescription: tokenBlueprintPatch.description,
    assetId,
    canOpenTokenContents:
      input.ownedByWallet === true &&
      Boolean(assetId) &&
      Boolean(tokenBlueprintPatch.tokenName),
  };
}

export function createScanResultPageViewModel(
  input: CreateScanResultPageViewModelInput,
): ScanResultPageViewModel | null {
  const previewState = input.previewState;
  if (!previewState) return null;

  const preview = previewState.raw;
  const patch = preview.productBlueprintPatch;

  const productId = preview.productId;
  const productBlueprintId = preview.productBlueprintId;
  const productName = patch?.productName ?? "";
  const modelNumber = preview.modelNumber;
  const brandId = patch?.brandId ?? "";
  const brandName = preview.brandName ?? "";
  const size = preview.size;
  const color = preview.color;
  const productBlueprintCategoryKind =
    preview.productBlueprintCategoryPath?.[0] ?? "";

  const alcoholInfo = createScanAlcoholInfo({
    categoryFields: patch?.categoryFields,
    volumeValue: preview.volumeValue,
    volumeUnit: preview.volumeUnit,
    productBlueprintCategoryKind,
  });

  return {
    product: {
      productId,
      productBlueprintId,
      title: productName || modelNumber || productId || "Scan Result",
      ownerLabel: resolveOwnerLabel(preview.owner),
      brandId,
      brandName,
      hasBrandInfo: Boolean(brandId || brandName),
      productBlueprintRows: createProductBlueprintRows(patch),
      qualityAssuranceTabs: createQualityAssuranceTabs(patch),
      modelNumber,
      size,
      color,
      swatch: rgbToCssColor(preview.rgb),
      measurementEntries: createMeasurementEntries(preview.measurements),
      alcoholInfo,
    },
    token: createTokenViewModel({
      previewState,
      ownedByWallet: input.ownedByWallet,
    }),
  };
}