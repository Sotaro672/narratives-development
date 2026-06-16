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
import { isRecord, tokenBlueprintPatchHasAnyField } from "../utils/format";

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

function textValue(value: unknown): string {
  if (value == null) return "";
  return String(value);
}

export function boolValue(value: unknown): boolean {
  if (value == null) return false;
  if (typeof value === "boolean") return value;
  if (typeof value === "number") return value !== 0;

  const s = String(value).toLowerCase();
  return s === "true" || s === "1" || s === "yes";
}

export function intValue(value: unknown): number {
  if (value == null) return 0;

  if (typeof value === "number" && Number.isFinite(value)) {
    return Math.trunc(value);
  }

  const s = String(value);
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

  const s = String(value);
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

  const values = raw.map(textValue).filter(Boolean);

  return values.length > 0 ? values : undefined;
}

function categoryKindFromJson(raw: unknown): ProductCategoryKind | undefined {
  const value = textValue(raw);

  return value ? value : undefined;
}

function productBlueprintCategorySnapshotFromJson(
  raw: unknown,
): ProductBlueprintCategorySnapshot | null {
  if (!isRecord(raw)) {
    return null;
  }

  const snapshot: ProductBlueprintCategorySnapshot = {
    ID: textValue(raw.ID) || textValue(raw.id),
    Code: textValue(raw.Code) || textValue(raw.code),
    NameJa: textValue(raw.NameJa) || textValue(raw.nameJa),
    NameEn: textValue(raw.NameEn) || textValue(raw.nameEn),
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

  const key = textValue(raw.key);
  const label = textValue(raw.label);

  if (!key && !label) {
    return null;
  }

  return {
    scope: textValue(raw.scope),
    key,
    label,
    type: textValue(raw.type),
    required: boolValue(raw.required),
    unit: textValue(raw.unit) || undefined,
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
    categoryCode: textValue(raw.categoryCode),
    categoryKind: textValue(raw.categoryKind),
    categoryNameJa: textValue(raw.categoryNameJa),
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

  const type = textValue(raw.Type) || textValue(raw.type);

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
      const modelId = textValue(item.ModelID) || textValue(item.modelId);
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

    productName: textValue(raw.productName) || undefined,
    description: textValue(raw.description) || undefined,
    brandId: textValue(raw.brandId) || undefined,
    companyId: textValue(raw.companyId) || undefined,

    productBlueprintCategory: productBlueprintCategory ?? undefined,
    categoryFields: categoryFields ?? undefined,

    productIdTag: productIdTagFromJson(raw.productIdTag),
    assigneeId: textValue(raw.assigneeId) || undefined,
    modelRefs: modelRefsFromJson(raw.modelRefs),
  };

  return patch;
}

export function mallOwnerInfoFromJson(raw: unknown): MallOwnerInfo {
  const j = unwrapData(raw);

  return {
    brandId: textValue(j.brandId),
    avatarId: textValue(j.avatarId),
    brandName: textValue(j.brandName),
    avatarName: textValue(j.avatarName),
  };
}

export function mallModelTokenPairFromJson(
  raw: unknown,
): MallModelTokenPair | null {
  if (!isRecord(raw)) return null;

  return {
    modelId: textValue(raw.modelId),
    tokenBlueprintId: textValue(raw.tokenBlueprintId),
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
    avatarId: textValue(j.avatarId),
    productId: textValue(j.productId),
    scannedModelId: textValue(j.scannedModelId),
    scannedTokenBlueprintId: textValue(j.scannedTokenBlueprintId),
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
  const transferredAt = textValue(j.transferredAt);

  return {
    transferredAt: transferredAt || null,

    fromWalletAddress: textValue(j.fromWalletAddress),
    toWalletAddress: textValue(j.toWalletAddress),

    fromAvatarId: textValue(j.fromAvatarId),
    fromAvatarName: textValue(j.fromAvatarName),
    fromAvatarIcon: textValue(j.fromAvatarIcon),
    fromBrandId: textValue(j.fromBrandId),
    fromBrandName: textValue(j.fromBrandName),
    fromBrandIcon: textValue(j.fromBrandIcon),

    toAvatarId: textValue(j.toAvatarId),
    toAvatarName: textValue(j.toAvatarName),
    toAvatarIcon: textValue(j.toAvatarIcon),
    toBrandId: textValue(j.toBrandId),
    toBrandName: textValue(j.toBrandName),
    toBrandIcon: textValue(j.toBrandIcon),
  };
}

export function mallTokenInfoFromJson(raw: unknown): MallTokenInfo | null {
  if (!isRecord(raw)) return null;

  const j = unwrapData(raw);

  return {
    productId: textValue(j.productId),
    brandId: textValue(j.brandId),
    brandName: textValue(j.brandName),
    tokenBlueprintId: textValue(j.tokenBlueprintId),
    toAddress: textValue(j.toAddress),
    metadataUri: textValue(j.metadataUri),
    mintAddress: textValue(j.mintAddress),
    onChainTxSignature: textValue(j.onChainTxSignature),
    mintedAt: textValue(j.mintedAt),
  };
}

export function measurementsFromJson(
  raw: unknown,
): Record<string, number> | null {
  if (!isRecord(raw)) return null;

  const out: Record<string, number> = {};

  Object.entries(raw).forEach(([key, value]) => {
    const k = textValue(key);
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
    ? textValue(product.id) || textValue(product.productId)
    : "";

  const productId = textValue(j.productId) || nestedProductId || textValue(j.id);

  const productBlueprintId =
    textValue(j.productBlueprintId) ||
    (product ? textValue(product.productBlueprintId) : "");

  const modelId = textValue(j.modelId) || (product ? textValue(product.modelId) : "");

  const modelKind =
    textValue(j.modelKind) || (product ? textValue(product.modelKind) : "");

  const modelNumber =
    textValue(j.modelNumber) || (product ? textValue(product.modelNumber) : "");

  const modelLabel =
    textValue(j.modelLabel) || (product ? textValue(product.modelLabel) : "");

  const size = textValue(j.size) || (product ? textValue(product.size) : "");
  const color = textValue(j.color) || (product ? textValue(product.color) : "");

  const rootRgb = intValue(j.rgb);
  const rgb = rootRgb !== 0 ? rootRgb : product ? intValue(product.rgb) : 0;

  const measurements =
    measurementsFromJson(j.measurements) ||
    (product ? measurementsFromJson(product.measurements) : null);

  const volumeValue =
    optionalIntValue(j.volumeValue) ??
    (product ? optionalIntValue(product.volumeValue) : undefined);

  const volumeUnit =
    textValue(j.volumeUnit) || (product ? textValue(product.volumeUnit) : "");

  const productBlueprintCategoryCode =
    textValue(j.productBlueprintCategoryCode) ||
    (product ? textValue(product.productBlueprintCategoryCode) : "");

  const productBlueprintCategoryKind =
    textValue(j.productBlueprintCategoryKind) ||
    (product ? textValue(product.productBlueprintCategoryKind) : "");

  const productBlueprintCategoryName =
    textValue(j.productBlueprintCategoryName) ||
    (product ? textValue(product.productBlueprintCategoryName) : "");

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

  const tokenBlueprintPatch =
    tokenBlueprintPatchVMFromMap(j.tokenBlueprintPatch) ||
    (product ? tokenBlueprintPatchVMFromMap(product.tokenBlueprintPatch) : null);

  const brandName =
    textValue(j.brandName) ||
    (product ? textValue(product.brandName) : "") ||
    token?.brandName ||
    tokenBlueprintPatch?.brandName ||
    "";

  const companyName =
    textValue(j.companyName) ||
    (product ? textValue(product.companyName) : "") ||
    tokenBlueprintPatch?.companyName ||
    "";

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
    brandName,
    companyName,
    token,
    owner,
    transfers: rootTransfers.length > 0 ? rootTransfers : productTransfers,
    tokenBlueprintPatch,
  };
}

export function mallTransferFlowStepFromJson(
  raw: unknown,
): MallTransferFlowStep | null {
  if (!isRecord(raw)) return null;

  return {
    no: intValue(raw.no),
    title: textValue(raw.title),
    note: textValue(raw.note),
  };
}

export function mallScanTransferResponseFromJson(
  raw: unknown,
): MallScanTransferResponse {
  const j = unwrapData(raw);

  return {
    avatarId: textValue(j.avatarId),
    productId: textValue(j.productId),
    matched: boolValue(j.matched),
    txSignature: textValue(j.txSignature),
    fromWallet: textValue(j.fromWallet),
    toWallet: textValue(j.toWallet),
    updatedToAddress: boolValue(j.updatedToAddress),
    mintAddress: textValue(j.mintAddress),
    flow: Array.isArray(j.flow)
      ? j.flow
          .map(mallTransferFlowStepFromJson)
          .filter((v): v is MallTransferFlowStep => Boolean(v))
      : [],
    fromDisplayName: textValue(j.fromDisplayName),
    toDisplayName: textValue(j.toDisplayName),
  };
}

export function tokenBlueprintPatchVMFromMap(
  raw: unknown,
): TokenBlueprintPatchVM | null {
  if (!isRecord(raw)) return null;

  const tokenIcon =
    textValue(raw.tokenIcon) || textValue(raw.iconUrl) || textValue(raw.icon);

  const vm: TokenBlueprintPatchVM = {
    id: textValue(raw.id),
    tokenName: textValue(raw.tokenName) || textValue(raw.name),
    symbol: textValue(raw.symbol),
    brandName: textValue(raw.brandName),
    companyName: textValue(raw.companyName),
    description: textValue(raw.description),
    tokenIcon,
  };

  return tokenBlueprintPatchHasAnyField(vm) ? vm : null;
}

export function catalogReviewFromJson(raw: unknown): CatalogReview | null {
  if (!isRecord(raw)) return null;

  return {
    id: textValue(raw.id),
    productBlueprintId: textValue(raw.productBlueprintId),
    avatarId: textValue(raw.avatarId),
    avatarName: textValue(raw.avatarName),
    avatarIcon: textValue(raw.avatarIcon),
    rating: intValue(raw.rating),
    title: textValue(raw.title),
    body: textValue(raw.body),
    helpfulVotes: intValue(raw.helpfulVotes),
    totalVotes: intValue(raw.totalVotes),
    reviewedAt: textValue(raw.reviewedAt || raw.createdAt),
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
    id: textValue(raw.id),
    name: textValue(raw.name || raw.fileName),
    viewUri: textValue(raw.viewUri || raw.url),
    contentType: textValue(raw.contentType),
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
    productId: textValue(root.productId),
    brandId: textValue(root.brandId),
    brandName: textValue(root.brandName),
    productBlueprintId: textValue(root.productBlueprintId),
    productName: textValue(root.productName),
    metadataUri: textValue(root.metadataUri),
    mintAddress: textValue(root.mintAddress),
    tokenBlueprintId: textValue(root.tokenBlueprintId),
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
      ? firstWallet.Tokens.map(textValue).filter(Boolean)
      : firstWallet && Array.isArray(firstWallet.tokens)
        ? firstWallet.tokens.map(textValue).filter(Boolean)
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
    mintAddress: textValue(root.mintAddress) || fallbackMintAddress,
    tokenContentsFiles: rawFiles
      .map(tokenContentFileFromJson)
      .filter((v): v is TokenContentFile => Boolean(v)),
  };
}