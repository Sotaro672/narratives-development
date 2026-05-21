// frontend/amol/src/features/scan-result/application/scanTransferViewModelFactory.ts
import type {
  MallPreviewTransferInfo,
  MallScanTransferResponse,
  MallTokenInfo,
  TokenBlueprintPatchVM,
} from "../types";

export type ScanTransferHistoryViewModel = {
  transfers: MallPreviewTransferInfo[];
  hasTransfers: boolean;
};

export type ScanTransferSuccessModalViewModel = {
  productId: string;
  productName: string;
  mintAddress: string;

  /**
   * ContentsPage navigation query params.
   * useContentsPage reads these from URLSearchParams.
   */
  metadataUri: string;
  tokenBlueprintId: string;
  tokenName: string;
  tokenIconUrl: string;
  brandId: string;
  brandName: string;

  fromName: string;
  toName: string;
  walletUpdated: boolean;
};

function normalize(value: string | null | undefined): string {
  return value?.trim() ?? "";
}

export function createScanTransferHistoryViewModel(input: {
  backendTransfers?: MallPreviewTransferInfo[];
  chainTransfers?: MallPreviewTransferInfo[];
}): ScanTransferHistoryViewModel {
  const backendTransfers = input.backendTransfers ?? [];
  const chainTransfers = input.chainTransfers ?? [];
  const transfers =
    backendTransfers.length > 0 ? backendTransfers : chainTransfers;

  return {
    transfers,
    hasTransfers: transfers.length > 0,
  };
}

export function createScanTransferSuccessModalViewModel(input: {
  result: MallScanTransferResponse | null;
  transferredMintAddress: string;
  token: MallTokenInfo | null;
  tokenBlueprintPatch: TokenBlueprintPatchVM | null;
  productName: string;
}): ScanTransferSuccessModalViewModel | null {
  const result = input.result;

  if (!result || result.matched !== true) {
    return null;
  }

  const mintAddress =
    normalize(result.mintAddress) || normalize(input.transferredMintAddress);

  if (!mintAddress) {
    return null;
  }

  const fromName = normalize(result.fromDisplayName);
  const toName = normalize(result.toDisplayName);

  if (!fromName || !toName) {
    return null;
  }

  const token = input.token;
  const tokenBlueprintPatch = input.tokenBlueprintPatch;

  const productName = normalize(input.productName);
  const metadataUri = normalize(token?.metadataUri);
  const tokenBlueprintId = normalize(token?.tokenBlueprintId);
  const tokenName = normalize(tokenBlueprintPatch?.tokenName);
  const tokenIconUrl = normalize(tokenBlueprintPatch?.tokenIcon);
  const brandId = normalize(token?.brandId);
  const brandName = normalize(token?.brandName);

  return {
    productId: result.productId,
    productName,
    mintAddress,
    metadataUri,
    tokenBlueprintId,
    tokenName,
    tokenIconUrl,
    brandId,
    brandName,
    fromName,
    toName,
    walletUpdated: result.updatedToAddress,
  };
}