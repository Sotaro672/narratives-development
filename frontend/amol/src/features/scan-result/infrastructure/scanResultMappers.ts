// frontend/amol/src/features/scan-result/infrastructure/scanResultMappers.ts
import type {
  CatalogReview,
  CatalogReviewPage,
  CategoryInputFieldDefinition,
  CategoryInputSchema,
  MallModelTokenPair,
  MallOwnerInfo,
  MallPreviewResponse,
  MallPreviewTransferInfo,
  MallScanTransferResponse,
  MallScanVerifyResponse,
  MallTokenInfo,
  MallTransferFlowStep,
  ProductBlueprintCategorySnapshot,
  ProductBlueprintPatch,
  ProductCategoryKind,
  TokenBlueprintPatchVM,
  TokenContentFile,
  TokenResolveDTO,
  WalletDTO,
} from "../types";
import {
  isRecord,
  trimText,
  tokenBlueprintPatchHasAnyField,
} from "../utils/format";

export type WalletResolvedTokenResponse = {
  productId: string;
  brandId: string;
  brandName: string;
  productBlueprintId: string;
  productName: string;
  metadataUri: string;
  mintAddress: string;
  tokenBlueprintId: string;
  tokenContentsFiles: TokenContentFile[];
};

export function boolValue(value: unknown): boolean {
  if (value == null) return false;
  if (typeof value === "boolean") return value;
  if (typeof value === "number") return value !== 0;

  const s = String(value).trim().toLowerCase();
  return s === "true" || s === "1" || s === "yes";
}

export function intValue(value: unknown): number {
  if (value == null) return 0;

  if (typeof value === "number" && Number.isFinite(value)) {
    return Math.trunc(value);
  }

  const s = String(value).trim();
  if (!s) return 0;

  if (s.startsWith("0x") || s.startsWith("0X")) {
    const parsed = Number.parseInt(s.slice(2), 16);
    return Number.isFinite(parsed) ? parsed : 0;
  }

  const parsed = Number.parseInt(s, 10);
  return Number.isFinite(parsed) ? parsed : 0;
}

export function optionalIntValue(value: unknown): number | undefined {
  if (value == null) return undefined;

  if (typeof value === "number" && Number.isFinite(value)) {
    return Math.trunc(value);
  }

  const s = String(value).trim();
  if (!s) return undefined;

  if (s.startsWith("0x") || s.startsWith("0X")) {
    const parsed = Number.parseInt(s.slice(2), 16);
    return Number.isFinite(parsed) ? parsed : undefined;
  }

  const parsed = Number.parseInt(s, 10);
  return Number.isFinite(parsed) ? parsed : undefined;
}

export function unwrapData(value: unknown): Record<string, unknown> {
  if (!isRecord(value)) {
    throw new Error("invalid json shape (expected object)");
  }

  const data = value.data;
  if (isRecord(data)) return unwrapData(data);

  return value;
}

export function objectOrNull(raw: unknown): Record<string, unknown> | null {
  return isRecord(raw) ? raw : null;
}

function stringArrayFromJson(raw: unknown): string[] | undefined {
  if (!Array.isArray(raw)) {
    return undefined;
  }

  const values = raw.map(trimText).filter(Boolean);

  return values.length > 0 ? values : undefined;
}

function categoryKindFromJson(raw: unknown): ProductCategoryKind | undefined {
  const value = trimText(raw);

  return value ? value : undefined;
}

function productBlueprintCategorySnapshotFromJson(
  raw: unknown,
): ProductBlueprintCategorySnapshot | null {
  if (!isRecord(raw)) {
    return null;
  }

  const snapshot: ProductBlueprintCategorySnapshot = {
    ID: trimText(raw.ID) || trimText(raw.id),
    Code: trimText(raw.Code) || trimText(raw.code),
    NameJa: trimText(raw.NameJa) || trimText(raw.nameJa),
    NameEn: trimText(raw.NameEn) || trimText(raw.nameEn),
    Kind: categoryKindFromJson(raw.Kind) || categoryKindFromJson(raw.kind),
    Path: stringArrayFromJson(raw.Path) || stringArrayFromJson(raw.path),
  };

  const hasAnyField = Boolean(
    snapshot.ID ||
      snapshot.Code ||
      snapshot.NameJa ||
      snapshot.NameEn ||
      snapshot.Kind ||
      (snapshot.Path && snapshot.Path.length > 0),
  );

  return hasAnyField ? snapshot : null;
}

