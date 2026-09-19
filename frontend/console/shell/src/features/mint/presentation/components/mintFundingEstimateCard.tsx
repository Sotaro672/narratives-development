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
    <Card className="mint-request-card">
      <CardHeader>
        <CardTitle>SOL見積</CardTitle>
      </CardHeader>

      <CardContent className="mint-request-card__body">
        <div className="mint-funding">
          {!selectedTokenBlueprintId ? (
            <Text as="div" tone="muted">
              トークン設計を選択すると、ミントに必要なSOLを見積もります。
            </Text>
          ) : loading ? (
            <div
              className="mint-funding__loading"
              role="status"
              aria-live="polite"
            >
              <div
                className="mint-funding__spinner"
                aria-hidden="true"
              />
              <Text tone="muted">
                SOL見積を取得中です…
              </Text>
            </div>
          ) : error ? (
            <ErrorMessage>
              {error}
            </ErrorMessage>
          ) : estimate ? (
            <div className="mint-funding__estimate">
              <div className="mint-funding__rows">
                <div className="mint-funding__row">
                  <Text tone="muted">
                    Reserve Wallet残高
                  </Text>
                  <Text weight="semibold">
                    {formatSol(estimate.reserve.balanceSol)} SOL
                  </Text>
                </div>
              </div>

              <div className="mint-funding__section">
                <div className="mint-funding__rows">
                  <div className="mint-funding__row">
                    <Text tone="muted">
                      1件あたりMint手数料
                    </Text>
                    <Text weight="semibold">
                      {formatSol(
                        estimate.estimate.mintTransactionFeePerItemSol,
                      )}{" "}
                      SOL
                    </Text>
                  </div>

                  <div className="mint-funding__row">
                    <Text tone="muted">
                      Mint手数料合計
                    </Text>
                    <Text weight="semibold">
                      {formatSol(
                        estimate.estimate.mintTransactionFeeTotalSol,
                      )}{" "}
                      SOL
                    </Text>
                  </div>

                  <div className="mint-funding__row">
                    <Text tone="muted">
                      初回作成費
                    </Text>
                    <Text weight="semibold">
                      {formatSol(
                        estimate.estimate.initialCreationCostSol,
                      )}{" "}
                      SOL
                    </Text>
                  </div>
                </div>
              </div>

              <div className="mint-funding__section">
                <div className="mint-funding__row">
                  <Text weight="semibold">
                    最終必要SOL合計
                  </Text>
                  <Text weight="semibold">
                    {formatSol(
                      estimate.estimate.totalRequiredSol,
                    )}{" "}
                    SOL
                  </Text>
                </div>
              </div>

              <Text
                as="div"
                weight="medium"
                className={
                  estimate.estimate.sufficient
                    ? "mint-funding__status mint-funding__status--sufficient"
                    : "mint-funding__status mint-funding__status--insufficient"
                }
              >
                {estimate.estimate.sufficient
                  ? "SOL残高はミント実行に必要な条件を満たしています。"
                  : "Reserve WalletのSOL残高が不足しています。"}
              </Text>
            </div>
          ) : null}

          <div className="mint-request-card__actions">
            <Button
              type="button"
              onClick={onMint}
              disabled={!canSubmit}
            >
              <Coins size={16} />
              ミント申請を実行
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}