// frontend/amol/src/features/scan-result/infrastructure/scanResultMappers.ts

import type {
  CatalogReview,
  CatalogReviewPage,
  CategoryInputFieldDefinition,
  CategoryInputSchema,
  MallOwnerInfo,
  MallPreviewResponse,
  MallPreviewTransferInfo,
  MallScanTransferResponse,
  MallTokenInfo,
  MallTransferFlowStep,
  ProductBlueprintCategorySnapshot,
  ProductBlueprintPatch,
  ProductCategoryKind,
  PreviewState,
  TokenBlueprintPatchVM,
  TokenContentFile,
  WalletDTO,
} from "../../shared/types/scanResult";
import {
  isFiniteNumber,
  isRecord,
} from "../../../components/utils/typeGuards";
import {
  safeUrl,
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

function textValue(value: unknown): string {
  if (value == null) return "";

  return String(value);
}

export function boolValue(value: unknown): boolean {
  if (value == null) return false;
  if (typeof value === "boolean") return value;
  if (typeof value === "number") return value !== 0;

  const text = String(value).toLowerCase();

  return (
    text === "true" ||
    text === "1" ||
    text === "yes"
  );
}

export function intValue(value: unknown): number {
  if (value == null) return 0;

  if (isFiniteNumber(value)) {
    return Math.trunc(value);
  }

  const text = String(value);

  if (!text) return 0;

  if (
    text.startsWith("0x") ||
    text.startsWith("0X")
  ) {
    const parsed = Number.parseInt(
      text.slice(2),
      16,
    );

    return isFiniteNumber(parsed)
      ? parsed
      : 0;
  }

  const parsed = Number.parseInt(text, 10);

  return isFiniteNumber(parsed)
    ? parsed
    : 0;
}

export function optionalIntValue(
  value: unknown,
): number | undefined {
  if (value == null) return undefined;

  if (isFiniteNumber(value)) {
    return Math.trunc(value);
  }

  const text = String(value);

  if (!text) return undefined;

  if (
    text.startsWith("0x") ||
    text.startsWith("0X")
  ) {
    const parsed = Number.parseInt(
      text.slice(2),
      16,
    );

    return isFiniteNumber(parsed)
      ? parsed
      : undefined;
  }

  const parsed = Number.parseInt(text, 10);

  return isFiniteNumber(parsed)
    ? parsed
    : undefined;
}

export function unwrapData(
  value: unknown,
): Record<string, unknown> {
  if (!isRecord(value)) {
    throw new Error(
      "invalid json shape (expected object)",
    );
  }

  const data = value.data;

  if (isRecord(data)) {
    return unwrapData(data);
  }

  return value;
}

function stringArrayFromJson(
  raw: unknown,
): string[] | undefined {
  if (!Array.isArray(raw)) {
    return undefined;
  }

  const values = raw
    .map(textValue)
    .filter(Boolean);

  return values.length > 0
    ? values
    : undefined;
}

function categoryKindFromJson(
  raw: unknown,
): ProductCategoryKind | undefined {
  const value = textValue(raw);

  return value
    ? value
    : undefined;
}

function productBlueprintCategorySnapshotFromJson(
  raw: unknown,
): ProductBlueprintCategorySnapshot | null {
  if (!isRecord(raw)) {
    return null;
  }

  const snapshot: ProductBlueprintCategorySnapshot = {
    ID:
      textValue(raw.ID) ||
      textValue(raw.id),
    Code:
      textValue(raw.Code) ||
      textValue(raw.code),
    NameJa:
      textValue(raw.NameJa) ||
      textValue(raw.nameJa),
    NameEn:
      textValue(raw.NameEn) ||
      textValue(raw.nameEn),
    Kind:
      categoryKindFromJson(raw.Kind) ||
      categoryKindFromJson(raw.kind),
    Path:
      stringArrayFromJson(raw.Path) ||
      stringArrayFromJson(raw.path),
  };

  const hasAnyField = Boolean(
    snapshot.ID ||
      snapshot.Code ||
      snapshot.NameJa ||
      snapshot.NameEn ||
      snapshot.Kind ||
      (
        snapshot.Path &&
        snapshot.Path.length > 0
      ),
  );

  return hasAnyField
    ? snapshot
    : null;
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
    unit:
      textValue(raw.unit) ||
      undefined,
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
    .filter(
      (
        item,
      ): item is CategoryInputFieldDefinition =>
        Boolean(item),
    );
}

function categoryInputSchemaFromJson(
  raw: unknown,
): CategoryInputSchema | null {
  if (!isRecord(raw)) {
    return null;
  }

  const schema: CategoryInputSchema = {
    categoryCode: textValue(
      raw.categoryCode,
    ),
    categoryKind: textValue(
      raw.categoryKind,
    ),
    categoryNameJa: textValue(
      raw.categoryNameJa,
    ),
    productBlueprintFields:
      categoryInputFieldDefinitionsFromJson(
        raw.productBlueprintFields,
      ),
    modelFields:
      categoryInputFieldDefinitionsFromJson(
        raw.modelFields,
      ),
  };

  const hasAnyField = Boolean(
    schema.categoryCode ||
      schema.categoryKind ||
      schema.categoryNameJa ||
      schema.productBlueprintFields.length >
        0 ||
      schema.modelFields.length > 0,
  );

  return hasAnyField
    ? schema
    : null;
}

function productIdTagFromJson(
  raw: unknown,
): ProductBlueprintPatch["productIdTag"] | undefined {
  if (!isRecord(raw)) {
    return undefined;
  }

  const type =
    textValue(raw.Type) ||
    textValue(raw.type);

  if (!type) {
    return undefined;
  }

  return {
    Type: type,
    type,
  };
}

function modelRefsFromJson(
  raw: unknown,
): ProductBlueprintPatch["modelRefs"] {
  if (!Array.isArray(raw)) {
    return undefined;
  }

  const refs = raw
    .filter(isRecord)
    .map((item) => {
      const modelId =
        textValue(item.ModelID) ||
        textValue(item.modelId);

      const displayOrder =
        optionalIntValue(
          item.DisplayOrder,
        ) ??
        optionalIntValue(
          item.displayOrder,
        );

      if (
        !modelId &&
        typeof displayOrder !== "number"
      ) {
        return null;
      }

      return {
        ModelID:
          modelId || undefined,
        modelId:
          modelId || undefined,
        DisplayOrder: displayOrder,
        displayOrder,
      };
    })
    .filter(
      (
        item,
      ): item is NonNullable<typeof item> =>
        Boolean(item),
    );

  return refs.length > 0
    ? refs
    : undefined;
}

export function productBlueprintPatchFromJson(
  raw: unknown,
): ProductBlueprintPatch | null {
  if (!isRecord(raw)) {
    return null;
  }

  const categoryFields =
    isRecord(raw.categoryFields)
      ? raw.categoryFields
      : null;

  const productBlueprintCategory =
    productBlueprintCategorySnapshotFromJson(
      raw.productBlueprintCategory,
    );

  const patch: ProductBlueprintPatch = {
    ...raw,

    productName:
      textValue(raw.productName) ||
      undefined,
    description:
      textValue(raw.description) ||
      undefined,
    brandId:
      textValue(raw.brandId) ||
      undefined,
    companyId:
      textValue(raw.companyId) ||
      undefined,

    productBlueprintCategory:
      productBlueprintCategory ??
      undefined,
    categoryFields:
      categoryFields ??
      undefined,

    productIdTag:
      productIdTagFromJson(
        raw.productIdTag,
      ),
    assigneeId:
      textValue(raw.assigneeId) ||
      undefined,
    modelRefs:
      modelRefsFromJson(
        raw.modelRefs,
      ),
  };

  return patch;
}

export function mallOwnerInfoFromJson(
  raw: unknown,
): MallOwnerInfo {
  const json = unwrapData(raw);

  return {
    brandId:
      textValue(json.brandId),
    avatarId:
      textValue(json.avatarId),
    brandName:
      textValue(json.brandName),
    avatarName:
      textValue(json.avatarName),
  };
}

export function mallPreviewTransferInfoFromJson(
  raw: unknown,
): MallPreviewTransferInfo | null {
  if (!isRecord(raw)) {
    return null;
  }

  const json = unwrapData(raw);
  const transferredAt =
    textValue(json.transferredAt);

  return {
    transferredAt:
      transferredAt || null,

    fromWalletAddress:
      textValue(
        json.fromWalletAddress,
      ),
    toWalletAddress:
      textValue(
        json.toWalletAddress,
      ),

    fromAvatarId:
      textValue(json.fromAvatarId),
    fromAvatarName:
      textValue(
        json.fromAvatarName,
      ),
    fromAvatarIcon:
      textValue(
        json.fromAvatarIcon,
      ),
    fromBrandId:
      textValue(json.fromBrandId),
    fromBrandName:
      textValue(
        json.fromBrandName,
      ),
    fromBrandIcon:
      textValue(
        json.fromBrandIcon,
      ),

    toAvatarId:
      textValue(json.toAvatarId),
    toAvatarName:
      textValue(
        json.toAvatarName,
      ),
    toAvatarIcon:
      textValue(
        json.toAvatarIcon,
      ),
    toBrandId:
      textValue(json.toBrandId),
    toBrandName:
      textValue(
        json.toBrandName,
      ),
    toBrandIcon:
      textValue(
        json.toBrandIcon,
      ),
  };
}

export function mallTokenInfoFromJson(
  raw: unknown,
): MallTokenInfo | null {
  if (!isRecord(raw)) {
    return null;
  }

  const json = unwrapData(raw);

  return {
    productId:
      textValue(json.productId),
    brandId:
      textValue(json.brandId),
    brandName:
      textValue(json.brandName),
    tokenBlueprintId:
      textValue(
        json.tokenBlueprintId,
      ),
    toAddress:
      textValue(json.toAddress),
    metadataUri:
      textValue(json.metadataUri),
    mintAddress:
      textValue(json.mintAddress),
    onChainTxSignature:
      textValue(
        json.onChainTxSignature,
      ),
    mintedAt:
      textValue(json.mintedAt),
  };
}

export function measurementsFromJson(
  raw: unknown,
): Record<string, number> | null {
  if (!isRecord(raw)) {
    return null;
  }

  const output: Record<string, number> = {};

  Object.entries(raw).forEach(
    ([key, value]) => {
      const normalizedKey =
        textValue(key);

      if (!normalizedKey) {
        return;
      }

      output[normalizedKey] =
        intValue(value);
    },
  );

  return Object.keys(output).length > 0
    ? output
    : null;
}

export function previewTransfersFromJson(
  raw: unknown,
): MallPreviewTransferInfo[] {
  if (!Array.isArray(raw)) {
    return [];
  }

  return raw
    .map(
      mallPreviewTransferInfoFromJson,
    )
    .filter(
      (
        value,
      ): value is MallPreviewTransferInfo =>
        Boolean(value),
    );
}

export function mallPreviewResponseFromJson(
  raw: unknown,
): MallPreviewResponse {
  const json = unwrapData(raw);

  const product = isRecord(
    json.product,
  )
    ? json.product
    : null;

  const nestedProductId = product
    ? textValue(product.id) ||
      textValue(product.productId)
    : "";

  const productId =
    textValue(json.productId) ||
    nestedProductId ||
    textValue(json.id);

  const productBlueprintId =
    textValue(
      json.productBlueprintId,
    ) ||
    (
      product
        ? textValue(
            product.productBlueprintId,
          )
        : ""
    );

  const modelId =
    textValue(json.modelId) ||
    (
      product
        ? textValue(product.modelId)
        : ""
    );

  const modelKind =
    textValue(json.modelKind) ||
    (
      product
        ? textValue(product.modelKind)
        : ""
    );

  const modelNumber =
    textValue(json.modelNumber) ||
    (
      product
        ? textValue(
            product.modelNumber,
          )
        : ""
    );

  const modelLabel =
    textValue(json.modelLabel) ||
    (
      product
        ? textValue(
            product.modelLabel,
          )
        : ""
    );

  const size =
    textValue(json.size) ||
    (
      product
        ? textValue(product.size)
        : ""
    );

  const color =
    textValue(json.color) ||
    (
      product
        ? textValue(product.color)
        : ""
    );

  const rootRgb =
    intValue(json.rgb);

  const rgb =
    rootRgb !== 0
      ? rootRgb
      : product
        ? intValue(product.rgb)
        : 0;

  const measurements =
    measurementsFromJson(
      json.measurements,
    ) ||
    (
      product
        ? measurementsFromJson(
            product.measurements,
          )
        : null
    );

  const volumeValue =
    optionalIntValue(
      json.volumeValue,
    ) ??
    (
      product
        ? optionalIntValue(
            product.volumeValue,
          )
        : undefined
    );

  const volumeUnit =
    textValue(json.volumeUnit) ||
    (
      product
        ? textValue(
            product.volumeUnit,
          )
        : ""
    );

  const productBlueprintCategoryCode =
    textValue(
      json.productBlueprintCategoryCode,
    ) ||
    (
      product
        ? textValue(
            product.productBlueprintCategoryCode,
          )
        : ""
    );

  const productBlueprintCategoryKind =
    textValue(
      json.productBlueprintCategoryKind,
    ) ||
    (
      product
        ? textValue(
            product.productBlueprintCategoryKind,
          )
        : ""
    );

  const productBlueprintCategoryName =
    textValue(
      json.productBlueprintCategoryName,
    ) ||
    (
      product
        ? textValue(
            product.productBlueprintCategoryName,
          )
        : ""
    );

  const productBlueprintCategory =
    productBlueprintCategorySnapshotFromJson(
      json.productBlueprintCategory,
    ) ||
    (
      product
        ? productBlueprintCategorySnapshotFromJson(
            product.productBlueprintCategory,
          )
        : null
    );

  const categoryInputSchema =
    categoryInputSchemaFromJson(
      json.categoryInputSchema,
    ) ||
    (
      product
        ? categoryInputSchemaFromJson(
            product.categoryInputSchema,
          )
        : null
    );

  const productBlueprintPatch =
    productBlueprintPatchFromJson(
      json.productBlueprintPatch,
    ) ||
    (
      product
        ? productBlueprintPatchFromJson(
            product.productBlueprintPatch,
          )
        : null
    );

  const token =
    mallTokenInfoFromJson(
      json.token,
    ) ||
    (
      product
        ? mallTokenInfoFromJson(
            product.token,
          )
        : null
    );

  const tokenBlueprintPatch =
    tokenBlueprintPatchVMFromMap(
      json.tokenBlueprintPatch,
    ) ||
    (
      product
        ? tokenBlueprintPatchVMFromMap(
            product.tokenBlueprintPatch,
          )
        : null
    );

  const brandName =
    textValue(json.brandName) ||
    (
      product
        ? textValue(product.brandName)
        : ""
    ) ||
    token?.brandName ||
    tokenBlueprintPatch?.brandName ||
    "";

  const companyName =
    textValue(json.companyName) ||
    (
      product
        ? textValue(
            product.companyName,
          )
        : ""
    ) ||
    tokenBlueprintPatch?.companyName ||
    "";

  const owner =
    (
      isRecord(json.owner)
        ? mallOwnerInfoFromJson(
            json.owner,
          )
        : null
    ) ||
    (
      product &&
      isRecord(product.owner)
        ? mallOwnerInfoFromJson(
            product.owner,
          )
        : null
    );

  const rootTransfers =
    previewTransfersFromJson(
      json.transfers,
    );

  const productTransfers = product
    ? previewTransfersFromJson(
        product.transfers,
      )
    : [];

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
    transfers:
      rootTransfers.length > 0
        ? rootTransfers
        : productTransfers,
    tokenBlueprintPatch,
  };
}

