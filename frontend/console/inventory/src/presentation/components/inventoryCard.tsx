// frontend/console/inventory/src/presentation/components/inventoryCard.tsx
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

import type { InventoryRow } from "../../application/inventoryTypes";

import { rgbIntToHex } from "../../../../shell/src/shared/util/color";

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
  // ✅ displayOrder のみに従って昇順ソート
  // - displayOrder が無い行は最後
  // - displayOrder が同じ場合は「元の順番」を維持（副基準で並べ替えない）
  const sortedRows = React.useMemo(() => {
    const src = rows ?? [];

    const orderOf = (r: any): number => {
      const v = (r as any)?.displayOrder;
      const n = Number(v);
      return Number.isFinite(n) && n > 0 ? n : Number.POSITIVE_INFINITY;
    };

    // 安定ソート（同一 displayOrder は入力順維持）
    return src
      .map((r, idx) => ({ r, idx }))
      .sort((a, b) => {
        const oa = orderOf(a.r);
        const ob = orderOf(b.r);
        if (oa !== ob) return oa - ob;
        return a.idx - b.idx; // ✅ 元の順番を維持
      })
      .map((x) => x.r);
  }, [rows]);

  const totalStock = React.useMemo(
    () => sortedRows.reduce((sum, r) => sum + (r.stock || 0), 0),
    [sortedRows],
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
              {sortedRows.map((row, idx) => {
                const rgbHex = rgbIntToHex(row.rgb) ?? null;

                // row.rgb が "#RRGGBB" を直接持っている場合はそれを優先（互換維持）
                const bgColor =
                  typeof row.rgb === "string" && row.rgb.trim().startsWith("#")
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

              {sortedRows.length === 0 && (
                <TableRow>
                  {/* ✅ 列数が4になったので colSpan も 4 */}
                  <TableCell colSpan={4} className="ivc__empty">
                    表示できる在庫データがありません。
                  </TableCell>
                </TableRow>
              )}

              {sortedRows.length > 0 && (
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
