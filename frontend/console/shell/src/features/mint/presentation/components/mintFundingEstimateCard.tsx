// frontend/console/shell/src/features/mint/presentation/components/mintFundingEstimateCard.tsx

import { Coins } from "lucide-react";

import { Button } from "../../../../shared/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";
import { ErrorMessage } from "../../../../shared/ui/error";
import {
  ProgressIndeterminate,
  ProgressMessage,
  ProgressMetric,
} from "../../../../shared/ui/progress";
import Stack from "../../../../shared/ui/stack";
import Text from "../../../../shared/ui/text";

import type { MintFundingEstimate } from "../../infrastructure/dto/MintRequestRepository";

export type MintFundingEstimateCardProps = {
  selectedTokenBlueprintId: string;
  estimate: MintFundingEstimate | null;
  loading: boolean;
  error: string | null;
  canSubmit: boolean;
  onMint: () => void;
};

function formatSol(value: number): string {
  if (!Number.isFinite(value)) {
    return "0";
  }

  return value.toLocaleString("ja-JP", {
    minimumFractionDigits: 0,
    maximumFractionDigits: 9,
  });
}

export default function MintFundingEstimateCard({
  selectedTokenBlueprintId,
  estimate,
  loading,
  error,
  canSubmit,
  onMint,
}: MintFundingEstimateCardProps) {
  return (
    <Card largeRadius>
      <CardHeader>
        <CardTitle>SOL見積</CardTitle>
      </CardHeader>

      <CardContent>
        <Stack gap="md">
          {!selectedTokenBlueprintId ? (
            <Text as="div" tone="muted">
              トークン設計を選択すると、ミントに必要なSOLを見積もります。
            </Text>
          ) : loading ? (
            <Stack gap="sm" role="status" aria-live="polite">
              <Text tone="muted">
                SOL見積を取得中です…
              </Text>
              <ProgressIndeterminate ariaLabel="SOL見積を取得中" />
            </Stack>
          ) : error ? (
            <ErrorMessage>
              {error}
            </ErrorMessage>
          ) : estimate ? (
            <Stack gap="md">
              <Stack gap="sm">
                <ProgressMetric
                  label="Reserve Wallet残高"
                  value={`${formatSol(estimate.reserve.balanceSol)} SOL`}
                />
              </Stack>

              <div className="mint-section">
                <Stack gap="sm">
                  <ProgressMetric
                    label="1件あたりMint手数料"
                    value={`${formatSol(
                      estimate.estimate.mintTransactionFeePerItemSol,
                    )} SOL`}
                  />

                  <ProgressMetric
                    label="Mint手数料合計"
                    value={`${formatSol(
                      estimate.estimate.mintTransactionFeeTotalSol,
                    )} SOL`}
                  />

                  <ProgressMetric
                    label="初回作成費"
                    value={`${formatSol(
                      estimate.estimate.initialCreationCostSol,
                    )} SOL`}
                  />
                </Stack>
              </div>

              <div className="mint-section">
                <ProgressMetric
                  label="最終必要SOL合計"
                  value={`${formatSol(
                    estimate.estimate.totalRequiredSol,
                  )} SOL`}
                />
              </div>

              <ProgressMessage
                variant={
                  estimate.estimate.sufficient
                    ? "notice"
                    : "warning"
                }
              >
                {estimate.estimate.sufficient
                  ? "SOL残高はミント実行に必要な条件を満たしています。"
                  : "Reserve WalletのSOL残高が不足しています。"}
              </ProgressMessage>
            </Stack>
          ) : null}

          <div>
            <Button
              type="button"
              onClick={onMint}
              disabled={!canSubmit}
            >
              <Coins size={16} />
              ミント申請を実行
            </Button>
          </div>
        </Stack>
      </CardContent>
    </Card>
  );
}