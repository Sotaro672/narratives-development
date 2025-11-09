// frontend/production/src/pages/productionQuantityCard.tsx
import * as React from "react";
import { BarChart3 } from "lucide-react";
import { Card, CardHeader, CardTitle, CardContent } from "../../../../shared/ui/card";
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from "../../../../shared/ui/table";
import { Input } from "../../../../shared/ui/input";
import "../styles/production.css";

export type QuantityCell = {
  size: string;   // 例: "S" | "M" | "L"
  color: string;  // 例: "ホワイト" | "ブラック" | "ネイビー"
  qty: number;    // 0 以上の整数
};

type ProductionQuantityCardProps = {
  /** 表示するサイズ（順序を保持） */
  sizes: string[];
  /** 表示するカラー（順序を保持） */
  colors: string[];
  /** (size, color) ごとの数量 */
  quantities: QuantityCell[];

  /** 編集モード or 閲覧モード（従来） */
  mode?: "view" | "edit";
  /** Figmaコード互換の編集フラグ（指定があればこちらを優先） */
  editable?: boolean;
  /** 変更ハンドラ（編集時のみ使用） */
  onChangeQty?: (size: string, color: string, qty: number) => void;

  className?: string;
};

/** 数量取得（存在しない組み合わせは 0 ） */
function getQuantityAtSizeColor(
  quantities: QuantityCell[],
  size: string,
  color: string
): number {
  const hit = quantities.find((q) => q.size === size && q.color === color);
  return hit ? hit.qty ?? 0 : 0;
}

/** 行合計（サイズごと） */
function calculateSizeTotal(
  quantities: QuantityCell[],
  size: string,
  colors: string[]
): number {
  return colors.reduce((acc, c) => acc + getQuantityAtSizeColor(quantities, size, c), 0);
}

/** 列合計（カラーごと） */
function calculateColorTotal(
  quantities: QuantityCell[],
  color: string,
  sizes: string[]
): number {
  return sizes.reduce((acc, s) => acc + getQuantityAtSizeColor(quantities, s, color), 0);
}

/** 総合計 */
function calculateGrandTotal(
  quantities: QuantityCell[],
  sizes: string[],
  colors: string[]
): number {
  return sizes.reduce(
    (sum, s) => sum + calculateSizeTotal(quantities, s, colors),
    0
  );
}

/** 無効値を除いたサイズ/カラー配列 */
function filterValidSizes(sizes: string[]): string[] {
  return (sizes ?? []).filter((s) => !!s && s.trim().length > 0);
}
function filterValidColors(colors: string[]): string[] {
  return (colors ?? []).filter((c) => !!c && c.trim().length > 0);
}

/** サイズ×カラーのどれか1つでも成立すれば true（行/列ヘッダが空でないことが肝） */
function hasValidSizeColorCombination(sizes: string[], colors: string[]): boolean {
  return filterValidSizes(sizes).length > 0 && filterValidColors(colors).length > 0;
}

export default function ProductionQuantityCard({
  sizes,
  colors,
  quantities,
  mode = "view",
  editable,
  onChangeQty,
  className,
}: ProductionQuantityCardProps) {
  // Figma互換: editable が指定されていればそれを優先、なければ mode で判断
  const isEditable = typeof editable === "boolean" ? editable : mode === "edit";

  const validSizes = React.useMemo(() => filterValidSizes(sizes), [sizes]);
  const validColors = React.useMemo(() => filterValidColors(colors), [colors]);
  const hasValid = React.useMemo(
    () => hasValidSizeColorCombination(validSizes, validColors),
    [validSizes, validColors]
  );

  // 行合計 / 列合計 / 総合計
  const rowSums = React.useMemo(
    () => validSizes.map((s) => calculateSizeTotal(quantities, s, validColors)),
    [validSizes, validColors, quantities]
  );
  const colSums = React.useMemo(
    () => validColors.map((c) => calculateColorTotal(quantities, c, validSizes)),
    [validColors, validSizes, quantities]
  );
  const grandTotal = React.useMemo(
    () => calculateGrandTotal(quantities, validSizes, validColors),
    [quantities, validSizes, validColors]
  );

  // 入力変更
  const handleQuantityChange = React.useCallback(
    (size: string, color: string, value: string) => {
      if (!isEditable || !onChangeQty) return;
      // 空文字は 0、負値/NaN は 0、少数は切り捨て
      const n = Math.max(0, Math.floor(Number(value || "0")));
      onChangeQty(size, color, Number.isFinite(n) ? n : 0);
    },
    [isEditable, onChangeQty]
  );

  return (
    <Card className={`mqc ${className ?? ""}`}>
      <CardHeader className="mqc__header">
        <div className="mqc__header-inner">
          <BarChart3 size={16} />
          <CardTitle className="mqc__title">
            生産数{!isEditable && <span className="ml-2 text-xs text-[hsl(var(--muted-foreground))]">（閲覧）</span>}
          </CardTitle>
        </div>
      </CardHeader>

      <CardContent className="mqc__body">
        {hasValid ? (
          <div className="overflow-x-auto">
            <Table className="mqc__table">
              <TableHeader>
                <TableRow>
                  <TableHead className="mqc__th mqc__th--left">サイズ / カラー</TableHead>
                  {validColors.map((color) => (
                    <TableHead key={color} className="mqc__th">{color}</TableHead>
                  ))}
                  <TableHead className="mqc__th">合計</TableHead>
                </TableRow>
              </TableHeader>

              <TableBody>
                {validSizes.map((s, rIdx) => (
                  <TableRow key={s}>
                    <TableCell className="mqc__size">{s}</TableCell>

                    {validColors.map((c) => {
                      const v = getQuantityAtSizeColor(quantities, s, c);
                      return (
                        <TableCell key={`${s}-${c}`} className="mqc__cell">
                          {isEditable ? (
                            <Input
                              type="number"
                              min={0}
                              step={1}
                              value={v}
                              onChange={(e) => handleQuantityChange(s, c, e.target.value)}
                              aria-label={`${s} / ${c} の数量`}
                              className="mqc__input w-16 text-center"
                            />
                          ) : (
                            v
                          )}
                        </TableCell>
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
          </div>
        ) : (
          <div className="text-center py-8 text-[hsl(var(--muted-foreground))]">
            <p>サイズとカラーを設定すると生産数を入力できます</p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
