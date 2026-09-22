// frontend/console/shell/src/features/mint/presentation/components/mintInfoCard.tsx

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";
import Text from "../../../../shared/ui/text";

import type { MintRequestManagementRowDTO } from "../../infrastructure/dto/mintRequestManagementRow";
import { mintStatusLabel } from "../formatter/mintStatusLabel";

export type MintInfoCardProps = {
  mintRequestRow: MintRequestManagementRowDTO;
  mintedAtLabel: string;
};

export default function MintInfoCard({
  mintRequestRow,
  mintedAtLabel,
}: MintInfoCardProps) {
  const statusLabel = mintStatusLabel(
    mintRequestRow.mintStatus,
  );

  return (
    <Card>
      <CardHeader>
        <CardTitle>ミント情報</CardTitle>
      </CardHeader>

      <CardContent>
        <div className="mint-info">
          <Text as="div">
            生産数:{" "}
            <Text weight="semibold">
              {mintRequestRow.productionQuantity ?? 0}
            </Text>
          </Text>

          <Text as="div">
            ミント数:{" "}
            <Text weight="semibold">
              {mintRequestRow.mintQuantity ?? 0}
            </Text>
          </Text>

          <Text as="div">
            ミント状態:{" "}
            <Text weight="semibold">
              {statusLabel}
            </Text>
          </Text>

          <Text as="div">
            リクエスト者:{" "}
            {mintRequestRow.requestedByName ||
              mintRequestRow.requestedBy ||
              "（不明）"}
          </Text>

          <Text as="div">
            ミント日時: {mintedAtLabel}
          </Text>
        </div>
      </CardContent>
    </Card>
  );
}