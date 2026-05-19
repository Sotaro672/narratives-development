// frontend/console/mintRequest/src/application/usecase/getMintRequestDetail.ts

import type { MintRequestRepository } from "../port/MintRequestRepository";
import { asNonEmptyString } from "../util/primitive";

function toDateStringOrNull(value: unknown): string | null {
  const v = asNonEmptyString(value);
  return v ? v : null;
}

function normalizeManagementRow(raw: unknown): any | null {
  if (Array.isArray(raw)) {
    return raw.length > 0 ? raw[0] : null;
  }

  if (raw && typeof raw === "object") {
    return raw;
  }

  return null;
}

function normalizeDetail(raw: unknown): any | null {
  if (raw && typeof raw === "object") {
    return raw;
  }

  return null;
}

function buildInspectionBatchFromManagementRow(row: any) {
  const productionId =
    asNonEmptyString(row?.productionId) || asNonEmptyString(row?.id);

  if (!productionId) return null;

  return {
    productionId,
    status: asNonEmptyString(row?.inspectionStatus) || "notYet",
    totalPassed: Number(row?.mintQuantity ?? 0),
    totalQuantity: Number(row?.productionQuantity ?? 0),
    inspections: [],
  };
}

function buildInspectionBatchFromDetail(detail: any, row: any) {
  const inspection = detail?.inspection ?? null;
  if (!inspection) return buildInspectionBatchFromManagementRow(row);

  const productionId =
    asNonEmptyString(inspection?.productionId) ||
    asNonEmptyString(detail?.productionId) ||
    asNonEmptyString(row?.productionId) ||
    asNonEmptyString(row?.id);

  if (!productionId) return buildInspectionBatchFromManagementRow(row);

  return {
    ...inspection,
    productionId,
    status:
      asNonEmptyString(inspection?.status) ||
      asNonEmptyString(detail?.inspectionStatus) ||
      asNonEmptyString(row?.inspectionStatus) ||
      "notYet",
    totalPassed: Number(
      inspection?.totalPassed ??
        detail?.mintQuantity ??
        row?.mintQuantity ??
        0,
    ),
    totalQuantity: Number(
      inspection?.totalQuantity ??
        inspection?.quantity ??
        detail?.productionQuantity ??
        row?.productionQuantity ??
        0,
    ),
    productName:
      asNonEmptyString(detail?.productName) ||
      asNonEmptyString(row?.productName) ||
      asNonEmptyString(inspection?.productName),
    modelMeta: detail?.modelMeta ?? {},
  };
}

function buildMintDTOFromManagementRow(row: any) {
  const minted = Boolean(row?.minted === true || row?.mint === true);
  if (!minted) return null;

  const productionId =
    asNonEmptyString(row?.productionId) || asNonEmptyString(row?.id);

  const tokenBlueprintId = asNonEmptyString(row?.tokenBlueprintId);
  const tokenName = asNonEmptyString(row?.tokenName);

  return {
    id: productionId,
    inspectionId: productionId,

    brandId: asNonEmptyString(row?.brandId),

    tokenBlueprintId,
    tokenName: tokenName || tokenBlueprintId,

    products: [],

    createdBy: asNonEmptyString(row?.requestedBy),
    createdByName: asNonEmptyString(row?.requestedByName),
    requestedByName: asNonEmptyString(row?.requestedByName),

    createdAt: toDateStringOrNull(row?.createdAt),

    minted,
    mintedAt: toDateStringOrNull(row?.mintedAt),
    scheduledBurnDate: toDateStringOrNull(row?.scheduledBurnDate),

    onChainTxSignature: asNonEmptyString(row?.onChainTxSignature),
  };
}

