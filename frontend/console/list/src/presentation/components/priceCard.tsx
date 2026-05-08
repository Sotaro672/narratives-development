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

// ✅ ロジックは hook に寄せる
import { usePriceCard } from "../hook/usePriceCard";

// ✅ 型は inventory/application を正とする（依存方向を正す）
import type { PriceCardProps } from "../../../../inventory/src/application/listCreate/priceCard.types";

const PriceCard: React.FC<PriceCardProps> = (props) => {
  const { className } = props;

  const {
    title,
    mode,
    isEdit,
    showModeBadge,
    currencySymbol,
    rowsVM,
    isEmpty,
  } = usePriceCard(props);

  return (
    <Card className={`prc ${className ?? ""}`}>
      <CardHeader className="prc__header">
        <div className="prc__header-inner flex items-center gap-2">
          <Tag size={18} />
          <CardTitle className="prc__title">
            {title}
            {showModeBadge && (
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
              {rowsVM.map((row) => {
                return (
                  // ✅ React key は識別子 modelId を使う（displayOrder は重複/未設定があり得る）
                  <TableRow key={row.modelId} className="prc__tr">
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
                          backgroundColor: row.bgColor,
                          boxShadow: "0 0 0 1px rgba(0,0,0,0.18)",
                        }}
                        title={row.rgbTitle}
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
                            value={row.priceInputValue}
                            placeholder="-"
                            onChange={row.onChangePriceInput}
                          />
                        </div>
                      ) : (
                        <span className="prc__price-value">{row.priceDisplayText}</span>
                      )}
                    </TableCell>
                  </TableRow>
                );
              })}

              {isEmpty && (
                <TableRow>
                  <TableCell colSpan={4} className="prc__empty">
                    表示できるデータがありません。
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
