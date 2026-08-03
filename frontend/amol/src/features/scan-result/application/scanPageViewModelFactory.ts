// frontend/amol/src/features/scan-result/application/scanPageViewModelFactory.ts

import {
  rgbToCssColor,
} from "../../../components/utils/color";

import type {
  MallOwnerInfo,
  PreviewState,
  ProductBlueprintCategoryFields,
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

  productBlueprintRows:
    ScanDisplayRowViewModel[];
  qualityAssuranceTabs: string[];

  modelNumber: string;
  size: string;
  color: string;
  swatch: string;

  measurementEntries:
    ScanDisplayRowViewModel[];

  alcoholInfo: ScanAlcoholInfo | null;
};

export type ScanTokenSectionViewModel = {
  tokenName: string;
  tokenIconUrl: string;
  tokenBrandName: string;
  tokenCompanyName: string;
  tokenDescription: string;

  mintAddress: string;
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

function normalizeText(
  value: string | null | undefined,
): string {
  return value?.trim() ?? "";
}

function toDisplayText(
  value: unknown,
): string {
  if (typeof value === "string") {
    return value.trim();
  }

  if (
    typeof value === "number" &&
    Number.isFinite(value)
  ) {
    return String(value);
  }

  if (typeof value === "boolean") {
    return String(value);
  }

  return "";
}

function resolveOwnerLabel(
  owner: MallOwnerInfo | null,
): string {
  if (!owner) {
    return "-";
  }

  return (
    normalizeText(owner.avatarName) ||
    normalizeText(owner.brandName) ||
    normalizeText(owner.avatarId) ||
    normalizeText(owner.brandId) ||
    "-"
  );
}

function resolvePatchValue(
  patch: ProductBlueprintPatch,
  categoryFields:
    ProductBlueprintCategoryFields,
  key: string,
): string {
  const categoryValue = toDisplayText(
    categoryFields[key],
  );

  if (categoryValue) {
    return categoryValue;
  }

  return toDisplayText(
    patch[key],
  );
}

function createProductBlueprintRows(
  patch: ProductBlueprintPatch | null,
): ScanDisplayRowViewModel[] {
  if (!patch) {
    return [];
  }

  const categoryFields =
    patch.categoryFields ?? {};

  const rows: ScanDisplayRowViewModel[] = [
    {
      label: "種別",
      value: resolvePatchValue(
        patch,
        categoryFields,
        "itemType",
      ),
    },
    {
      label: "フィット",
      value: resolvePatchValue(
        patch,
        categoryFields,
        "fit",
      ),
    },
    {
      label: "素材",
      value: resolvePatchValue(
        patch,
        categoryFields,
        "material",
      ),
    },
    {
      label: "重量",
      value: resolvePatchValue(
        patch,
        categoryFields,
        "weight",
      ),
    },
    {
      label: "商品IDタグ",
      value:
        normalizeText(
          patch.productIdTag?.Type,
        ) ||
        normalizeText(
          patch.productIdTag?.type,
        ),
    },
  ];

  return rows.filter(
    (row) => Boolean(row.value),
  );
}

function createQualityAssuranceTabs(
  patch: ProductBlueprintPatch | null,
): string[] {
  if (!patch) {
    return [];
  }

  const rawValue =
    patch.categoryFields?.qualityAssurance ??
    patch.qualityAssurance;

  if (!Array.isArray(rawValue)) {
    return [];
  }

  return rawValue
    .map(toDisplayText)
    .filter(Boolean);
}

function createMeasurementEntries(
  measurements:
    Record<string, number> | null,
): ScanDisplayRowViewModel[] {
  if (!measurements) {
    return [];
  }

  return Object.entries(measurements)
    .filter(([key, value]) => {
      return (
        Boolean(key.trim()) &&
        Number.isFinite(value)
      );
    })
    .sort(([left], [right]) => {
      return left.localeCompare(right);
    })
    .map(([label, value]) => ({
      label,
      value: `${value}cm`,
    }));
}

function createTokenViewModel(input: {
  previewState: PreviewState;
  ownedByWallet: boolean | null;
}): ScanTokenSectionViewModel | null {
  const {
    previewState,
    ownedByWallet,
  } = input;

  const preview =
    previewState.raw;

  const token =
    preview.token;

  const tokenBlueprintPatch =
    preview.tokenBlueprintPatch ??
    previewState.tokenBlueprintPatch;

  const tokenName =
    normalizeText(
      tokenBlueprintPatch?.tokenName,
    );

  const tokenIconUrl =
    normalizeText(
      previewState.tokenIconUrlEncoded,
    ) ||
    normalizeText(
      tokenBlueprintPatch?.tokenIcon,
    );

  const tokenBrandName =
    normalizeText(
      tokenBlueprintPatch?.brandName,
    );

  const tokenCompanyName =
    normalizeText(
      tokenBlueprintPatch?.companyName,
    );

  const tokenDescription =
    normalizeText(
      tokenBlueprintPatch?.description,
    );

  const mintAddress =
    normalizeText(
      token?.mintAddress,
    );

  const hasTokenInfo = Boolean(
    tokenName ||
      tokenIconUrl ||
      tokenBrandName ||
      tokenCompanyName ||
      tokenDescription,
  );

  if (!hasTokenInfo) {
    return null;
  }

  return {
    tokenName,
    tokenIconUrl,
    tokenBrandName,
    tokenCompanyName,
    tokenDescription,

    mintAddress,
    canOpenTokenContents:
      ownedByWallet === true &&
      Boolean(tokenName) &&
      Boolean(mintAddress),
  };
}

export function createScanResultPageViewModel(
  input: CreateScanResultPageViewModelInput,
): ScanResultPageViewModel | null {
  const previewState =
    input.previewState;

  if (!previewState) {
    return null;
  }

  const preview =
    previewState.raw;

  const patch =
    preview.productBlueprintPatch;

  const token =
    preview.token;

  const tokenBlueprintPatch =
    preview.tokenBlueprintPatch ??
    previewState.tokenBlueprintPatch;

  const productId =
    normalizeText(
      preview.productId,
    );

  const productBlueprintId =
    normalizeText(
      preview.productBlueprintId,
    );

  const productName =
    normalizeText(
      patch?.productName,
    );

  const modelNumber =
    normalizeText(
      preview.modelNumber,
    );

  const brandId =
    normalizeText(
      patch?.brandId,
    ) ||
    normalizeText(
      token?.brandId,
    );

  const brandName =
    normalizeText(
      preview.brandName,
    ) ||
    normalizeText(
      token?.brandName,
    ) ||
    normalizeText(
      tokenBlueprintPatch?.brandName,
    );

  const size =
    normalizeText(
      preview.size,
    );

  const color =
    normalizeText(
      preview.color,
    );

  const alcoholInfo =
    createScanAlcoholInfo({
      categoryFields:
        patch?.categoryFields,
      volumeValue:
        preview.volumeValue,
      volumeUnit:
        preview.volumeUnit,
      modelLabel:
        preview.modelLabel,
      modelKind:
        preview.modelKind,
      productBlueprintCategoryKind:
        preview.productBlueprintCategoryKind,
      productBlueprintCategory:
        preview.productBlueprintCategory,
      categoryInputSchema:
        preview.categoryInputSchema,
    });

  return {
    product: {
      productId,
      productBlueprintId,
      title:
        productName ||
        modelNumber ||
        productId ||
        "Scan Result",

      ownerLabel:
        resolveOwnerLabel(
          preview.owner,
        ),

      brandId,
      brandName,
      hasBrandInfo: Boolean(
        brandId ||
        brandName,
      ),

      productBlueprintRows:
        createProductBlueprintRows(
          patch,
        ),

      qualityAssuranceTabs:
        createQualityAssuranceTabs(
          patch,
        ),

      modelNumber,
      size,
      color,
      swatch:
        rgbToCssColor(
          preview.rgb,
        ),

      measurementEntries:
        createMeasurementEntries(
          preview.measurements,
        ),

      alcoholInfo,
    },

    token:
      createTokenViewModel({
        previewState,
        ownedByWallet:
          input.ownedByWallet,
      }),
  };
}