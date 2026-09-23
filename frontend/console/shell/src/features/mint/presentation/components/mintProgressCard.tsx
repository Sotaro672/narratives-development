// frontend/console/shell/src/features/mint/presentation/components/mintProgressCard.tsx

import { Badge } from "../../../../shared/ui/badge";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";
import {
  Progress,
  ProgressMetric,
} from "../../../../shared/ui/progress";
import Stack from "../../../../shared/ui/stack";
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
    <Card role="status" aria-live="polite">
      <CardHeader>
        <CardTitle>ミント進捗</CardTitle>
      </CardHeader>

      <CardContent>
        <Stack gap="md">
          <Stack gap="sm">
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
          </Stack>

          <div className="mint-section">
            <Stack gap="sm">
              <ProgressMetric
                label="待機中"
                value={progress.pending}
              />

              <ProgressMetric
                label="ミント中"
                value={progress.minting}
              />

              <ProgressMetric
                label="完了"
                value={progress.minted}
              />

              <ProgressMetric
                label="再試行待ち"
                value={
                  progress.failedRetryable > 0 ? (
                    <Badge variant="warning">
                      {progress.failedRetryable}
                    </Badge>
                  ) : (
                    progress.failedRetryable
                  )
                }
              />

              <ProgressMetric
                label="失敗"
                value={
                  progress.failedFatal > 0 ? (
                    <Badge variant="danger">
                      {progress.failedFatal}
                    </Badge>
                  ) : (
                    progress.failedFatal
                  )
                }
              />
            </Stack>
          </div>
        </Stack>
      </CardContent>
    </Card>
  );
}