function categoryInputFieldDefinitionFromJson(
  raw: unknown,
): CategoryInputFieldDefinition | null {
  if (!isRecord(raw)) {
    return null;
  }

  const key = trimText(raw.key);
  const label = trimText(raw.label);

  if (!key && !label) {
    return null;
  }

  return {
    scope: trimText(raw.scope),
    key,
    label,
    type: trimText(raw.type),
    required: boolValue(raw.required),
    unit: trimText(raw.unit) || undefined,
  };
}

function categoryInputFieldDefinitionsFromJson(
  raw: unknown,
): CategoryInputFieldDefinition[] {
  if (!Array.isArray(raw)) {
    return [];
  }

  return raw
    .map(categoryInputFieldDefinitionFromJson)
    .filter((item): item is CategoryInputFieldDefinition => Boolean(item));
}

function categoryInputSchemaFromJson(raw: unknown): CategoryInputSchema | null {
  if (!isRecord(raw)) {
    return null;
  }

  const schema: CategoryInputSchema = {
    categoryCode: trimText(raw.categoryCode),
    categoryKind: trimText(raw.categoryKind),
    categoryNameJa: trimText(raw.categoryNameJa),
    productBlueprintFields: categoryInputFieldDefinitionsFromJson(
      raw.productBlueprintFields,
    ),
    modelFields: categoryInputFieldDefinitionsFromJson(raw.modelFields),
  };

  const hasAnyField = Boolean(
    schema.categoryCode ||
      schema.categoryKind ||
      schema.categoryNameJa ||
      schema.productBlueprintFields.length > 0 ||
      schema.modelFields.length > 0,
  );

  return hasAnyField ? schema : null;
}

function productIdTagFromJson(
  raw: unknown,
): ProductBlueprintPatch["productIdTag"] | undefined {
  if (!isRecord(raw)) {
    return undefined;
  }

  const type = trimText(raw.Type) || trimText(raw.type);

  if (!type) {
    return undefined;
  }

  return {
    Type: type,
    type,
  };
}

function modelRefsFromJson(raw: unknown): ProductBlueprintPatch["modelRefs"] {
  if (!Array.isArray(raw)) {
    return undefined;
  }

  const refs = raw
    .filter(isRecord)
    .map((item) => {
      const modelId = trimText(item.ModelID) || trimText(item.modelId);
      const displayOrder =
        optionalIntValue(item.DisplayOrder) ?? optionalIntValue(item.displayOrder);

      if (!modelId && typeof displayOrder !== "number") {
        return null;
      }

      return {
        ModelID: modelId || undefined,
        modelId: modelId || undefined,
        DisplayOrder: displayOrder,
        displayOrder,
      };
    })
    .filter((item): item is NonNullable<typeof item> => Boolean(item));

  return refs.length > 0 ? refs : undefined;
}

export function productBlueprintPatchFromJson(
  raw: unknown,
): ProductBlueprintPatch | null {
  if (!isRecord(raw)) {
    return null;
  }

  const categoryFields = objectOrNull(raw.categoryFields);
  const productBlueprintCategory = productBlueprintCategorySnapshotFromJson(
    raw.productBlueprintCategory,
  );

  const patch: ProductBlueprintPatch = {
    ...raw,

    productName: trimText(raw.productName) || undefined,
    description: trimText(raw.description) || undefined,
    brandId: trimText(raw.brandId) || undefined,
    companyId: trimText(raw.companyId) || undefined,

    productBlueprintCategory: productBlueprintCategory ?? undefined,
    categoryFields: categoryFields ?? undefined,

    productIdTag: productIdTagFromJson(raw.productIdTag),
    assigneeId: trimText(raw.assigneeId) || undefined,
    modelRefs: modelRefsFromJson(raw.modelRefs),
  };

  return patch;
}

export function mallOwnerInfoFromJson(raw: unknown): MallOwnerInfo {
  const j = unwrapData(raw);

  return {
    brandId: trimText(j.brandId),
    avatarId: trimText(j.avatarId),
    brandName: trimText(j.brandName),
    avatarName: trimText(j.avatarName),
  };
}

