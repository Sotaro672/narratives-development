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

import { Input } from "../../../../shell/src/shared/ui/input";

import "../styles/production.css";

export type ProductionQuantityRow = {
  modelCode: string;
  size: string;
  colorName: string;
  colorCode?: string; // "#000000" | undefined
  stock: number;
};

type ProductionQuantityCardProps = {
  title?: string;
  rows: ProductionQuantityRow[];
  className?: string;

  /** 追加: edit/view */
  mode?: "edit" | "view";

  /** 追加: 編集時の変更ハンドラ */
  onChangeStock?: (index: number, value: number) => void;
};

const ProductionQuantityCard: React.FC<ProductionQuantityCardProps> = ({
  title = "モデル別生産数一覧",
  rows,
  className,
  mode = "view",
  onChangeStock,
}) => {
  const totalStock = React.useMemo(
    () => rows.reduce((sum, r) => sum + (r.stock || 0), 0),
    [rows],
  );

  return (
    <Card className={`ivc ${className ?? ""}`}>
      <CardHeader className="ivc__header">
        <div className="ivc__header-inner">
          <Palette className="ivc__icon" size={18} />
          <CardTitle className="ivc__title">
            {title}
            {mode === "edit" && (
              <span className="ml-2 text-xs text-gray-500">（編集）</span>
            )}
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
                <TableHead className="ivc__th ivc__th--right">生産数</TableHead>
              </TableRow>
            </TableHeader>

            <TableBody>
              {rows.map((row, idx) => (
                <TableRow key={`${row.modelCode}-${idx}`} className="ivc__tr">
                  <TableCell className="ivc__model">
                    {row.modelCode}
                  </TableCell>
                  <TableCell className="ivc__size">{row.size}</TableCell>

                  <TableCell className="ivc__color-cell">
                    <span
                      className="ivc__color-dot"
                      style={{
                        backgroundColor:
                          row.colorCode && row.colorCode.trim()
                            ? row.colorCode
                            : "#000000", // ★ rgb:0 を黒で表示
                        boxShadow: "0 0 0 1px rgba(0,0,0,0.18)",
                      }}
                    />
                    <span className="ivc__color-label">{row.colorName}</span>
                  </TableCell>

                  <TableCell className="ivc__stock">
                    {mode === "edit" ? (
                      <Input
                        type="number"
                        min={0}
                        value={row.stock}
                        className="w-20 text-right"
                        onChange={(e) => {
                          const v = Math.max(0, parseInt(e.target.value || "0"));
                          onChangeStock?.(idx, v);
                        }}
                      />
                    ) : (
                      <span className="ivc__stock-number">{row.stock}</span>
                    )}
                  </TableCell>
                </TableRow>
              ))}

              {rows.length === 0 && (
                <TableRow>
                  <TableCell colSpan={4} className="ivc__empty">
                    表示できる型番がありません。
                  </TableCell>
                </TableRow>
              )}

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

export default ProductionQuantityCard;
