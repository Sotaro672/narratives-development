// frontend/production/src/pages/productionQuantityCard.tsx
import * as React from "react";
import { BarChart2 } from "lucide-react";
import { Card, CardHeader, CardTitle, CardContent } from "../../../shared/ui/card";
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from "../../../shared/ui/table";
import "./productionQuantityCard.css";

export type QuantityCell = {
  size: string;   // 例: "S" | "M" | "L"
  color: string;  // 例: "ホワイト" | "ブラック" | "ネイビー"
  qty: number;    // 0 以上の整数
};

type ProductionQuantityCardProps = {
  /** 表示するサイズの並び順 */
  sizes: string[];
  /** 表示するカラーの並び順 */
  colors: string[];
  /** (size, color) ごとの数量 */
  quantities: QuantityCell[];
  className?: string;

  /** 表示モード（既定: "edit"） */
  mode?: "edit" | "view";
  /**
   * 数量変更ハンドラ（編集モード時のみ有効）
   * 未指定の場合、編集モードでも読み取り専用表示になります
   */
  onChangeQty?: (size: string, color: string, nextQty: number) => void;
};

export default function ProductionQuantityCard({
  sizes,
  colors,
  quantities,
  className,
  mode = "edit",
  onChangeQty,
}: ProductionQuantityCardProps) {
  const isEdit = mode === "edit";
  const canEdit = isEdit && typeof onChangeQty === "function";

  // Map を用意: size -> color -> qty
  const matrix: Record<string, Record<string, number>> = React.useMemo(() => {
    const m: Record<string, Record<string, number>> = {};
    for (const s of sizes) m[s] = {};
    for (const { size, color, qty } of quantities) {
      if (!m[size]) m[size] = {};
      m[size][color] = (typeof qty === "number" ? qty : 0) ?? 0;
    }
    return m;
  }, [sizes, quantities]);

  // 行合計・列合計・総合計
  const rowSums = React.useMemo(() => {
    return sizes.map((s) =>
      colors.reduce((acc, c) => acc + (matrix[s]?.[c] ?? 0), 0)
    );
  }, [sizes, colors, matrix]);

  const colSums = React.useMemo(() => {
    return colors.map((c) =>
      sizes.reduce((acc, s) => acc + (matrix[s]?.[c] ?? 0), 0)
    );
  }, [sizes, colors, matrix]);

  const grandTotal = React.useMemo(
    () => rowSums.reduce((a, b) => a + b, 0),
    [rowSums]
  );

  // 入力変更
  const handleChange =
    (size: string, color: string) =>
    (e: React.ChangeEvent<HTMLInputElement>) => {
      if (!canEdit) return;
      const raw = e.target.value;
      // 空文字は 0 と解釈、負値は 0 に丸め、少数は切り捨て
      const n = Math.max(0, Math.floor(Number(raw || "0")));
      onChangeQty!(size, color, Number.isFinite(n) ? n : 0);
    };

  return (
    <Card className={`mqc ${className ?? ""}`}>
      <CardHeader className="mqc__header">
        <div className="mqc__header-inner">
          <BarChart2 size={16} />
          <CardTitle className="mqc__title">
            生産数{mode === "view" && <span className="ml-2 text-xs text-[hsl(var(--muted-foreground))]">（閲覧）</span>}
          </CardTitle>
        </div>
      </CardHeader>

      <CardContent className="mqc__body">
        <Table className="mqc__table">
          <TableHeader>
            <TableRow>
              <TableHead className="mqc__th mqc__th--left">サイズ / カラー</TableHead>
              {colors.map((color) => (
                <TableHead key={color} className="mqc__th">{color}</TableHead>
              ))}
              <TableHead className="mqc__th">合計</TableHead>
            </TableRow>
          </TableHeader>

          <TableBody>
            {sizes.map((s, rIdx) => (
              <TableRow key={s}>
                <TableCell className="mqc__size">{s}</TableCell>

                {colors.map((c) => {
                  const v = matrix[s]?.[c] ?? 0;

                  // 編集モード + onChangeQty あり → number input
                  if (canEdit) {
                    return (
                      <TableCell key={`${s}-${c}`} className="mqc__cell">
                        <input
                          type="number"
                          min={0}
                          step={1}
                          value={v}
                          onChange={handleChange(s, c)}
                          aria-label={`${s} / ${c} の数量`}
                          className="mqc__input w-16 text-center border rounded-md px-2 py-1"
                        />
                      </TableCell>
                    );
                  }

                  // 閲覧 or 編集でも onChangeQty 未指定 → 値だけ表示
                  return (
                    <TableCell key={`${s}-${c}`} className="mqc__cell">{v}</TableCell>
                  );
                })}

                <TableCell className="mqc__sum">
                  <span className="mqc__pill">{rowSums[rIdx]}</span>
                </TableCell>
              </TableRow>
            ))}

            {/* フッター合計行 */}
            <TableRow className="mqc__footer-row">
              <TableCell className="mqc__footer-label">合計</TableCell>

              {colSums.map((sum, i) => (
                <TableCell key={`colsum-${i}`} className="mqc__footer-cell">
                  <span className="mqc__pill">{sum}</span>
                </TableCell>
              ))}

              <TableCell className="mqc__footer-cell">
                <span className="mqc__pill mqc__pill--total">{grandTotal}</span>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}
