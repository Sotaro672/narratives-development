// frontend/console/shell/src/features/mint/presentation/formatter/mintStatusLabel.ts

import type { MintStatus } from "../../../../shared/types/mints";

export function mintStatusLabel(
  status: MintStatus | string | null | undefined,
): string {
  switch (status) {
    case "MINTED":
      return "ミント完了";
    case "QUEUED":
      return "ミント待機中";
    case "MINTING":
      return "ミント中";
    case "PARTIALLY_MINTED":
      return "一部ミント完了";
    case "FAILED_RETRYABLE":
      return "再試行待ち";
    case "FAILED_FATAL":
      return "ミント失敗";
    case "CREATED":
      return "作成済み";
    case null:
    case undefined:
    case "":
      return "（未設定）";
    default:
      return status;
  }
}