export function mallModelTokenPairFromJson(
  raw: unknown,
): MallModelTokenPair | null {
  if (!isRecord(raw)) return null;

  return {
    modelId: trimText(raw.modelId),
    tokenBlueprintId: trimText(raw.tokenBlueprintId),
  };
}

export function mallScanVerifyResponseFromJson(
  raw: unknown,
): MallScanVerifyResponse {
  const j = unwrapData(raw);

  const purchasedPairs = Array.isArray(j.purchasedPairs)
    ? j.purchasedPairs
        .map(mallModelTokenPairFromJson)
        .filter((v): v is MallModelTokenPair => Boolean(v))
    : [];

  return {
    avatarId: trimText(j.avatarId),
    productId: trimText(j.productId),
    scannedModelId: trimText(j.scannedModelId),
    scannedTokenBlueprintId: trimText(j.scannedTokenBlueprintId),
    purchasedPairs,
    matched: boolValue(j.matched),
    match: mallModelTokenPairFromJson(j.match),
  };
}

export function mallPreviewTransferInfoFromJson(
  raw: unknown,
): MallPreviewTransferInfo | null {
  if (!isRecord(raw)) return null;

  const j = unwrapData(raw);
  const transferredAt = trimText(j.transferredAt);

  return {
    transferredAt: transferredAt || null,

    fromWalletAddress: trimText(j.fromWalletAddress),
    toWalletAddress: trimText(j.toWalletAddress),

    fromAvatarId: trimText(j.fromAvatarId),
    fromAvatarName: trimText(j.fromAvatarName),
    fromAvatarIcon: trimText(j.fromAvatarIcon),
    fromBrandId: trimText(j.fromBrandId),
    fromBrandName: trimText(j.fromBrandName),
    fromBrandIcon: trimText(j.fromBrandIcon),

    toAvatarId: trimText(j.toAvatarId),
    toAvatarName: trimText(j.toAvatarName),
    toAvatarIcon: trimText(j.toAvatarIcon),
    toBrandId: trimText(j.toBrandId),
    toBrandName: trimText(j.toBrandName),
    toBrandIcon: trimText(j.toBrandIcon),
  };
}

export function mallTokenInfoFromJson(raw: unknown): MallTokenInfo | null {
  if (!isRecord(raw)) return null;

  const j = unwrapData(raw);

  return {
    productId: trimText(j.productId),
    brandId: trimText(j.brandId),
    brandName: trimText(j.brandName),
    tokenBlueprintId: trimText(j.tokenBlueprintId),
    toAddress: trimText(j.toAddress),
    metadataUri: trimText(j.metadataUri),
    mintAddress: trimText(j.mintAddress),
    onChainTxSignature: trimText(j.onChainTxSignature),
    mintedAt: trimText(j.mintedAt),
  };
}

export function measurementsFromJson(
  raw: unknown,
): Record<string, number> | null {
  if (!isRecord(raw)) return null;

  const out: Record<string, number> = {};

  Object.entries(raw).forEach(([key, value]) => {
    const k = trimText(key);
    if (!k) return;
    out[k] = intValue(value);
  });

  return Object.keys(out).length > 0 ? out : null;
}

export function previewTransfersFromJson(
  raw: unknown,
): MallPreviewTransferInfo[] {
  if (!Array.isArray(raw)) return [];

  return raw
    .map(mallPreviewTransferInfoFromJson)
    .filter((v): v is MallPreviewTransferInfo => Boolean(v));
}

