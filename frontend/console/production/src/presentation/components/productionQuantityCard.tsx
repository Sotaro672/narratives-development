// frontend/console/production/src/presentation/components/productionQuantityCard.tsx
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
  /** 型番 (例: "LM-SB-S-WHT") */
  modelCode: string;
  /** サイズ (例: "S" | "M" | "L") */
  size: string;
  /** カラー表示名 (例: "ホワイト") */
  colorName: string;
  /** カラーコード (例: "#000000") - 無指定の場合は白い円 */
  colorCode?: string;
  /** 生産数 */
  stock: number;
};

type ProductionQuantityCardProps = {
  title?: string;
  rows: ProductionQuantityRow[];
  className?: string;

  /** 表示モード */
  mode?: "view" | "edit";

  /** 親に編集結果を返したい場合に使用（任意） */
  onRowsChange?: (rows: ProductionQuantityRow[]) => void;
};

/**
 * モデル別生産数一覧カード
 * - view: 閲覧専用
 * - edit: 生産数を編集可能
 */
const ProductionQuantityCard: React.FC<ProductionQuantityCardProps> = ({
  title = "モデル別生産数一覧",
  rows,
  className,
  mode = "view",
  onRowsChange,
}) => {
  const isEditable = mode === "edit";

  // 生産数編集用にローカルコピーを持つ
  const [localRows, setLocalRows] = React.useState<ProductionQuantityRow[]>(rows);

  // rows が外から更新されたときはローカルも同期
  React.useEffect(() => {
    setLocalRows(rows);
  }, [rows]);

  // 生産数合計
  const totalStock = React.useMemo(
    () => localRows.reduce((sum, r) => sum + (r.stock || 0), 0),
    [localRows],
  );

  const handleChangeStock = React.useCallback(
    (index: number, value: string) => {
      if (!isEditable) return;

      // 空文字は 0、負値/NaN は 0、少数は切り捨て
      const n = Math.max(0, Math.floor(Number(value || "0")));
      const next = [...localRows];
      next[index] = {
        ...next[index],
        stock: Number.isFinite(n) ? n : 0,
      };
      setLocalRows(next);
      onRowsChange?.(next);
    },
    [isEditable, localRows, onRowsChange],
  );

  return (
    <Card className={`ivc ${className ?? ""}`}>
      <CardHeader className="ivc__header">
        <div className="ivc__header-inner">
          <Palette className="ivc__icon" size={18} />
          <CardTitle className="ivc__title">
            {title}
            {isEditable && (
              <span className="ml-2 text-xs text-[hsl(var(--muted-foreground))]">
                （編集）
              </span>
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
              {localRows.map((row, idx) => (
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
                    {isEditable ? (
                      <Input
                        type="number"
                        min={0}
                        step={1}
                        value={localRows[idx].stock}
                        onChange={(e) =>
                          handleChangeStock(idx, e.target.value)
                        }
                        className="ivc__stock-input w-20 text-right"
                        aria-label={`${row.modelCode} / ${row.size} / ${row.colorName} の生産数`}
                      />
                    ) : (
                      <span className="ivc__stock-number">
                        {row.stock}
                      </span>
                    )}
                  </TableCell>
                </TableRow>
              ))}

              {/* データなし表示 */}
              {localRows.length === 0 && (
                <TableRow>
                  <TableCell colSpan={4} className="ivc__empty">
                    表示できる生産数データがありません。
                  </TableCell>
                </TableRow>
              )}

              {/* 合計行 */}
              {localRows.length > 0 && (
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
