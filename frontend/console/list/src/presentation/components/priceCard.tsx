// frontend/console/list/src/presentation/components/priceCard.tsx
import * as React from "react";
import { Tag } from "lucide-react";
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

// ----------------------------------------------------------
// RGB(int) → HEX (#RRGGBB)
// - row.rgb は string | number | null どれでも来うる前提で安全に変換
// ----------------------------------------------------------
function rgbIntToHex(rgb: number | string | null | undefined): string | null {
  if (rgb === null || rgb === undefined) return null;
  const n = typeof rgb === "string" ? Number(rgb) : rgb;
  if (!Number.isFinite(n)) return null;

  const clamped = Math.max(0, Math.min(0xffffff, Math.floor(n)));
  const hex = clamped.toString(16).padStart(6, "0");
  return `#${hex}`;
}

export type PriceRow = {
  /** サイズ (例: "S" | "M" | "L") */
  size: string;

  /** カラー表示名 (例: "ホワイト") */
  color: string;

  /**
   * RGB
   * - int(0xRRGGBB) で来ることもあるので、表示時に hex 化して dot に反映する
   * - "#RRGGBB" の string が来ても許容
   */
  rgb?: number | string | null;

  /** 在庫数 */
  stock: number;

  /** ✅ 価格（円など。UIは数値入力） */
  price?: number | null;
};

type PriceCardProps = {
  title?: string;
  rows: PriceRow[];
  className?: string;

  /** view / edit */
  mode?: "view" | "edit";

  /**
   * edit 時に価格を更新するコールバック
   * - 親が rows を state 管理している前提
   */
  onChangePrice?: (index: number, price: number | null, row: PriceRow) => void;

  /** 表示用（例: "¥" / "$"）。未指定なら空 */
  currencySymbol?: string;

  /** 合計行の表示（デフォルト true） */
  showTotal?: boolean;
};

const PriceCard: React.FC<PriceCardProps> = ({
  title = "価格設定",
  rows,
  className,
  mode = "view",
  onChangePrice,
  currencySymbol = "¥",
  showTotal = true,
}) => {
  const isEdit = mode === "edit";

  const totalStock = React.useMemo(
    () => rows.reduce((sum, r) => sum + (r.stock || 0), 0),
    [rows],
  );

  const totalPrice = React.useMemo(
    () => rows.reduce((sum, r) => sum + (Number(r.price) || 0), 0),
    [rows],
  );

  return (
    <Card className={`prc ${className ?? ""}`}>
      <CardHeader className="prc__header">
        <div className="prc__header-inner flex items-center gap-2">
          <Tag size={18} />
          <CardTitle className="prc__title">
            {title}
            {mode !== "view" && (
              <span className="ml-2 text-xs text-[hsl(var(--muted-foreground))]">
                （{mode}）
              </span>
            )}
          </CardTitle>
        </div>
      </CardHeader>

      <CardContent className="prc__body">
        <div className="prc__table-wrap">
          <Table className="prc__table">
            <TableHeader>
              <TableRow>
                {/* ✅ 型番列は無し */}
                <TableHead className="prc__th">サイズ</TableHead>
                <TableHead className="prc__th">カラー</TableHead>
                <TableHead className="prc__th prc__th--right">在庫数</TableHead>
                <TableHead className="prc__th prc__th--right">価格</TableHead>
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

                const priceValue =
                  row.price === null || row.price === undefined
                    ? ""
                    : String(row.price);

                return (
                  <TableRow
                    key={`${String(row.size)}-${String(row.color)}-${idx}`}
                    className="prc__tr"
                  >
                    {/* サイズ */}
                    <TableCell className="prc__size">{row.size}</TableCell>

                    {/* カラー */}
                    <TableCell className="prc__color-cell">
                      <span
                        className="prc__color-dot inline-block align-middle mr-2"
                        style={{
                          width: 12,
                          height: 12,
                          borderRadius: 9999,
                          backgroundColor: bgColor,
                          boxShadow: "0 0 0 1px rgba(0,0,0,0.18)",
                        }}
                        title={rgbHex ?? (typeof row.rgb === "string" ? row.rgb : "")}
                      />
                      <span className="prc__color-label">{row.color}</span>
                    </TableCell>

                    {/* 在庫数 */}
                    <TableCell className="prc__stock text-right">
                      <span className="prc__stock-number">{row.stock}</span>
                    </TableCell>

                    {/* 価格 */}
                    <TableCell className="prc__price text-right">
                      {isEdit ? (
                        <div className="flex items-center gap-2 justify-end">
                          {currencySymbol ? (
                            <span className="text-xs text-[hsl(var(--muted-foreground))]">
                              {currencySymbol}
                            </span>
                          ) : null}
                          <Input
                            inputMode="numeric"
                            type="number"
                            min={0}
                            step={1}
                            className="h-8 w-32 text-right"
                            value={priceValue}
                            placeholder="-"
                            onChange={(e) => {
                              const v = e.target.value;
                              const next =
                                v.trim() === ""
                                  ? null
                                  : Number.isFinite(Number(v))
                                    ? Number(v)
                                    : null;

                              onChangePrice?.(idx, next, row);
                            }}
                          />
                        </div>
                      ) : (
                        <span className="prc__price-value">
                          {row.price === null || row.price === undefined
                            ? "-"
                            : `${currencySymbol ?? ""}${row.price}`}
                        </span>
                      )}
                    </TableCell>
                  </TableRow>
                );
              })}

              {rows.length === 0 && (
                <TableRow>
                  <TableCell colSpan={4} className="prc__empty">
                    表示できるデータがありません。
                  </TableCell>
                </TableRow>
              )}

              {showTotal && rows.length > 0 && (
                <TableRow className="prc__total-row">
                  <TableCell colSpan={2} className="prc__total-label text-right">
                    合計
                  </TableCell>
                  <TableCell className="prc__total-value text-right">
                    <strong>{totalStock}</strong>
                  </TableCell>
                  <TableCell className="prc__total-value text-right">
                    <strong>
                      {currencySymbol ?? ""}
                      {totalPrice}
                    </strong>
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

export default PriceCard;