export function mallPreviewResponseFromJson(
  raw: unknown,
): MallPreviewResponse {
  const j = unwrapData(raw);
  const product = isRecord(j.product) ? j.product : null;

  const nestedProductId = product
    ? trimText(product.id) || trimText(product.productId)
    : "";

  const productId = trimText(j.productId) || nestedProductId || trimText(j.id);

  const productBlueprintId =
    trimText(j.productBlueprintId) ||
    (product ? trimText(product.productBlueprintId) : "");

  const modelId = trimText(j.modelId) || (product ? trimText(product.modelId) : "");

  const modelKind =
    trimText(j.modelKind) || (product ? trimText(product.modelKind) : "");

  const modelNumber =
    trimText(j.modelNumber) || (product ? trimText(product.modelNumber) : "");

  const modelLabel =
    trimText(j.modelLabel) || (product ? trimText(product.modelLabel) : "");

  const size = trimText(j.size) || (product ? trimText(product.size) : "");
  const color = trimText(j.color) || (product ? trimText(product.color) : "");

  const rootRgb = intValue(j.rgb);
  const rgb = rootRgb !== 0 ? rootRgb : product ? intValue(product.rgb) : 0;

  const measurements =
    measurementsFromJson(j.measurements) ||
    (product ? measurementsFromJson(product.measurements) : null);

  const volumeValue =
    optionalIntValue(j.volumeValue) ??
    (product ? optionalIntValue(product.volumeValue) : undefined);

  const volumeUnit =
    trimText(j.volumeUnit) || (product ? trimText(product.volumeUnit) : "");

  const productBlueprintCategoryCode =
    trimText(j.productBlueprintCategoryCode) ||
    (product ? trimText(product.productBlueprintCategoryCode) : "");

  const productBlueprintCategoryKind =
    trimText(j.productBlueprintCategoryKind) ||
    (product ? trimText(product.productBlueprintCategoryKind) : "");

  const productBlueprintCategoryName =
    trimText(j.productBlueprintCategoryName) ||
    (product ? trimText(product.productBlueprintCategoryName) : "");

  const productBlueprintCategory =
    productBlueprintCategorySnapshotFromJson(j.productBlueprintCategory) ||
    (product
      ? productBlueprintCategorySnapshotFromJson(product.productBlueprintCategory)
      : null);

  const categoryInputSchema =
    categoryInputSchemaFromJson(j.categoryInputSchema) ||
    (product ? categoryInputSchemaFromJson(product.categoryInputSchema) : null);

  const productBlueprintPatch =
    productBlueprintPatchFromJson(j.productBlueprintPatch) ||
    (product ? productBlueprintPatchFromJson(product.productBlueprintPatch) : null);

  const token =
    mallTokenInfoFromJson(j.token) ||
    (product ? mallTokenInfoFromJson(product.token) : null);

  const owner =
    (isRecord(j.owner) ? mallOwnerInfoFromJson(j.owner) : null) ||
    (product && isRecord(product.owner)
      ? mallOwnerInfoFromJson(product.owner)
      : null);

  const rootTransfers = previewTransfersFromJson(j.transfers);
  const productTransfers = product ? previewTransfersFromJson(product.transfers) : [];

  return {
    productId,
    productBlueprintId,
    modelId,
    modelKind,
    modelNumber,
    modelLabel,
    size,
    color,
    rgb,
    measurements,
    volumeValue,
    volumeUnit,
    productBlueprintCategoryCode,
    productBlueprintCategoryKind,
    productBlueprintCategoryName,
    productBlueprintCategory,
    categoryInputSchema,
    productBlueprintPatch,
    token,
    owner,
    transfers: rootTransfers.length > 0 ? rootTransfers : productTransfers,
  };
}

export function mallTransferFlowStepFromJson(
  raw: unknown,
): MallTransferFlowStep | null {
  if (!isRecord(raw)) return null;

  return {
    no: intValue(raw.no),
    title: trimText(raw.title),
    note: trimText(raw.note),
  };
}

export function mallScanTransferResponseFromJson(
  raw: unknown,
): MallScanTransferResponse {
  const j = unwrapData(raw);

  return {
    avatarId: trimText(j.avatarId),
    productId: trimText(j.productId),
    matched: boolValue(j.matched),
    txSignature: trimText(j.txSignature),
    fromWallet: trimText(j.fromWallet),
    toWallet: trimText(j.toWallet),
    updatedToAddress: boolValue(j.updatedToAddress),
    mintAddress: trimText(j.mintAddress),
    flow: Array.isArray(j.flow)
      ? j.flow
          .map(mallTransferFlowStepFromJson)
          .filter((v): v is MallTransferFlowStep => Boolean(v))
      : [],
    fromDisplayName: trimText(j.fromDisplayName),
    toDisplayName: trimText(j.toDisplayName),
  };
}