export function previewStateFromJson(
  raw: unknown,
): PreviewState {
  const preview =
    mallPreviewResponseFromJson(raw);

  const tokenBlueprintPatch =
    preview.tokenBlueprintPatch ?? null;

  const tokenIcon =
    tokenBlueprintPatch?.tokenIcon.trim() ??
    "";

  return {
    raw: preview,
    tokenBlueprintPatch,
    tokenIconUrlEncoded: tokenIcon
      ? safeUrl(tokenIcon)
      : null,
  };
}

export function mallTransferFlowStepFromJson(
  raw: unknown,
): MallTransferFlowStep | null {
  if (!isRecord(raw)) {
    return null;
  }

  return {
    no: intValue(raw.no),
    title:
      textValue(raw.title),
    note:
      textValue(raw.note),
  };
}

export function mallScanTransferResponseFromJson(
  raw: unknown,
): MallScanTransferResponse {
  const json = unwrapData(raw);

  return {
    avatarId:
      textValue(json.avatarId),
    productId:
      textValue(json.productId),
    matched:
      boolValue(json.matched),
    txSignature:
      textValue(
        json.txSignature,
      ),
    fromWallet:
      textValue(json.fromWallet),
    toWallet:
      textValue(json.toWallet),
    updatedToAddress:
      boolValue(
        json.updatedToAddress,
      ),
    mintAddress:
      textValue(json.mintAddress),
    flow: Array.isArray(json.flow)
      ? json.flow
          .map(
            mallTransferFlowStepFromJson,
          )
          .filter(
            (
              value,
            ): value is MallTransferFlowStep =>
              Boolean(value),
          )
      : [],
    fromDisplayName:
      textValue(
        json.fromDisplayName,
      ),
    toDisplayName:
      textValue(
        json.toDisplayName,
      ),
  };
}