function buildMintDTOFromDetail(detail: any, row: any) {
  const detailMint = detail?.mint ?? null;
  const rowMint = buildMintDTOFromManagementRow(row);

  if (!detailMint) return rowMint;

  const productionId =
    asNonEmptyString(detailMint?.inspectionId) ||
    asNonEmptyString(detail?.productionId) ||
    asNonEmptyString(row?.productionId) ||
    asNonEmptyString(row?.id);

  const tokenBlueprintId =
    asNonEmptyString(detailMint?.tokenBlueprintId) ||
    asNonEmptyString(detail?.tokenBlueprintId) ||
    asNonEmptyString(row?.tokenBlueprintId);

  const tokenName =
    asNonEmptyString(detailMint?.tokenName) ||
    asNonEmptyString(detail?.tokenName) ||
    asNonEmptyString(row?.tokenName) ||
    tokenBlueprintId;

  const minted = Boolean(
    detailMint?.minted === true ||
      row?.minted === true ||
      row?.mint === true,
  );

  if (!minted) return null;

  return {
    id: asNonEmptyString(detailMint?.id) || productionId,
    inspectionId: productionId,

    brandId:
      asNonEmptyString(detailMint?.brandId) || asNonEmptyString(row?.brandId),

    tokenBlueprintId,
    tokenName,

    products: Array.isArray(detailMint?.productIds)
      ? detailMint.productIds
      : Array.isArray(detailMint?.products)
        ? detailMint.products
        : [],

    createdBy: asNonEmptyString(detailMint?.createdBy),
    createdByName:
      asNonEmptyString(detailMint?.createdByName) ||
      asNonEmptyString(detail?.createdByName) ||
      asNonEmptyString(row?.requestedByName),
    requestedByName:
      asNonEmptyString(detail?.requestedByName) ||
      asNonEmptyString(row?.requestedByName),

    createdAt: toDateStringOrNull(detailMint?.createdAt ?? row?.createdAt),

    minted,
    mintedAt: toDateStringOrNull(detailMint?.mintedAt ?? detail?.mintedAt ?? row?.mintedAt),
    scheduledBurnDate: toDateStringOrNull(detailMint?.scheduledBurnDate ?? row?.scheduledBurnDate),

    onChainTxSignature:
      asNonEmptyString(detailMint?.onChainTxSignature) ||
      asNonEmptyString(row?.onChainTxSignature),
  };
}

function buildFallbackProductBlueprintPatchFromManagementRow(row: any) {
  const productName = asNonEmptyString(row?.productName);
  const brandId = asNonEmptyString(row?.brandId);
  const brandName = asNonEmptyString(row?.brandName);
  const companyId = asNonEmptyString(row?.companyId);

  if (!productName && !brandId && !brandName && !companyId) {
    return null;
  }

  return {
    productName: productName || null,
    description: null,

    brandId: brandId || null,
    brandName: brandName || null,
    companyId: companyId || null,

    productBlueprintCategory: null,
    categoryFields: null,

    productIdTag: null,
    assigneeId: null,
    modelRefs: null,
  };
}

function buildFallbackTokenBlueprintPatchFromManagementRow(row: any) {
  const tokenBlueprintId = asNonEmptyString(row?.tokenBlueprintId);
  const tokenName = asNonEmptyString(row?.tokenName);
  const symbol = asNonEmptyString(row?.symbol);

  if (!tokenBlueprintId && !tokenName && !symbol) {
    return null;
  }

  return {
    id: tokenBlueprintId,
    tokenName: tokenName || "",
    name: tokenName || "",
    symbol: symbol || "",
    brandId: asNonEmptyString(row?.brandId),
    brandName: asNonEmptyString(row?.brandName),
    companyId: asNonEmptyString(row?.companyId),
    description: asNonEmptyString(row?.description),
    minted: Boolean(row?.minted === true || row?.mint === true),
    metadataUri: asNonEmptyString(row?.metadataUri),
    iconUrl: asNonEmptyString(row?.iconUrl),
  };
}

async function resolveManagementRow(
  repo: MintRequestRepository,
  productionId: string,
): Promise<any | null> {
  const fetcher = (repo as any).fetchMintRequestManagementRowsByProductionIds;

  if (typeof fetcher !== "function") {
    console.error(
      "[getMintRequestDetail] fetchMintRequestManagementRowsByProductionIds is not implemented",
    );
    return null;
  }

  const raw = await fetcher.call(repo, [productionId]).catch((e: unknown) => {
    console.error("[getMintRequestDetail] fetch management row failed", {
      productionId,
      error: e,
    });

    return null;
  });

  return normalizeManagementRow(raw);
}

async function resolveInspectionDetail(
  repo: MintRequestRepository,
  productionId: string,
): Promise<any | null> {
  const detail = await repo.fetchInspectionByProductionId(productionId).catch(
    (e: unknown) => {
      console.error("[getMintRequestDetail] fetch inspection detail failed", {
        productionId,
        error: e,
      });

      return null;
    },
  );

  return normalizeDetail(detail);
}

