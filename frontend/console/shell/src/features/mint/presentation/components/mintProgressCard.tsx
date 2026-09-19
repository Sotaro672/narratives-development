// frontend/console/shell/src/features/mint/presentation/components/mintProgressCard.tsx

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";
import { Progress } from "../../../../shared/ui/progress";
import Text from "../../../../shared/ui/text";

import type { MintTaskProgressDTO } from "../../infrastructure/dto/mintRequestManagementRow";

export type MintProgressCardProps = {
  progress: MintTaskProgressDTO;
};

/**
 * mints/{mintId}/products の集計結果を表示する進捗カード。
 */
export default function MintProgressCard({
  progress,
}: MintProgressCardProps) {
  return (
    <Card className="pb-select" role="status" aria-live="polite">
      <CardHeader>
        <CardTitle>ミント進捗</CardTitle>
      </CardHeader>

      <CardContent>
        <div className="mint-progress">
          <div className="mint-progress__progress">
            <Progress
              value={progress.percentage}
              label="進捗"
              ariaLabel="ミント進捗"
            />

            <Text
              as="div"
              size="xs"
              tone="muted"
              className="mint-progress__summary"
            >
              <Text size="xs" weight="semibold">
                {progress.minted}
              </Text>{" "}
              / {progress.total} 完了
            </Text>
          </div>

          <div className="mint-progress__details">
            <div className="mint-progress__rows">
              <div className="mint-progress__row">
                <Text tone="muted">待機中</Text>
                <Text weight="semibold">{progress.pending}</Text>
              </div>

              <div className="mint-progress__row">
                <Text tone="muted">ミント中</Text>
                <Text weight="semibold">{progress.minting}</Text>
              </div>

              <div className="mint-progress__row">
                <Text tone="muted">完了</Text>
                <Text weight="semibold">{progress.minted}</Text>
              </div>

              <div className="mint-progress__row">
                <Text tone="muted">再試行待ち</Text>
                <Text
                  weight="semibold"
                  className={
                    progress.failedRetryable > 0
                      ? "mint-progress__value--warning"
                      : undefined
                  }
                >
                  {progress.failedRetryable}
                </Text>
              </div>

              <div className="mint-progress__row">
                <Text tone="muted">失敗</Text>
                <Text
                  weight="semibold"
                  className={
                    progress.failedFatal > 0
                      ? "mint-progress__value--danger"
                      : undefined
                  }
                >
                  {progress.failedFatal}
                </Text>
              </div>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}