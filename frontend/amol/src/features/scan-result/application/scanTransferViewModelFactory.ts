// frontend/amol/src/features/scan-result/application/scanTransferViewModelFactory.ts

import type {
  MallScanTransferResponse,
  MallTokenInfo,
  TokenBlueprintPatchVM,
} from "../../shared/types/scanResult";

export type ScanTransferSuccessModalViewModel = {
  productId: string;
  productName: string;
  assetId: string;
  metadataUri: string;
  tokenBlueprintId: string;
  tokenName: string;
  tokenIconUrl: string;
  brandId: string;
  brandName: string;
  canOpenContents: boolean;
  fromName: string;
  toName: string;
  walletUpdated: boolean;
};

export function createScanTransferSuccessModalViewModel(input: {
  result: MallScanTransferResponse | null;
  token: MallTokenInfo | null;
  tokenBlueprintPatch: TokenBlueprintPatchVM | null;
  productName: string;
}): ScanTransferSuccessModalViewModel | null {
  const result = input.result;

  if (!result || !result.matched) {
    return null;
  }

  const token = input.token;
  const tokenBlueprintPatch = input.tokenBlueprintPatch;
  const metadataUri = token?.metadataUri ?? "";

  return {
    productId: result.productId,
    productName: input.productName,
    assetId: result.assetId,
    metadataUri,
    tokenBlueprintId: token?.tokenBlueprintId ?? "",
    tokenName: tokenBlueprintPatch?.tokenName ?? "",
    tokenIconUrl: tokenBlueprintPatch?.tokenIcon ?? "",
    brandId: token?.brandId ?? "",
    brandName: tokenBlueprintPatch?.brandName ?? "",
    canOpenContents: Boolean(result.assetId && metadataUri),
    fromName: result.fromDisplayName,
    toName: result.toDisplayName,
    walletUpdated: result.updatedToAddress,
  };
}