export function tokenBlueprintPatchVMFromMap(
  raw: unknown,
): TokenBlueprintPatchVM | null {
  if (!isRecord(raw)) return null;

  const tokenIcon =
    trimText(raw.tokenIcon) || trimText(raw.iconUrl) || trimText(raw.icon);

  const vm: TokenBlueprintPatchVM = {
    id: trimText(raw.id),
    tokenName: trimText(raw.tokenName) || trimText(raw.name),
    symbol: trimText(raw.symbol),
    brandName: trimText(raw.brandName),
    companyName: trimText(raw.companyName),
    description: trimText(raw.description),
    tokenIcon,
  };

  return tokenBlueprintPatchHasAnyField(vm) ? vm : null;
}

export function catalogReviewFromJson(raw: unknown): CatalogReview | null {
  if (!isRecord(raw)) return null;

  return {
    id: trimText(raw.id),
    productBlueprintId: trimText(raw.productBlueprintId),
    avatarId: trimText(raw.avatarId),
    avatarName: trimText(raw.avatarName),
    avatarIcon: trimText(raw.avatarIcon),
    rating: intValue(raw.rating),
    title: trimText(raw.title),
    body: trimText(raw.body),
    helpfulVotes: intValue(raw.helpfulVotes),
    totalVotes: intValue(raw.totalVotes),
    reviewedAt: trimText(raw.reviewedAt || raw.createdAt),
  };
}

export function catalogReviewPageFromJson(
  raw: unknown,
  fallbackPage: number,
  fallbackPerPage: number,
): CatalogReviewPage {
  const root = unwrapData(raw);

  const rawItems = Array.isArray(root.items)
    ? root.items
    : Array.isArray(root.reviews)
      ? root.reviews
      : [];

  const items = rawItems
    .map(catalogReviewFromJson)
    .filter((v): v is CatalogReview => Boolean(v));

  return {
    items,
    page: intValue(root.page) || fallbackPage,
    perPage: intValue(root.perPage) || fallbackPerPage,
    total: intValue(root.total),
    hasNext: boolValue(root.hasNext),
  };
}

export function tokenContentFileFromJson(
  raw: unknown,
): TokenContentFile | null {
  if (!isRecord(raw)) return null;

  return {
    id: trimText(raw.id),
    name: trimText(raw.name || raw.fileName),
    viewUri: trimText(raw.viewUri || raw.url),
    contentType: trimText(raw.contentType),
    isPreviewable: boolValue(raw.isPreviewable),
  };
}

export function walletResolvedTokenResponseFromJson(
  raw: unknown,
): WalletResolvedTokenResponse {
  const root = unwrapData(raw);

  const rawFiles = Array.isArray(root.tokenContentsFiles)
    ? root.tokenContentsFiles
    : Array.isArray(root.files)
      ? root.files
      : [];

  return {
    productId: trimText(root.productId),
    brandId: trimText(root.brandId),
    brandName: trimText(root.brandName),
    productBlueprintId: trimText(root.productBlueprintId),
    productName: trimText(root.productName),
    metadataUri: trimText(root.metadataUri),
    mintAddress: trimText(root.mintAddress),
    tokenBlueprintId: trimText(root.tokenBlueprintId),
    tokenContentsFiles: rawFiles
      .map(tokenContentFileFromJson)
      .filter((v): v is TokenContentFile => Boolean(v)),
  };
}

export function walletDTOFromJson(raw: unknown): WalletDTO {
  const root = unwrapData(raw);
  const walletsRaw = Array.isArray(root.wallets) ? root.wallets : [];
  const firstWallet = walletsRaw.find(isRecord);

  const tokens =
    firstWallet && Array.isArray(firstWallet.Tokens)
      ? firstWallet.Tokens.map(trimText).filter(Boolean)
      : firstWallet && Array.isArray(firstWallet.tokens)
        ? firstWallet.tokens.map(trimText).filter(Boolean)
        : [];

  return { tokens };
}

export function tokenResolveDTOFromJson(
  raw: unknown,
  fallbackMintAddress: string,
): TokenResolveDTO {
  const root = unwrapData(raw);

  const rawFiles = Array.isArray(root.tokenContentsFiles)
    ? root.tokenContentsFiles
    : Array.isArray(root.files)
      ? root.files
      : [];

  return {
    mintAddress: trimText(root.mintAddress) || fallbackMintAddress,
    tokenContentsFiles: rawFiles
      .map(tokenContentFileFromJson)
      .filter((v): v is TokenContentFile => Boolean(v)),
  };
}