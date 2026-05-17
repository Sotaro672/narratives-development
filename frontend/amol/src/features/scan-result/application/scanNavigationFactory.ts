// frontend/amol/src/features/scan-result/application/scanNavigationFactory.ts
import type { TokenResolveDTO } from "../types";

export function createTransferredTokenContentsPath(input: {
  mintAddress: string;
  productId?: string;
}): string {
  const params = new URLSearchParams();

  params.set("mintAddress", input.mintAddress.trim());
  params.set("from", "preview_transfer");

  const productId = input.productId?.trim() ?? "";

  if (productId) {
    params.set("productId", productId);
  }

  return `/wallet/contents?${params.toString()}`;
}

export function findFirstPreviewableTokenFile(resolved: TokenResolveDTO) {
  return resolved.tokenContentsFiles.find((item) => {
    return item.isPreviewable && item.viewUri.trim();
  });
}

export function createOwnedTokenContentsPath(input: {
  mintAddress: string;
  resolved: {
    productId?: string;
    brandId?: string;
    brandName?: string;
    productName?: string;
    productBlueprintId?: string;
    tokenBlueprintId?: string;
    metadataUri?: string;
  };
}): string {
  const params = new URLSearchParams();

  params.set("mintAddress", input.mintAddress.trim());

  const productId = input.resolved.productId?.trim() ?? "";
  const brandId = input.resolved.brandId?.trim() ?? "";
  const brandName = input.resolved.brandName?.trim() ?? "";
  const productName = input.resolved.productName?.trim() ?? "";
  const productBlueprintId = input.resolved.productBlueprintId?.trim() ?? "";
  const tokenBlueprintId = input.resolved.tokenBlueprintId?.trim() ?? "";
  const metadataUri = input.resolved.metadataUri?.trim() ?? "";

  if (productId) {
    params.set("productId", productId);
  }

  if (brandId) {
    params.set("brandId", brandId);
  }

  if (brandName) {
    params.set("brandName", brandName);
  }

  if (productName) {
    params.set("productName", productName);
  }

  if (productBlueprintId) {
    params.set("productBlueprintId", productBlueprintId);
  }

  if (tokenBlueprintId) {
    params.set("tokenBlueprintId", tokenBlueprintId);
  }

  if (metadataUri) {
    params.set("metadataUri", metadataUri);
  }

  return `/contents?${params.toString()}`;
}