async function resolveProductBlueprintPatch(
  repo: MintRequestRepository,
  productBlueprintId: string,
): Promise<unknown | null> {
  const id = asNonEmptyString(productBlueprintId);
  if (!id) return null;

  return await repo.fetchProductBlueprintPatch(id).catch((e: unknown) => {
    console.error("[getMintRequestDetail] fetchProductBlueprintPatch failed", {
      productBlueprintId: id,
      error: e,
    });

    return null;
  });
}

async function resolveTokenBlueprintPatch(
  repo: MintRequestRepository,
  tokenBlueprintId: string,
): Promise<unknown | null> {
  const id = asNonEmptyString(tokenBlueprintId);
  if (!id) return null;

  return await repo.fetchTokenBlueprintPatch(id).catch((e: unknown) => {
    console.error("[getMintRequestDetail] fetchTokenBlueprintPatch failed", {
      tokenBlueprintId: id,
      error: e,
    });

    return null;
  });
}

export async function getMintRequestDetail(
  repo: MintRequestRepository,
  productionId: string,
) {
  const pid = String(productionId ?? "").trim();

  if (!pid) {
    return {
      inspectionBatch: null,
      mintDTO: null,
      productBlueprintId: "",
      productBlueprintPatch: null,
      tokenBlueprintPatch: null,
      managementRow: null,
      inspectionDetail: null,
    };
  }

  /**
   * Backend 正:
   *
   * /mint/requests?productionIds={productionId}&view=management
   *   - productBlueprintId / tokenBlueprintId / productName / tokenName の正
   *
   * /mint/inspections/{productionId}
   *   - inspection / mint / modelMeta の正
   *   - 現状 backend DTO に productBlueprintId は無いので期待しない
   */
  const [row, inspectionDetail] = await Promise.all([
    resolveManagementRow(repo, pid),
    resolveInspectionDetail(repo, pid),
  ]);

  if (!row && !inspectionDetail) {
    return {
      inspectionBatch: null,
      mintDTO: null,
      productBlueprintId: "",
      productBlueprintPatch: null,
      tokenBlueprintPatch: null,
      managementRow: null,
      inspectionDetail: null,
    };
  }

  const source = row ?? inspectionDetail ?? {};

  const productBlueprintId = asNonEmptyString(row?.productBlueprintId);

  const tokenBlueprintId =
    asNonEmptyString(row?.tokenBlueprintId) ||
    asNonEmptyString(inspectionDetail?.tokenBlueprintId) ||
    asNonEmptyString(inspectionDetail?.mint?.tokenBlueprintId);

  const [resolvedProductBlueprintPatch, resolvedTokenBlueprintPatch] =
    await Promise.all([
      resolveProductBlueprintPatch(repo, productBlueprintId),
      resolveTokenBlueprintPatch(repo, tokenBlueprintId),
    ]);

  const fallbackProductBlueprintPatch =
    buildFallbackProductBlueprintPatchFromManagementRow(source);

  const fallbackTokenBlueprintPatch =
    buildFallbackTokenBlueprintPatchFromManagementRow(source);

  const productBlueprintPatch =
    resolvedProductBlueprintPatch ?? fallbackProductBlueprintPatch;

  const tokenBlueprintPatch =
    resolvedTokenBlueprintPatch ?? fallbackTokenBlueprintPatch;

  const inspectionBatch = buildInspectionBatchFromDetail(inspectionDetail, row);

  const mintDTO = buildMintDTOFromDetail(inspectionDetail, row);

  console.log("[getMintRequestDetail] resolved detail", {
    productionId: pid,
    productBlueprintId,
    tokenBlueprintId,
    managementRow: row,
    inspectionDetail,
    resolvedProductBlueprintPatch,
    fallbackProductBlueprintPatch,
    productBlueprintPatch,
    resolvedTokenBlueprintPatch,
    fallbackTokenBlueprintPatch,
    tokenBlueprintPatch,
    inspectionBatch,
    mintDTO,
  });

  return {
    inspectionBatch,
    mintDTO,
    productBlueprintId,
    productBlueprintPatch,
    tokenBlueprintPatch,

    /**
     * 念のため残す。
     * applyDetail 側や debugging で必要になった場合に使える。
     */
    managementRow: row,
    inspectionDetail,
  };
}