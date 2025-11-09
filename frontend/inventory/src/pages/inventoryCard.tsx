import * as React from "react";
import { Palette } from "lucide-react";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../shared/ui/card";
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from "../../../shared/ui/table";

import "./inventoryCard.css";

export type InventoryRow = {
  /** 型番 (例: "LM-SB-S-WHT") */
  modelCode: string;
  /** サイズ (例: "S" | "M" | "L") */
  size: string;
  /** カラー表示名 (例: "ホワイト") */
  colorName: string;
  /** カラーコード (例: "#000000") - 無指定の場合は白い円 */
  colorCode?: string;
  /** 在庫数 */
  stock: number;
};

type InventoryCardProps = {
  title?: string;
  rows: InventoryRow[];
  className?: string;
};

/**
 * モデル別在庫一覧カード（閲覧専用）
 * Screenshot のトーンに合わせた表示専用テーブル
 */
const InventoryCard: React.FC<InventoryCardProps> = ({
  title = "モデル別在庫一覧",
  rows,
  className,
}) => {
  // 在庫合計数を算出
  const totalStock = React.useMemo(
    () => rows.reduce((sum, r) => sum + (r.stock || 0), 0),
    [rows]
  );

  return (
    <Card className={`ivc ${className ?? ""}`}>
      <CardHeader className="ivc__header">
        <div className="ivc__header-inner">
          <Palette className="ivc__icon" size={18} />
          <CardTitle className="ivc__title">{title}</CardTitle>
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
                <TableHead className="ivc__th ivc__th--right">在庫数</TableHead>
              </TableRow>
            </TableHeader>

            <TableBody>
              {rows.map((row, idx) => (
                <TableRow key={`${row.modelCode}-${idx}`} className="ivc__tr">
                  <TableCell className="ivc__model">{row.modelCode}</TableCell>
                  <TableCell className="ivc__size">{row.size}</TableCell>
                  <TableCell className="ivc__color-cell">
                    <span
                      className="ivc__color-dot"
                      style={{
                        backgroundColor:
                          row.colorCode && row.colorCode.trim()
                            ? row.colorCode
                            : "#ffffff",
                        boxShadow: "0 0 0 1px rgba(0,0,0,0.18)",
                      }}
                    />
                    <span className="ivc__color-label">{row.colorName}</span>
                  </TableCell>
                  <TableCell className="ivc__stock">
                    <span className="ivc__stock-number">{row.stock}</span>
                  </TableCell>
                </TableRow>
              ))}

              {/* データなし表示 */}
              {rows.length === 0 && (
                <TableRow>
                  <TableCell colSpan={4} className="ivc__empty">
                    表示できる在庫データがありません。
                  </TableCell>
                </TableRow>
              )}

              {/* ✅ 合計行 */}
              {rows.length > 0 && (
                <TableRow className="ivc__total-row">
                  <TableCell
                    colSpan={3}
                    className="ivc__total-label ivc__th--right"
                  >
                    合計
                  </TableCell>
                  <TableCell className="ivc__total-value">
                    <strong>{totalStock}</strong>
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

export default InventoryCard;
