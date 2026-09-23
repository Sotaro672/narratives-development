// frontend/console/shell/src/features/mint/presentation/components/mintInspectionCompleteCard.tsx

import { CheckCircle2 } from "lucide-react";

import { Button } from "../../../../shared/ui/button";
import {
  Card,
  CardContent,
} from "../../../../shared/ui/card";
import Stack from "../../../../shared/ui/stack";
import Text from "../../../../shared/ui/text";

export type MintInspectionCompleteCardProps = {
  completing: boolean;
  disabled?: boolean;
  onComplete: () => void;
};

export default function MintInspectionCompleteCard({
  completing,
  disabled = false,
  onComplete,
}: MintInspectionCompleteCardProps) {
  return (
    <Card largeRadius>
      <CardContent>
        <Stack gap="md">
          <Stack gap="xs">
            <Text as="div" weight="medium">
              検品完了
            </Text>

            <Text
              as="p"
              size="xs"
              tone="muted"
            >
              除外対象がない場合でも、ここで検品完了を確定できます。
              完了後、未入力の検品結果は合格として扱われます。
            </Text>
          </Stack>

          <Button
            type="button"
            onClick={onComplete}
            disabled={disabled || completing}
          >
            <CheckCircle2 size={16} />
            {completing ? "検品完了中..." : "検品を完了する"}
          </Button>
        </Stack>
      </CardContent>
    </Card>
  );
}