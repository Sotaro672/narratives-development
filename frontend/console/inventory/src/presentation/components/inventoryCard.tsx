// frontend\console\inventory\src\presentation\components\inventoryCard.tsx
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

import "../styles/inventory.css";

// ----------------------------------------------------------
// RGB(int) → HEX (#RRGGBB)
// - InventoryRow.rgb は string | number | null どれでも来うる前提で安全に変換
// ----------------------------------------------------------
function rgbIntToHex(rgb: number | string | null | undefined): string | null {
  if (rgb === null || rgb === undefined) return null;
  const n = typeof rgb === "string" ? Number(rgb) : rgb;
  if (!Number.isFinite(n)) return null;

  const clamped = Math.max(0, Math.min(0xffffff, Math.floor(n)));
  const hex = clamped.toString(16).padStart(6, "0");
  return `#${hex}`;
}

export type InventoryRow = {
  /** トークン表示（例: "Token A" / "TB-xxxx" / tokenBlueprint name など） */
  // NOTE: 互換のため残す（画面では表示しない）
  token?: string;

  /** 型番 (例: "LM-SB-S-WHT") */
  modelNumber: string;

  /** サイズ (例: "S" | "M" | "L") */
  size: string;

  /** カラー表示名 (例: "ホワイト") */
  color: string;

  /**
   * RGB
   * - int(0xRRGGBB) で来ることもあるので、表示時に hex 化して dot に反映する
   */
  rgb?: number | string | null;

  /** 在庫数 */
  stock: number;
};

type InventoryCardProps = {
  title?: string;
  rows: InventoryRow[];
  className?: string;
  mode?: "view"; // 現状は閲覧専用
};

const InventoryCard: React.FC<InventoryCardProps> = ({
  title = "モデル別在庫一覧",
  rows,
  className,
  mode = "view",
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
            {mode !== "view" && (
              <span className="ml-2 text-xs text-[hsl(var(--muted-foreground))]">
                （{mode}）
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
                {/* ✅ トークン列を削除 */}
                <TableHead className="ivc__th ivc__th--left">型番</TableHead>
                <TableHead className="ivc__th">サイズ</TableHead>
                <TableHead className="ivc__th">カラー</TableHead>
                <TableHead className="ivc__th ivc__th--right">在庫数</TableHead>
              </TableRow>
            </TableHeader>

            <TableBody>
              {rows.map((row, idx) => {
                const rgbHex = rgbIntToHex(row.rgb);
                const bgColor =
                  row.rgb &&
                  typeof row.rgb === "string" &&
                  row.rgb.trim().startsWith("#")
                    ? row.rgb.trim()
                    : rgbHex ?? "#ffffff";

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
                        title={rgbHex ?? (typeof row.rgb === "string" ? row.rgb : "")}
                      />
                      <span className="ivc__color-label">{row.color}</span>
                    </TableCell>

                    {/* 在庫数 */}
                    <TableCell className="ivc__stock">
                      <span className="ivc__stock-number">{row.stock}</span>
                    </TableCell>
                  </TableRow>
                );
              })}

              {rows.length === 0 && (
                <TableRow>
                  {/* ✅ 列数が4になったので colSpan も 4 */}
                  <TableCell colSpan={4} className="ivc__empty">
                    表示できる在庫データがありません。
                  </TableCell>
                </TableRow>
              )}

              {rows.length > 0 && (
                <TableRow className="ivc__total-row">
                  {/* ✅ 先頭3列に「合計」、最後1列に数値 */}
                  <TableCell colSpan={3} className="ivc__total-label ivc__th--right">
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
