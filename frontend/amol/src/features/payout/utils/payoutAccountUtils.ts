// frontend/amol/src/features/payout/utils/payoutAccountUtils.ts

import type {
  PayoutAccount,
} from "../../shared/types/payoutAccount";

export function getPayoutAccountStatusLabel(
  payoutAccount: PayoutAccount | null,
  isLoading: boolean
): string {
  if (isLoading) {
    return "確認中...";
  }

  if (!payoutAccount) {
    return "未登録";
  }

  if (payoutAccount.payoutsEnabled) {
    return "登録済み";
  }

  if (payoutAccount.detailsSubmitted) {
    return "確認中";
  }

  return "登録未完了";
}

export function getPayoutAccountActionLabel(
  payoutAccount: PayoutAccount | null,
  isOpeningStripe: boolean
): string {
  if (isOpeningStripe) {
    return "Stripeへ接続中...";
  }

  if (payoutAccount) {
    return "口座情報を変更";
  }

  return "売上受取口座を登録";
}