export function tokenBlueprintPatchVMFromMap(
  raw: unknown,
): TokenBlueprintPatchVM | null {
  if (!isRecord(raw)) {
    return null;
  }

  const tokenIcon =
    textValue(raw.tokenIcon) ||
    textValue(raw.iconUrl) ||
    textValue(raw.icon);

  const viewModel: TokenBlueprintPatchVM = {
    id:
      textValue(raw.id),
    tokenName:
      textValue(raw.tokenName) ||
      textValue(raw.name),
    symbol:
      textValue(raw.symbol),
    brandName:
      textValue(raw.brandName),
    companyName:
      textValue(raw.companyName),
    description:
      textValue(raw.description),
    tokenIcon,
  };

  return tokenBlueprintPatchHasAnyField(
    viewModel,
  )
    ? viewModel
    : null;
}

export function catalogReviewFromJson(
  raw: unknown,
): CatalogReview | null {
  if (!isRecord(raw)) {
    return null;
  }

  return {
    id:
      textValue(raw.id),
    productBlueprintId:
      textValue(
        raw.productBlueprintId,
      ),
    avatarId:
      textValue(raw.avatarId),
    avatarName:
      textValue(raw.avatarName),
    avatarIcon:
      textValue(raw.avatarIcon),
    rating:
      intValue(raw.rating),
    title:
      textValue(raw.title),
    body:
      textValue(raw.body),
    helpfulVotes:
      intValue(
        raw.helpfulVotes,
      ),
    totalVotes:
      intValue(raw.totalVotes),
    reviewedAt:
      textValue(
        raw.reviewedAt ||
          raw.createdAt,
      ),
  };
}

