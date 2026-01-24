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

// ✅ Application ではなく Presentation の UI 型を参照する
import type { ProductionQuantityRow } from "../create/types";

import "../styles/production.css";

// ----------------------------------------------------------
// RGB → HEX (#RRGGBB)
// - number: 0xRRGGBB 相当（10進の数値として渡ってくる想定）
// - string: "#RRGGBB"（backend DTO 想定）または数値文字列（10進）
// ----------------------------------------------------------
function rgbIntToHex(rgb: number | string | null | undefined): string | null {
  if (rgb === null || rgb === undefined) return null;

  // string の場合: "#RRGGBB" をそのまま許容し、
  // それ以外は数値文字列として解釈する
  if (typeof rgb === "string") {
    const s = rgb.trim();

    // backend の想定: "#RRGGBB"
    if (/^#[0-9a-fA-F]{6}$/.test(s)) return s;

    // 数値文字列（10進）として解釈
    const n = Number(s);
    if (!Number.isFinite(n)) return null;

    const clamped = Math.max(0, Math.min(0xffffff, Math.floor(n)));
    const hex = clamped.toString(16).padStart(6, "0");
    return `#${hex}`;
  }

  // number の場合
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
              {rows.map((row, idx) => {
                const rgb = row.rgb;
                const rgbHex = rgbIntToHex(rgb);
                const bgColor = rgbHex ?? "#ffffff";

                return (
                  <TableRow key={`${row.modelNumber}-${idx}`} className="ivc__tr">
                    {/* 型番 */}
                    <TableCell className="ivc__model">{row.modelNumber}</TableCell>

                    {/* サイズ */}
                    <TableCell className="ivc__size">{row.size}</TableCell>

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
                      <span className="ivc__color-label">{row.color}</span>
                    </TableCell>

                    {/* 生産数 */}
                    <TableCell className="ivc__quantity">
                      {isEditable ? (
                        <Input
                          type="number"
                          min={0}
                          step={1}
                          value={row.quantity ?? 0}
                          onChange={(e) => handleChangeQuantity(idx, e.target.value)}
                          className="ivc__quantity-input w-20 text-right"
                          aria-label={`${row.modelNumber} の生産数`}
                        />
                      ) : (
                        <span className="ivc__quantity-number">{row.quantity}</span>
                      )}
                    </TableCell>
                  </TableRow>
                );
              })}

              {rows.length === 0 && (
                <TableRow>
                  <TableCell colSpan={4} className="ivc__empty">
                    表示できる生産数データがありません。
                  </TableCell>
                </TableRow>
              )}

              {rows.length > 0 && (
                <TableRow className="ivc__total-row">
                  <TableCell colSpan={3} className="ivc__total-label ivc__th--right">
                    合計
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

export default ProductionQuantityCard;
