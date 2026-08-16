// frontend/console/shell/src/features/mint/presentation/components/inspectionResultCard.tsx

import * as React from "react";
import { Palette } from "lucide-react";
import { Card, CardHeader, CardTitle, CardContent } from "../../../../shared/ui/card";
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
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
    <Card className={`ivc ${className ?? ""}`}>
      <CardHeader className="ivc__header">
        <div className="ivc__header-inner">
          <Palette className="ivc__icon" size={18} />
          <CardTitle className="ivc__title">
            {title || "モデル別検査結果"}
          </CardTitle>
        </div>
      </CardHeader>

      <CardContent className="ivc__body">
        <div className="ivc__table-wrap">
          <Table className="ivc__table">
            <TableHeader>
              <TableRow>
                <TableHead className="ivc__th ivc__th--left">型番</TableHead>

                {showVolumeColumn ? (
                  <TableHead className="ivc__th">容量</TableHead>
                ) : (
                  <>
                    <TableHead className="ivc__th">サイズ</TableHead>
                    <TableHead className="ivc__th">カラー</TableHead>
                  </>
                )}

                <TableHead className="ivc__th ivc__th--right">合格数</TableHead>
                <TableHead className="ivc__th ivc__th--right">生産数</TableHead>
              </TableRow>
            </TableHeader>

            <TableBody>
              {rows.map((row, index) => {
                const rgbHex = rgbIntToHex(row.rgb ?? null);
                const backgroundColor = rgbHex ?? "#ffffff";

                return (
                  <TableRow
                    key={`${row.modelNumber}-${index}`}
                    className="ivc__tr"
                  >
                    <TableCell className="ivc__model">
                      {row.modelNumber || "-"}
                    </TableCell>

                    {showVolumeColumn ? (
                      <TableCell className="ivc__size">
                        {row.volumeLabel || "-"}
                      </TableCell>
                    ) : (
                      <>
                        <TableCell className="ivc__size">
                          {row.size || "-"}
                        </TableCell>

                        <TableCell className="ivc__color-cell">
                          <span
                            className="ivc__color-dot"
                            style={{
                              backgroundColor,
                              boxShadow: "0 0 0 1px rgba(0,0,0,0.18)",
                            }}
                            title={rgbHex ?? ""}
                          />
                          <span className="ivc__color-label">
                            {row.color || "-"}
                          </span>
                        </TableCell>
                      </>
                    )}

                    <TableCell className="ivc__quantity">
                      <span className="ivc__quantity-number">
                        {row.passedQuantity}
                      </span>
                    </TableCell>

                    <TableCell className="ivc__quantity">
                      <span className="ivc__quantity-number">
                        {row.quantity}
                      </span>
                    </TableCell>
                  </TableRow>
                );
              })}

              {rows.length === 0 && (
                <TableRow>
                  <TableCell colSpan={emptyColSpan} className="ivc__empty">
                    表示できる検査結果データがありません。
                  </TableCell>
                </TableRow>
              )}

              {rows.length > 0 && (
                <TableRow className="ivc__total-row">
                  <TableCell
                    colSpan={totalLabelColSpan}
                    className="ivc__total-label ivc__th--right"
                  >
                    合計
                  </TableCell>

                  <TableCell className="ivc__total-value">
                    <strong>{totalPassed}</strong>
                  </TableCell>

                  <TableCell className="ivc__total-value">
                    <strong>{totalQuantity}</strong>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>
  );
};

export default InspectionResultCard;