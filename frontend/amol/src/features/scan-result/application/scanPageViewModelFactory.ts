// frontend/amol/src/features/scan-result/application/scanPageViewModelFactory.ts
import type {
  MallPreviewTransferInfo,
  PreviewState,
  ScanResultPageState,
} from "../types";
import {
  createScanProductInfoViewModel,
  type ScanProductInfoViewModel,
} from "./scanProductInfoFactory";
import {
  createScanTransferHistoryViewModel,
  type ScanTransferHistoryViewModel,
} from "./scanTransferViewModelFactory";

export type ScanResultPageViewModel = {
  product: ScanProductInfoViewModel | null;
  transferHistory: ScanTransferHistoryViewModel;
};

export function createScanResultPageViewModel(input: {
  state: ScanResultPageState;
  previewState: PreviewState | null;
  chainTransfers: MallPreviewTransferInfo[];
}): ScanResultPageViewModel {
  const backendTransfers = input.previewState?.raw.transfers ?? [];

  return {
    product: createScanProductInfoViewModel(input.previewState),
    transferHistory: createScanTransferHistoryViewModel({
      backendTransfers,
      chainTransfers: input.chainTransfers,
    }),
  };
}