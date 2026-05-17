// frontend/amol/src/features/scan-result/application/scanTransferViewModelFactory.ts
import type { MallPreviewTransferInfo } from "../types";

export type ScanTransferHistoryViewModel = {
  transfers: MallPreviewTransferInfo[];
  hasTransfers: boolean;
};

export function createScanTransferHistoryViewModel(input: {
  backendTransfers?: MallPreviewTransferInfo[];
  chainTransfers?: MallPreviewTransferInfo[];
}): ScanTransferHistoryViewModel {
  const backendTransfers = input.backendTransfers ?? [];
  const chainTransfers = input.chainTransfers ?? [];
  const transfers = backendTransfers.length > 0 ? backendTransfers : chainTransfers;

  return {
    transfers,
    hasTransfers: transfers.length > 0,
  };
}