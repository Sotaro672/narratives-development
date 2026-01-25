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

import type { ProductionQuantityRow } from "../create/types";

// ✅ あなたが貼ったCSSファイルに合わせて import を修正してください。
// 例：同階層に配置するなら "./productionQuantityCard.css" など。
// ここでは、コンポーネントと同階層に置く想定のパス例を示します。
import "../styles/production.css";

// ----------------------------------------------------------
// RGB → HEX (#RRGGBB)
// ----------------------------------------------------------
function rgbIntToHex(rgb: number | string | null | undefined): string | null {
  if (rgb === null || rgb === undefined) return null;

  if (typeof rgb === "string") {
    const s = rgb.trim();

    // "#RRGGBB"
    if (/^#[0-9a-fA-F]{6}$/.test(s)) return s;

    // "RRGGBB" (hex without '#')
    if (/^[0-9a-fA-F]{6}$/.test(s)) return `#${s}`;

    // "0xRRGGBB"
    if (/^0x[0-9a-fA-F]{6}$/.test(s)) return `#${s.slice(2)}`;

    // 数値文字列（10進）
    const n = Number(s);
    if (!Number.isFinite(n)) return null;

    const clamped = Math.max(0, Math.min(0xffffff, Math.floor(n)));
    const hex = clamped.toString(16).padStart(6, "0");
    return `#${hex}`;
  }

  if (!Number.isFinite(rgb)) return null;

  const clamped = Math.max(0, Math.min(0xffffff, Math.floor(rgb)));
  const hex = clamped.toString(16).padStart(6, "0");
  return `#${hex}`;
}

type ProductionQuantityCardProps = {
  title?: string;
  rows: ProductionQuantityRow[];
  className?: string;
  mode?: "view" | "edit";
  onChangeRows?: (rows: ProductionQuantityRow[]) => void;
};

const ProductionQuantityCard: React.FC<ProductionQuantityCardProps> = ({
  title = "モデル別生産数一覧",
  rows,
  className,
  mode = "view",
  onChangeRows,
}) => {
  const isEditable = mode === "edit";

  const totalQuantity = React.useMemo(
    () => rows.reduce((sum, r) => sum + (r.quantity || 0), 0),
    [rows],
  );

  const handleChangeQuantity = React.useCallback(
    (index: number, value: string) => {
      if (!onChangeRows) return;

      const n = Math.max(0, Math.floor(Number(value || "0")));
      const safe = Number.isFinite(n) ? n : 0;

      const next = rows.map((row, i) =>
        i === index ? { ...row, quantity: safe } : row,
      );
      onChangeRows(next);
    },
    [rows, onChangeRows],
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
            {rows.map((row, idx) => {
              const rgbHex = rgbIntToHex(row.rgb);
              const bgColor = rgbHex ?? "#ffffff";

              return (
                <TableRow key={`${row.modelNumber}-${idx}`}>
                  <TableCell>{row.modelNumber}</TableCell>
                  <TableCell className="mqc__size">{row.size}</TableCell>

                  <TableCell>
                    <span className="mqc__color">
                      <span
                        className="mqc__color-dot"
                        style={{
                          backgroundColor: bgColor,
                        }}
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
                        onChange={(e) => handleChangeQuantity(idx, e.target.value)}
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

            {rows.length > 0 && (
              <TableRow className="mqc__footer-row">
                <TableCell colSpan={3} className="mqc__footer-label">
                  合計
                </TableCell>
                <TableCell className="mqc__footer-cell">
                  <span className="mqc__pill mqc__pill--total">{totalQuantity}</span>
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
