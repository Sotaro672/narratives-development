// frontend/console/mintRequest/src/presentation/components/inspectionResultCard.tsx

import * as React from "react";
import { Palette } from "lucide-react";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shell/src/shared/ui/card";
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from "../../../../shell/src/shared/ui/table";

import type {
  InspectionResultRow,
  UseInspectionResultCardResult,
} from "../hook/useInspectionResultCard";

type InspectionResultCardProps = {
  /** useInspectionResultCard の戻り値をそのまま受け取る想定 */
  data: UseInspectionResultCardResult;
  className?: string;
};

const InspectionResultCard: React.FC<InspectionResultCardProps> = ({
  data,
  className,
}) => {
  const { title, rows, totalPassed, totalQuantity, rgbIntToHex } = data;

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
                <TableHead className="ivc__th">サイズ</TableHead>
                <TableHead className="ivc__th">カラー</TableHead>
                {/* 合格数（生産数の左隣） */}
                <TableHead className="ivc__th ivc__th--right">
                  合格数
                </TableHead>
                <TableHead className="ivc__th ivc__th--right">
                  生産数
                </TableHead>
              </TableRow>
            </TableHeader>

            <TableBody>
              {rows.map((row, idx) => {
                const rgbHex = rgbIntToHex(row.rgb ?? null);
                const bgColor = rgbHex ?? "#ffffff";

                return (
                  <TableRow
                    key={`${row.modelNumber}-${idx}`}
                    className="ivc__tr"
                  >
                    {/* 型番 */}
                    <TableCell className="ivc__model">
                      {row.modelNumber}
                    </TableCell>

                    {/* サイズ */}
                    <TableCell className="ivc__size">
                      {row.size || "-"}
                    </TableCell>

                    {/* カラー */}
                    <TableCell className="ivc__color-cell">
                      <span
                        className="ivc__color-dot"
                        style={{
                          backgroundColor: bgColor,
                          boxShadow: "0 0 0 1px rgba(0,0,0,0.18)",
                        }}
                        title={rgbHex ?? ""}
                      />
                      <span className="ivc__color-label">
                        {row.color || "-"}
                      </span>
                    </TableCell>

                    {/* 合格数 */}
                    <TableCell className="ivc__quantity">
                      <span className="ivc__quantity-number">
                        {row.passedQuantity}
                      </span>
                    </TableCell>

                    {/* 生産数 */}
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
                  <TableCell colSpan={5} className="ivc__empty">
                    表示できる検査結果データがありません。
                  </TableCell>
                </TableRow>
              )}

              {rows.length > 0 && (
                <TableRow className="ivc__total-row">
                  {/* 「合計」ラベルを 3 列分にまたがせる */}
                  <TableCell
                    colSpan={3}
                    className="ivc__total-label ivc__th--right"
                  >
                    合計
                  </TableCell>
                  {/* 合格数合計 */}
                  <TableCell className="ivc__total-value">
                    <strong>{totalPassed}</strong>
                  </TableCell>
                  {/* 生産数合計 */}
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
