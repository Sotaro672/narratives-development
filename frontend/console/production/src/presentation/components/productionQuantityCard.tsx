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
import { rgbIntToHex } from "../../../../shell/src/shared/util/color";

// ✅ ViewModel を正として受け取る
import type { ProductionQuantityRowVM } from "../viewModels/productionQuantityRowVM";

type ProductionQuantityCardProps = {
  title?: string;
  rows: ProductionQuantityRowVM[];
  className?: string;
  mode?: "view" | "edit";
  onChangeRows?: (rows: ProductionQuantityRowVM[]) => void;
};

function displayOrderRank(v: unknown): number {
  return Number.isFinite(v as number)
    ? (v as number)
    : Number.POSITIVE_INFINITY; // 未設定は末尾
}

const ProductionQuantityCard: React.FC<ProductionQuantityCardProps> = ({
  title = "モデル別生産数一覧",
  rows,
  className,
  mode = "view",
  onChangeRows,
}) => {
  const isEditable = mode === "edit";

  // ✅ displayOrder のみに従って並べる
  const sortedRows = React.useMemo(() => {
    const safe = Array.isArray(rows) ? rows : [];
    const copied = [...safe];

    copied.sort((a, b) => {
      const da = displayOrderRank(a.displayOrder);
      const db = displayOrderRank(b.displayOrder);
      return da - db;
    });

    return copied;
  }, [rows]);

  const totalQuantity = React.useMemo(
    () => sortedRows.reduce((sum, r) => sum + (r.quantity || 0), 0),
    [sortedRows],
  );

  const handleChangeQuantity = React.useCallback(
    (id: string, value: string) => {
      if (!onChangeRows) return;

      const n = Math.max(0, Math.floor(Number(value || "0")));
      const safe = Number.isFinite(n) ? n : 0;

      // ✅ 並び替えに影響されないよう id で更新
      const next = sortedRows.map((row) =>
        row.id === id ? { ...row, quantity: safe } : row,
      );

      onChangeRows(next);
    },
    [sortedRows, onChangeRows],
  );

  return (
    <Card className={`mqc ${className ?? ""}`}>
      <CardHeader className="mqc__header">
        <div className="mqc__header-inner">
          <Palette size={18} />
          <CardTitle className="mqc__title">{title}</CardTitle>
        </div>
      </CardHeader>

      <CardContent className="mqc__body">
        <Table className="mqc__table">
          <TableHeader>
            <TableRow>
              <TableHead className="mqc__th mqc__th--left">型番</TableHead>
              <TableHead className="mqc__th">サイズ</TableHead>
              <TableHead className="mqc__th">カラー</TableHead>
              <TableHead className="mqc__th mqc__cell">生産数</TableHead>
            </TableRow>
          </TableHeader>

          <TableBody>
            {sortedRows.map((row) => {
              const rgbHex = rgbIntToHex(row.rgb) ?? null;
              const bgColor = rgbHex ?? "#ffffff";

              return (
                <TableRow key={row.id}>
                  <TableCell>{row.modelNumber}</TableCell>
                  <TableCell className="mqc__size">{row.size}</TableCell>

                  <TableCell>
                    <span className="mqc__color">
                      <span
                        className="mqc__color-dot"
                        style={{ backgroundColor: bgColor }}
                        title={rgbHex ?? ""}
                      />
                      <span>{row.color}</span>
                    </span>
                  </TableCell>

                  <TableCell className="mqc__cell">
                    {isEditable ? (
                      <Input
                        type="number"
                        min={0}
                        step={1}
                        value={row.quantity ?? 0}
                        onChange={(e) =>
                          handleChangeQuantity(row.id, e.target.value)
                        }
                        className="mqc__input"
                        aria-label={`${row.modelNumber} の生産数`}
                      />
                    ) : (
                      <span>{row.quantity}</span>
                    )}
                  </TableCell>
                </TableRow>
              );
            })}

            {sortedRows.length > 0 && (
              <TableRow className="mqc__footer-row">
                <TableCell colSpan={3} className="mqc__footer-label">
                  合計
                </TableCell>
                <TableCell className="mqc__footer-cell">
                  <span className="mqc__pill mqc__pill--total">
                    {totalQuantity}
                  </span>
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
};

export default ProductionQuantityCard;
