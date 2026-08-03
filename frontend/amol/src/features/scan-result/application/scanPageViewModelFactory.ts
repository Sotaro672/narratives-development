// frontend/amol/src/features/scan-result/application/scanPageViewModelFactory.ts

import type {
  PreviewState,
} from "../../shared/types/scanResult";
import {
  createScanProductInfoViewModel,
  type ScanProductInfoViewModel,
} from "./scanProductInfoFactory";

export type ScanResultPageViewModel = {
  product: ScanProductInfoViewModel | null;
};

export function createScanResultPageViewModel(input: {
  previewState: PreviewState | null;
}): ScanResultPageViewModel {
  return {
    product: createScanProductInfoViewModel(
      input.previewState,
    ),
  };
}