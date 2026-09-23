// frontend/console/shell/src/features/mint/presentation/components/inspectionResultCard.tsx

import * as React from "react";

import {
  Card,
  CardContent,
  CardHeader,
  CardHeaderLeft,
  CardTitle,
} from "../../../../shared/ui/card";
import { ColorValue } from "../../../../shared/ui/color";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../../../../shared/ui/table";
import { rgbIntToHex } from "../../../../shared/util/color";
import type { InspectionResultCardData } from "../../application/mapper/buildInspectionResultCardData";

type InspectionResultCardProps = {
  data: InspectionResultCardData;
  className?: string;
};

const InspectionResultCard: React.FC<InspectionResultCardProps> = ({
  data,
  className,
}) => {
  const {
    title,
    rows,
    totalPassed,
    totalQuantity,
    showVolumeColumn,
  } = data;

  const emptyColSpan = showVolumeColumn ? 4 : 5;
  const totalLabelColSpan = showVolumeColumn ? 2 : 3;

  return (
    <Card className={className}>
      <CardHeader>
        <CardHeaderLeft>
          <CardTitle strong>{title || "モデル別検査結果"}</CardTitle>
        </CardHeaderLeft>
      </CardHeader>

      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>型番</TableHead>

              {showVolumeColumn ? (
                <TableHead>容量</TableHead>
              ) : (
                <>
                  <TableHead>サイズ</TableHead>
                  <TableHead>カラー</TableHead>
                </>
              )}

              <TableHead>合格数</TableHead>
              <TableHead>生産数</TableHead>
            </TableRow>
          </TableHeader>

          <TableBody>
            {rows.map((row, index) => {
              const rgbHex = rgbIntToHex(row.rgb ?? null);
              const backgroundColor = rgbHex ?? "#ffffff";

              return (
                <TableRow key={`${row.modelNumber}-${index}`}>
                  <TableCell>{row.modelNumber || "-"}</TableCell>

                  {showVolumeColumn ? (
                    <TableCell>{row.volumeLabel || "-"}</TableCell>
                  ) : (
                    <>
                      <TableCell>{row.size || "-"}</TableCell>
                      <TableCell>
                        <ColorValue
                          color={backgroundColor}
                          swatchTitle={rgbHex ?? undefined}
                        >
                          {row.color || "-"}
                        </ColorValue>
                      </TableCell>
                    </>
                  )}

                  <TableCell>{row.passedQuantity}</TableCell>
                  <TableCell>{row.quantity}</TableCell>
                </TableRow>
              );
            })}

            {rows.length === 0 && (
              <TableRow>
                <TableCell colSpan={emptyColSpan}>
                  表示できる検査結果データがありません。
                </TableCell>
              </TableRow>
            )}

            {rows.length > 0 && (
              <TableRow>
                <TableCell colSpan={totalLabelColSpan}>合計</TableCell>
                <TableCell>
                  <strong>{totalPassed}</strong>
                </TableCell>
                <TableCell>
                  <strong>{totalQuantity}</strong>
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
};

export default InspectionResultCard;