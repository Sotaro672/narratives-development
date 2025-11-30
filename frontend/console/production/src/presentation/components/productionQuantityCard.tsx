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

import type { ProductionQuantityRow } from "../../application/productionCreateService";

import "../styles/production.css";

type ProductionQuantityCardProps = {
  title?: string;
  rows: ProductionQuantityRow[];
  className?: string;

  /** "edit" なら数量入力可 / "view" なら閲覧のみ */
  mode?: "view" | "edit";

  /** 行全体を更新するためのコールバック */
  onChangeRows?: (rows: ProductionQuantityRow[]) => void;
};

/**
 * number / string な RGB (0xRRGGBB) を CSS 用 #RRGGBB に変換
 */
function rgbIntToHex(rgb: number | string | null | undefined): string | null {
  if (rgb === null || rgb === undefined) return null;
  const n = typeof rgb === "string" ? Number(rgb) : rgb;
  if (!Number.isFinite(n)) return null;

  const clamped = Math.max(0, Math.min(0xffffff, Math.floor(n)));
  const hex = clamped.toString(16).padStart(6, "0");
  return `#${hex}`;
}

/**
 * モデル別生産数一覧カード
 * inventoryCard と同じ見た目 + edit/view 切り替え
 */
const ProductionQuantityCard: React.FC<ProductionQuantityCardProps> = ({
  title = "モデル別生産数一覧",
  rows,
  className,
  mode = "view",
  onChangeRows,
}) => {
  const isEditable = mode === "edit";

  // 生産数合計を算出
  const totalQuantity = React.useMemo(
    () => rows.reduce((sum, r) => sum + (r.quantity || 0), 0),
    [rows],
  );

  const handleChangeQuantity = React.useCallback(
    (index: number, value: string) => {
      if (!onChangeRows) return;

      // 空文字は 0、負値/NaN は 0、少数は切り捨て
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
                <TableHead className="ivc__th ivc__th--right">
                  生産数
                </TableHead>
              </TableRow>
            </TableHeader>

            <TableBody>
              {rows.map((row, idx) => {
                // create / detail 双方で追加済みの rgb プロパティを拾う
                const rgb = (row as any).rgb as number | string | null | undefined;
                const rgbHex = rgbIntToHex(rgb);

                const bgColor =
                  row.colorCode && row.colorCode.trim()
                    ? row.colorCode
                    : rgbHex ?? "#ffffff";

                return (
                  <TableRow
                    key={`${row.modelCode}-${idx}`}
                    className="ivc__tr"
                  >
                    <TableCell className="ivc__model">
                      {row.modelCode}
                    </TableCell>
                    <TableCell className="ivc__size">{row.size}</TableCell>
                    <TableCell className="ivc__color-cell">
                      <span
                        className="ivc__color-dot"
                        style={{
                          backgroundColor: bgColor,
                          boxShadow: "0 0 0 1px rgba(0,0,0,0.18)",
                        }}
                        // 参考として title に HEX を入れておく
                        title={rgbHex ?? row.colorCode ?? ""}
                      />
                      <span className="ivc__color-label">
                        {row.colorName}
                      </span>
                    </TableCell>
                    <TableCell className="ivc__quantity">
                      {isEditable ? (
                        <Input
                          type="number"
                          min={0}
                          step={1}
                          value={row.quantity ?? 0}
                          onChange={(e) =>
                            handleChangeQuantity(idx, e.target.value)
                          }
                          className="ivc__quantity-input w-20 text-right"
                          aria-label={`${row.modelCode} の生産数`}
                        />
                      ) : (
                        <span className="ivc__quantity-number">{row.quantity}</span>
                      )}
                    </TableCell>
                  </TableRow>
                );
              })}

              {/* データなし表示 */}
              {rows.length === 0 && (
                <TableRow>
                  <TableCell colSpan={4} className="ivc__empty">
                    表示できる生産数データがありません。
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
