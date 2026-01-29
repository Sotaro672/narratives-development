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

// ✅ dto/detail.go を正: 表示カードは detail の行型を受け取る（create 側はアダプトして渡す）
import type { ProductionQuantityRow } from "../../application/detail/types";

type ProductionQuantityCardProps = {
  title?: string;
  rows: ProductionQuantityRow[];
  className?: string;
  mode?: "view" | "edit";
  onChangeRows?: (rows: ProductionQuantityRow[]) => void;
};

// サイズ並びの優先順位（未定義は末尾へ）
function sizeRank(size: string): number {
  const s = (size ?? "").trim().toLowerCase();

  // よくある表記ゆれを吸収
  const normalized =
    s === "xs" || s === "x-small" ? "xs" :
    s === "s" || s === "small" ? "s" :
    s === "m" || s === "medium" ? "m" :
    s === "l" || s === "large" ? "l" :
    s === "xl" || s === "x-large" ? "xl" :
    s === "xxl" || s === "2xl" ? "xxl" :
    s;

  const order: Record<string, number> = {
    xxs: 0,
    xs: 1,
    s: 2,
    m: 3,
    l: 4,
    xl: 5,
    xxl: 6,
    xxxl: 7,
    free: 99,
    one: 99,
    onesize: 99,
  };

  return order[normalized] ?? 1000; // 未定義は最後
}

function compareString(a: string, b: string): number {
  const aa = (a ?? "").trim();
  const bb = (b ?? "").trim();
  // 空は末尾へ
  if (!aa && !bb) return 0;
  if (!aa) return 1;
  if (!bb) return -1;
  return aa.localeCompare(bb, "ja");
}

const ProductionQuantityCard: React.FC<ProductionQuantityCardProps> = ({
  title = "モデル別生産数一覧",
  rows,
  className,
  mode = "view",
  onChangeRows,
}) => {
  const isEditable = mode === "edit";

  // ✅ 表示順を「色 → サイズ」に統一（安定ソート）
  // - 1st: color（文字列）
  // - 2nd: size（S/M/L... の順位）
  // - 3rd: modelNumber（安定化）
  // - 4th: modelId（最後の安定化）
  const sortedRows = React.useMemo(() => {
    const safe = Array.isArray(rows) ? rows : [];
    const copied = [...safe];

    copied.sort((a, b) => {
      // 1) color
      const c = compareString(a.color ?? "", b.color ?? "");
      if (c !== 0) return c;

      // 2) size
      const sa = sizeRank(a.size ?? "");
      const sb = sizeRank(b.size ?? "");
      if (sa !== sb) return sa - sb;

      // size が両方 unknown の場合は文字列で比較して安定化
      if (sa === 1000 && sb === 1000) {
        const s = compareString(a.size ?? "", b.size ?? "");
        if (s !== 0) return s;
      }

      // 3) modelNumber
      const mn = compareString(a.modelNumber ?? "", b.modelNumber ?? "");
      if (mn !== 0) return mn;

      // 4) modelId（最後の安定化）
      return compareString(a.modelId ?? "", b.modelId ?? "");
    });

    return copied;
  }, [rows]);

  const totalQuantity = React.useMemo(
    () => sortedRows.reduce((sum, r) => sum + (r.quantity || 0), 0),
    [sortedRows],
  );

  const handleChangeQuantity = React.useCallback(
    (index: number, value: string) => {
      if (!onChangeRows) return;

      const n = Math.max(0, Math.floor(Number(value || "0")));
      const safe = Number.isFinite(n) ? n : 0;

      // ✅ 並び替え後の行に対して編集するため、sortedRows をベースに更新して返す
      const next = sortedRows.map((row, i) =>
        i === index ? { ...row, quantity: safe } : row,
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
            {sortedRows.map((row, idx) => {
              const rgbHex = rgbIntToHex(row.rgb) ?? null;
              const bgColor = rgbHex ?? "#ffffff";

              return (
                <TableRow key={`${row.modelId}-${idx}`}>
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

            {sortedRows.length > 0 && (
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