export function catalogReviewPageFromJson(
  raw: unknown,
  fallbackPage: number,
  fallbackPerPage: number,
): CatalogReviewPage {
  const root = unwrapData(raw);

  const rawItems = Array.isArray(
    root.items,
  )
    ? root.items
    : Array.isArray(root.reviews)
      ? root.reviews
      : [];

  const items = rawItems
    .map(catalogReviewFromJson)
    .filter(
      (
        value,
      ): value is CatalogReview =>
        Boolean(value),
    );

  return {
    items,
    page:
      intValue(root.page) ||
      fallbackPage,
    perPage:
      intValue(root.perPage) ||
      fallbackPerPage,
    total:
      intValue(root.total),
    hasNext:
      boolValue(root.hasNext),
  };
}

export function tokenContentFileFromJson(
  raw: unknown,
): TokenContentFile | null {
  if (!isRecord(raw)) {
    return null;
  }

  return {
    id:
      textValue(raw.id),
    name:
      textValue(
        raw.name ||
          raw.fileName,
      ),
    viewUri:
      textValue(
        raw.viewUri ||
          raw.url,
      ),
    contentType:
      textValue(
        raw.contentType,
      ),
    isPreviewable:
      boolValue(
        raw.isPreviewable,
      ),
  };
}

export function walletResolvedTokenResponseFromJson(
  raw: unknown,
): WalletResolvedTokenResponse {
  const root = unwrapData(raw);

  const rawFiles = Array.isArray(
    root.tokenContentsFiles,
  )
    ? root.tokenContentsFiles
    : Array.isArray(root.files)
      ? root.files
      : [];

  return {
    productId:
      textValue(root.productId),
    brandId:
      textValue(root.brandId),
    brandName:
      textValue(root.brandName),
    productBlueprintId:
      textValue(
        root.productBlueprintId,
      ),
    productName:
      textValue(root.productName),
    metadataUri:
      textValue(root.metadataUri),
    mintAddress:
      textValue(root.mintAddress),
    tokenBlueprintId:
      textValue(
        root.tokenBlueprintId,
      ),
    tokenContentsFiles: rawFiles
      .map(
        tokenContentFileFromJson,
      )
      .filter(
        (
          value,
        ): value is TokenContentFile =>
          Boolean(value),
      ),
  };
}

export function walletDTOFromJson(
  raw: unknown,
): WalletDTO {
  const root = unwrapData(raw);

  const walletsRaw = Array.isArray(
    root.wallets,
  )
    ? root.wallets
    : [];

  const firstWallet =
    walletsRaw.find(isRecord);

  const tokens =
    firstWallet &&
    Array.isArray(
      firstWallet.Tokens,
    )
      ? firstWallet.Tokens
          .map(textValue)
          .filter(Boolean)
      : firstWallet &&
          Array.isArray(
            firstWallet.tokens,
          )
        ? firstWallet.tokens
            .map(textValue)
            .filter(Boolean)
        : [];

  return {
    tokens,
  };
}